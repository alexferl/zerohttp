package zerohttp

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// Bind is the default binder instance used by the package
var Bind Binder = &defaultBinder{}

// B is a short alias for Bind for convenience
var B = Bind

// Binder handles request binding and parsing for various content types.
// It provides methods to decode request data into Go structs.
type Binder interface {
	// JSON decodes JSON request body into the destination struct.
	// It uses json.NewDecoder with DisallowUnknownFields enabled
	// for safer JSON parsing that rejects unknown fields.
	JSON(r io.Reader, dst any) error

	// Form parses form data from the request body (application/x-www-form-urlencoded)
	// and binds it to the destination struct using `form` tags.
	// It also parses the query string and includes those values.
	Form(r *http.Request, dst any) error

	// MultipartForm parses multipart/form-data from the request,
	// including file uploads, and binds values to the destination struct.
	// The maxMemory parameter controls how much of the form data is stored in memory
	// before being written to temp files (similar to http.Request.ParseMultipartForm).
	// File uploads are bound to fields of type FileHeader or []FileHeader.
	MultipartForm(r *http.Request, dst any, maxMemory int64) error

	// Query binds query parameters from the request URL to a destination struct.
	// Uses `query` struct tags for field mapping. Fields without tags are mapped
	// using snake_case conversion of the field name.
	// Returns an error if binding fails due to type mismatch.
	Query(r *http.Request, dst any) error
}

// Ensure defaultBinder implements Binder
var _ Binder = (*defaultBinder)(nil)

// defaultBinder implements the Binder interface with standard decoding
type defaultBinder struct{}

// JSON decodes JSON request body into the destination struct.
// It configures the decoder to disallow unknown fields for stricter validation.
// Returns an error if the JSON is malformed or contains unknown fields.
func (b *defaultBinder) JSON(r io.Reader, dst any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

// FileHeader represents an uploaded file in a multipart form.
// It provides access to the file's metadata and content.
type FileHeader struct {
	Filename string
	Size     int64
	Header   map[string][]string
	file     multipart.File // internal reference to the open file
}

// Open opens the uploaded file for reading.
// The caller is responsible for closing the file.
// Returns an error if the file cannot be opened or has already been closed.
func (fh *FileHeader) Open() (multipart.File, error) {
	if fh.file == nil {
		return nil, fmt.Errorf("file %s: no longer available (already processed or closed)", fh.Filename)
	}
	return fh.file, nil
}

// ReadAll reads the entire file content into a byte slice.
// This is a convenience method that opens, reads, and closes the file.
// Note: This loads the entire file into memory; use Open() for large files.
func (fh *FileHeader) ReadAll() (data []byte, err error) {
	file, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	return io.ReadAll(file)
}

// Form binds form data from a url.Values to a destination struct.
func (b *defaultBinder) Form(r *http.Request, dst any) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parse form: %w", err)
	}
	return bindValues(r.Form, dst, "form", false)
}

// MultipartForm parses multipart form data including file uploads.
func (b *defaultBinder) MultipartForm(r *http.Request, dst any, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return fmt.Errorf("parse multipart form: %w", err)
	}

	if r.MultipartForm == nil {
		return fmt.Errorf("no multipart form data")
	}

	// Bind form values first
	if err := bindValues(r.MultipartForm.Value, dst, "form", true); err != nil {
		return err
	}

	// Bind files to struct fields
	return BindMultipartFormFiles(r, dst)
}

// Query binds query parameters from the request URL to a destination struct.
// Uses `query` struct tags for field mapping. Fields without tags are mapped
// using snake_case conversion of the field name.
// Returns an error if binding fails due to type mismatch.
func (b *defaultBinder) Query(r *http.Request, dst any) error {
	return bindValues(r.URL.Query(), dst, "query", false)
}

