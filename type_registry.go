package zerohttp

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Pre-computed types for efficient comparison
var (
	fileHeaderType      = reflect.TypeOf(FileHeader{})
	fileHeaderPtrType   = reflect.TypeOf(&FileHeader{})
	fileHeaderSliceType = reflect.TypeOf([]*FileHeader{})
)

// fieldPath represents the path to a field, including through embedded structs.
// For a top-level field, path is []int{index}.
// For an embedded field, path is []int{embeddedIndex, fieldIndex}.
// Uses a fixed-size array instead of slice to enforce immutability by value.
// The len field indicates how many indices are valid (up to 4 levels deep).
type fieldPath struct {
	indices [4]int
	len     int
}

// toSlice returns the valid portion of the path as a slice for FieldByIndex.
// This creates a new slice header but shares the underlying array (safe because
// the array is fixed-size and value-copied).
func (fp fieldPath) toSlice() []int {
	return fp.indices[:fp.len]
}

// append creates a new fieldPath with an additional index.
// The original path is unchanged (immutability by value).
func (fp fieldPath) append(idx int) fieldPath {
	newPath := fieldPath{len: fp.len + 1}
	copy(newPath.indices[:], fp.indices[:fp.len])
	newPath.indices[fp.len] = idx
	return newPath
}

// single creates a new fieldPath with a single index.
func singleFieldPath(idx int) fieldPath {
	return fieldPath{indices: [4]int{idx}, len: 1}
}

// fieldInfo stores cached reflection information for a single struct field.
type fieldInfo struct {
	index      int
	name       string
	fieldType  reflect.StructField
	isEmbedded bool
	canSet     bool
}

// bindableField represents a field that can be bound from form/query values.
type bindableField struct {
	path fieldPath
	name string
	tag  string
}

// fileBindableField represents a field that can be bound from multipart file uploads.
type fileBindableField struct {
	path    fieldPath
	tag     string
	isSlice bool // true for []*FileHeader, false for *FileHeader
}

// typeInfo stores cached reflection information for a struct type.
// All bindable fields are pre-computed during analysis to avoid lazy writes.
type typeInfo struct {
	typ    reflect.Type
	fields []fieldInfo
	// Pre-computed bindable fields for all supported tag/allowFiles combinations.
	// Populated once during analyzeType - read-only after that.
	formBindableFields  []bindableField // tag="form", allowFiles=false
	formWithFilesFields []bindableField // tag="form", allowFiles=true
	queryBindableFields []bindableField // tag="query", allowFiles=false
	fileBindableFields  []fileBindableField
}

// typeRegistry caches reflection information for struct types using sync.Map.
// This avoids repeated reflection overhead during request binding.
type typeRegistry struct {
	cache sync.Map // map[reflect.Type]*typeInfo
}

// globalTypeRegistry is the package-level type registry instance.
var globalTypeRegistry = &typeRegistry{}

// getTypeInfo retrieves cached type information for the given type.
// If the type hasn't been analyzed yet, it analyzes and caches it.
func (tr *typeRegistry) getTypeInfo(t reflect.Type) (*typeInfo, error) {
	// Fast path: sync.Map Load
	if info, ok := tr.cache.Load(t); ok {
		return info.(*typeInfo), nil
	}

	// Slow path: create type info
	info, err := tr.analyzeType(t)
	if err != nil {
		return nil, err
	}

	// Store in cache (if another goroutine stored first, use that)
	if existing, loaded := tr.cache.LoadOrStore(t, info); loaded {
		return existing.(*typeInfo), nil
	}
	return info, nil
}

// analyzeType analyzes a struct type and extracts all field information.
// All bindable field slices are pre-computed here to avoid data races.
func (tr *typeRegistry) analyzeType(t reflect.Type) (*typeInfo, error) {
	info := &typeInfo{
		typ: t,
	}

	// First pass: collect field metadata
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		fi := fieldInfo{
			index:      i,
			name:       field.Name,
			fieldType:  field,
			isEmbedded: field.Anonymous && field.Type.Kind() == reflect.Struct,
			canSet:     field.IsExported(),
		}

		info.fields = append(info.fields, fi)
	}

	// Second pass: pre-compute all bindable field combinations
	// This happens during construction, before the typeInfo is visible to other goroutines.
	info.formBindableFields = info.computeBindableFields("form", false)
	info.formWithFilesFields = info.computeBindableFields("form", true)
	info.queryBindableFields = info.computeBindableFields("query", false)

	fileFields, err := info.computeFileBindableFields()
	if err != nil {
		return nil, err
	}
	info.fileBindableFields = fileFields

	return info, nil
}

// getBindableFields returns all bindable fields for a given tag name.
// Uses pre-computed slices - no lazy initialization.
func (ti *typeInfo) getBindableFields(tagName string, allowFiles bool) []bindableField {
	switch tagName {
	case "form":
		if allowFiles {
			return ti.formWithFilesFields
		}
		return ti.formBindableFields
	case "query":
		return ti.queryBindableFields
	default:
		// Fallback for unknown tags - compute on the fly (shouldn't happen in practice)
		return ti.computeBindableFields(tagName, allowFiles)
	}
}