// bindValues binds url.Values to a struct using the specified tag name.
// The tagName parameter specifies which struct tag to use (e.g., "form", "query").
// Uses the type registry to cache reflection information for improved performance.
func bindValues(values url.Values, dst any, tagName string, allowFiles bool) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to a struct")
	}

	t := v.Type()

	// Get cached type information from the registry
	info, err := globalTypeRegistry.getTypeInfo(t)
	if err != nil {
		return err
	}

	// Get pre-computed bindable fields for this tag
	bindableFields := info.getBindableFields(tagName, allowFiles)

	// Bind each field using the pre-computed path
	for _, bf := range bindableFields {
		// Get the field value using the cached path
		field := v.FieldByIndex(bf.path.toSlice())
		if !field.IsValid() {
			continue // Skip invalid fields (shouldn't happen with correct paths)
		}

		// Get value(s) from url.Values
		fieldValues, exists := values[bf.tag]
		if !exists || len(fieldValues) == 0 {
			continue
		}

		if err := setFieldValue(field, fieldValues); err != nil {
			return fmt.Errorf("field %s: %w", bf.name, err)
		}
	}

	return nil
}

// setFieldValue sets a field's value from form string values.
func setFieldValue(field reflect.Value, values []string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(values[0])

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", values[0])
		}
		field.SetInt(val)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer: %s", values[0])
		}
		field.SetUint(val)

	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(values[0], 64)
		if err != nil {
			return fmt.Errorf("invalid float: %s", values[0])
		}
		field.SetFloat(val)

	case reflect.Bool:
		val, err := strconv.ParseBool(values[0])
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", values[0])
		}
		field.SetBool(val)

	case reflect.Slice:
		return setSliceValue(field, values)

	case reflect.Ptr:
		// If value is empty, leave pointer as nil (optional parameter)
		if values[0] == "" {
			return nil
		}
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValue(field.Elem(), values)

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// setSliceValue sets a slice field's value.
func setSliceValue(field reflect.Value, values []string) error {
	slice := reflect.MakeSlice(field.Type(), len(values), len(values))

	for i, v := range values {
		elem := slice.Index(i)
		if err := setFieldValue(elem, []string{v}); err != nil {
			return fmt.Errorf("slice element %d: %w", i, err)
		}
	}

	field.Set(slice)
	return nil
}

// camelToSnake converts CamelCase to snake_case.
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// BindMultipartFormFiles binds file uploads to struct fields after the main form binding.
// This is called internally by handler logic when processing multipart forms.
// Exported for advanced use cases.
func BindMultipartFormFiles(r *http.Request, dst any) error {
	if r.MultipartForm == nil {
		return nil
	}

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to a struct")
	}

	return bindFilesToStruct(v, r.MultipartForm.File)
}

// bindFilesToStruct binds multipart files to struct fields using cached type information.
func bindFilesToStruct(v reflect.Value, files map[string][]*multipart.FileHeader) error {
	t := v.Type()

	// Get cached type information from the registry
	info, err := globalTypeRegistry.getTypeInfo(t)
	if err != nil {
		return err
	}

	// Use pre-computed file bindable fields
	for _, fbf := range info.fileBindableFields {
		// Check for files with this field tag
		fileHeaders, exists := files[fbf.tag]
		if !exists {
			continue
		}

		// Get the field using the cached path
		field := v.FieldByIndex(fbf.path.toSlice())
		if !field.IsValid() {
			continue
		}

		// Bind based on field type (single file or slice)
		if fbf.isSlice {
			// Slice of files
			fileList := make([]*FileHeader, len(fileHeaders))
			for j, fh := range fileHeaders {
				file, err := fh.Open()
				if err != nil {
					return fmt.Errorf("open file %s: %w", fh.Filename, err)
				}
				fileList[j] = &FileHeader{
					Filename: fh.Filename,
					Size:     fh.Size,
					Header:   fh.Header,
					file:     file,
				}
			}
			field.Set(reflect.ValueOf(fileList))
		} else {
			// Single file
			if len(fileHeaders) > 0 {
				fh := fileHeaders[0]
				file, err := fh.Open()
				if err != nil {
					return fmt.Errorf("open file %s: %w", fh.Filename, err)
				}
				field.Set(reflect.ValueOf(&FileHeader{
					Filename: fh.Filename,
					Size:     fh.Size,
					Header:   fh.Header,
					file:     file,
				}))
			}
		}
	}

	return nil
}