// computeBindableFields computes bindable fields for a tag/allowFiles combination.
func (ti *typeInfo) computeBindableFields(tagName string, allowFiles bool) []bindableField {
	result := make([]bindableField, 0, len(ti.fields))
	for i := range ti.fields {
		result = ti.collectFieldsRecursive(singleFieldPath(i), tagName, allowFiles, result)
	}
	return result
}

// computeFileBindableFields computes file-bindable fields.
func (ti *typeInfo) computeFileBindableFields() ([]fileBindableField, error) {
	result := make([]fileBindableField, 0)
	for i := range ti.fields {
		var err error
		result, err = ti.collectFileFieldsRecursive(singleFieldPath(i), result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// getTagValue returns the tag value for a specific tag name.
// No per-field caching - tag resolution is fast enough.
func (fi *fieldInfo) getTagValue(tagName string) string {
	return fi.fieldType.Tag.Get(tagName)
}

// collectFieldsRecursive recursively collects fields, expanding embedded structs.
// The path accumulates field indices for proper v.FieldByIndex access.
// Each index in the path is relative to its parent struct type.
func (ti *typeInfo) collectFieldsRecursive(path fieldPath, tagName string, allowFiles bool, result []bindableField) []bindableField {
	fieldIdx := path.indices[path.len-1]
	fi := &ti.fields[fieldIdx]

	// Handle embedded structs recursively BEFORE other checks
	// This allows embedded structs to be traversed even if they have tags.
	// Note: We recurse into embedded structs even if they're unexported
	// because their exported fields are still accessible via reflection.
	if fi.isEmbedded {
		embeddedType := fi.fieldType.Type
		embeddedInfo, _ := globalTypeRegistry.getTypeInfo(embeddedType)
		for j := range embeddedInfo.fields {
			// Build path relative to the TOP-LEVEL struct:
			// []int{topLevelIdx, embeddedFieldIdx, ...}
			newPath := path.append(j)
			result = embeddedInfo.collectFieldsRecursive(newPath, tagName, allowFiles, result)
		}
		return result
	}

	// Skip unexported fields (for non-embedded fields)
	if !fi.canSet {
		return result
	}

	// Consolidate tag lookup: single Tag.Get call per field
	tag := fi.fieldType.Tag.Get(tagName)
	if tag == "-" {
		return result
	}

	// Strip tag options (e.g., "name,omitempty" -> "name")
	if idx := strings.Index(tag, ","); idx != -1 {
		tag = tag[:idx]
	}

	// Check for file fields first (before pointer-to-struct guard)
	fieldType := fi.fieldType.Type
	isFileField := fieldType == fileHeaderPtrType || fieldType == fileHeaderSliceType

	// Skip file fields when not allowing files
	if !allowFiles && isFileField {
		return result
	}

	// Skip pointer-to-struct fields (except file fields) — they are not bindable
	// primitives and pointer embeds are not expanded (Kind() == Ptr bypassed isEmbedded check).
	if !isFileField && fieldType.Kind() == reflect.Ptr &&
		fieldType.Elem().Kind() == reflect.Struct {
		return result
	}

	// Use field name as default if tag is empty
	if tag == "" {
		tag = camelToSnake(fi.name)
	}

	bf := bindableField{
		path: path,
		name: fi.name,
		tag:  tag,
	}

	result = append(result, bf)
	return result
}

// collectFileFieldsRecursive recursively collects file-bindable fields.
// The path accumulates field indices for proper v.FieldByIndex access.
func (ti *typeInfo) collectFileFieldsRecursive(path fieldPath, result []fileBindableField) ([]fileBindableField, error) {
	fieldIdx := path.indices[path.len-1]
	fi := &ti.fields[fieldIdx]

	// Handle embedded structs recursively
	// Note: We recurse into embedded structs even if they're unexported
	// because their exported fields are still accessible via reflection.
	if fi.isEmbedded {
		embeddedType := fi.fieldType.Type
		embeddedInfo, _ := globalTypeRegistry.getTypeInfo(embeddedType)
		for j := range embeddedInfo.fields {
			newPath := path.append(j)
			var err error
			result, err = embeddedInfo.collectFileFieldsRecursive(newPath, result)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}

	// Skip unexported fields
	if !fi.canSet {
		return result, nil
	}

	// Check if field is a file header type (before pointer-to-struct guard)
	fieldType := fi.fieldType.Type
	isFileField := false
	isSlice := false

	switch fieldType {
	case fileHeaderPtrType:
		isFileField = true
		isSlice = false
	case fileHeaderSliceType:
		isFileField = true
		isSlice = true
	case fileHeaderType:
		// Non-pointer FileHeader is a configuration error
		return nil, fmt.Errorf("field %s: FileHeader must be a pointer (*FileHeader)", fi.name)
	}

	if !isFileField {
		return result, nil
	}

	// Get form tag (file binding only uses "form" tag)
	tag := fi.getTagValue("form")
	if tag == "-" {
		return result, nil
	}
	if tag == "" {
		tag = camelToSnake(fi.name)
	}

	bff := fileBindableField{
		path:    path,
		tag:     tag,
		isSlice: isSlice,
	}

	result = append(result, bff)
	return result, nil
}
