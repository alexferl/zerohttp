package zerohttp

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestBinder_JSON(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid JSON",
			json:      `{"name": "John", "age": 30}`,
			wantError: false,
		},
		{
			name:      "invalid JSON",
			json:      `{"invalid": json}`,
			wantError: true,
		},
		{
			name:      "unknown field",
			json:      `{"name": "John", "unknown": "field"}`,
			wantError: true,
			errorMsg:  "unknown field",
		},
		{
			name:      "empty JSON object",
			json:      `{}`,
			wantError: false,
		},
		{
			name:      "null values",
			json:      `{"name": null, "age": null}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.json)

			var result struct {
				Name *string `json:"name"`
				Age  *int    `json:"age"`
			}

			err := B.JSON(reader, &result)

			if tt.wantError {
				zhtest.AssertError(t, err)
				if tt.errorMsg != "" {
					zhtest.AssertErrorContains(t, err, tt.errorMsg)
				}
				return
			}

			zhtest.AssertNoError(t, err)

			// Verify valid cases
			if tt.name == "valid JSON" {
				zhtest.AssertNotNil(t, result.Name)
				zhtest.AssertEqual(t, "John", *result.Name)
				zhtest.AssertNotNil(t, result.Age)
				zhtest.AssertEqual(t, 30, *result.Age)
			}
		})
	}
}

func TestBinder_Form(t *testing.T) {
	tests := []struct {
		name      string
		formData  url.Values
		wantError bool
		expected  func(t *testing.T, result *testFormStruct)
	}{
		{
			name: "basic fields",
			formData: url.Values{
				"name":  []string{"John"},
				"email": []string{"john@example.com"},
			},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				zhtest.AssertEqual(t, "John", result.Name)
				zhtest.AssertEqual(t, "john@example.com", result.Email)
			},
		},
		{
			name: "numeric fields",
			formData: url.Values{
				"age":    []string{"30"},
				"score":  []string{"95.5"},
				"count":  []string{"100"},
				"active": []string{"true"},
			},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				zhtest.AssertEqual(t, 30, result.Age)
				zhtest.AssertEqual(t, 95.5, result.Score)
				zhtest.AssertEqual(t, uint(100), result.Count)
				zhtest.AssertTrue(t, result.Active)
			},
		},
		{
			name: "slice fields",
			formData: url.Values{
				"tags": []string{"go", "web", "api"},
			},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				expected := []string{"go", "web", "api"}
				zhtest.AssertLen(t, result.Tags, len(expected))
				for i, tag := range result.Tags {
					zhtest.AssertEqual(t, expected[i], tag)
				}
			},
		},
		{
			name: "custom tag names",
			formData: url.Values{
				"user_name": []string{"johndoe"},
			},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				zhtest.AssertEqual(t, "johndoe", result.Username)
			},
		},
		{
			name: "ignored field",
			formData: url.Values{
				"ignored": []string{"should not be set"},
			},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				zhtest.AssertEmpty(t, result.Ignored)
			},
		},
		{
			name:      "empty form",
			formData:  url.Values{},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				// All fields should remain at zero values
				zhtest.AssertEmpty(t, result.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

			var result testFormStruct
			err := B.Form(req, &result)

			if tt.wantError {
				zhtest.AssertError(t, err)
				return
			}

			zhtest.AssertNoError(t, err)

			tt.expected(t, &result)
		})
	}
}

func TestBinder_Form_Errors(t *testing.T) {
	tests := []struct {
		name     string
		formData url.Values
		errMsg   string
	}{
		{
			name:     "invalid integer",
			formData: url.Values{"age": []string{"not a number"}},
			errMsg:   "invalid integer",
		},
		{
			name:     "invalid float",
			formData: url.Values{"score": []string{"not a float"}},
			errMsg:   "invalid float",
		},
		{
			name:     "invalid boolean",
			formData: url.Values{"active": []string{"not a bool"}},
			errMsg:   "invalid boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

			var result testFormStruct
			err := B.Form(req, &result)

			zhtest.AssertError(t, err)
			zhtest.AssertErrorContains(t, err, tt.errMsg)
		})
	}
}

func TestBinder_Form_InvalidDestination(t *testing.T) {
	tests := []struct {
		name string
		dst  interface{}
	}{
		{
			name: "nil pointer",
			dst:  nil,
		},
		{
			name: "non-pointer",
			dst:  testFormStruct{},
		},
		{
			name: "pointer to non-struct",
			dst:  new(string),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=test"))
			req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

			var err error
			if tt.dst == nil {
				err = bindValues(url.Values{"name": []string{"test"}}, tt.dst, "form", false)
			} else {
				err = B.Form(req, tt.dst)
			}

			zhtest.AssertError(t, err)
		})
	}
}

func TestBinder_MultipartForm(t *testing.T) {
	tests := []struct {
		name      string
		setupForm func(*multipart.Writer)
		wantError bool
		expected  func(t *testing.T, result *testMultipartStruct)
	}{
		{
			name: "form values only",
			setupForm: func(w *multipart.Writer) {
				_ = w.WriteField("name", "John")
				_ = w.WriteField("age", "30")
			},
			wantError: false,
			expected: func(t *testing.T, result *testMultipartStruct) {
				zhtest.AssertEqual(t, "John", result.Name)
				zhtest.AssertEqual(t, 30, result.Age)
			},
		},
		{
			name: "single file upload",
			setupForm: func(w *multipart.Writer) {
				_ = w.WriteField("name", "Test Upload")
				fileWriter, _ := w.CreateFormFile("document", "test.txt")
				_, _ = fileWriter.Write([]byte("Hello, World!"))
			},
			wantError: false,
			expected: func(t *testing.T, result *testMultipartStruct) {
				zhtest.AssertNotNil(t, result.Document)
				zhtest.AssertEqual(t, "test.txt", result.Document.Filename)
				zhtest.AssertEqual(t, int64(13), result.Document.Size)
			},
		},
		{
			name: "multiple file uploads",
			setupForm: func(w *multipart.Writer) {
				for i := 0; i < 3; i++ {
					fileWriter, _ := w.CreateFormFile("attachments", "file"+string(rune('0'+i))+".txt")
					_, _ = fileWriter.Write([]byte("content" + string(rune('0'+i))))
				}
			},
			wantError: false,
			expected: func(t *testing.T, result *testMultipartStruct) {
				zhtest.AssertLen(t, result.Attachments, 3)
				expectedNames := []string{"file0.txt", "file1.txt", "file2.txt"}
				for i, fh := range result.Attachments {
					zhtest.AssertEqual(t, expectedNames[i], fh.Filename)
				}
			},
		},
		{
			name: "files and values together",
			setupForm: func(w *multipart.Writer) {
				_ = w.WriteField("name", "Mixed Content")
				_ = w.WriteField("age", "25")
				fileWriter, _ := w.CreateFormFile("document", "mixed.txt")
				_, _ = fileWriter.Write([]byte("Mixed"))
			},
			wantError: false,
			expected: func(t *testing.T, result *testMultipartStruct) {
				zhtest.AssertEqual(t, "Mixed Content", result.Name)
				zhtest.AssertEqual(t, 25, result.Age)
				zhtest.AssertNotNil(t, result.Document)
				zhtest.AssertEqual(t, "mixed.txt", result.Document.Filename)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			tt.setupForm(writer)
			_ = writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/", &body)
			req.Header.Set(httpx.HeaderContentType, writer.FormDataContentType())

			var result testMultipartStruct
			err := B.MultipartForm(req, &result, 32<<20)

			if tt.wantError {
				zhtest.AssertError(t, err)
				return
			}

			zhtest.AssertNoError(t, err)

			tt.expected(t, &result)

			// Cleanup: close any open file handles
			if result.Document != nil && result.Document.File != nil {
				_ = result.Document.File.Close()
			}
			for _, fh := range result.Attachments {
				if fh != nil && fh.File != nil {
					_ = fh.File.Close()
				}
			}
		})
	}
}

func TestBindValues_EmbeddedStruct(t *testing.T) {
	type Embedded struct {
		Name string `form:"name"`
	}

	type Container struct {
		Embedded
		Email string `form:"email"`
	}

	values := url.Values{
		"name":  []string{"John"},
		"email": []string{"john@example.com"},
	}

	var result Container
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "John", result.Name)
	zhtest.AssertEqual(t, "john@example.com", result.Email)
}

func TestBindValues_PointerFields(t *testing.T) {
	type WithPointers struct {
		Name  *string  `form:"name"`
		Age   *int     `form:"age"`
		Score *float64 `form:"score"`
	}

	values := url.Values{
		"name":  []string{"John"},
		"age":   []string{"30"},
		"score": []string{"95.5"},
	}

	var result WithPointers
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	zhtest.AssertNotNil(t, result.Name)
	zhtest.AssertEqual(t, "John", *result.Name)
	zhtest.AssertNotNil(t, result.Age)
	zhtest.AssertEqual(t, 30, *result.Age)
	zhtest.AssertNotNil(t, result.Score)
	zhtest.AssertEqual(t, 95.5, *result.Score)
}

func TestBindValues_IntSlice(t *testing.T) {
	type WithIntSlice struct {
		IDs []int `form:"ids"`
	}

	values := url.Values{
		"ids": []string{"1", "2", "3"},
	}

	var result WithIntSlice
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	expected := []int{1, 2, 3}
	zhtest.AssertLen(t, result.IDs, len(expected))

	for i, id := range result.IDs {
		zhtest.AssertEqual(t, expected[i], id)
	}
}

// Test struct types
type testFormStruct struct {
	Name     string   `form:"name"`
	Email    string   `form:"email"`
	Age      int      `form:"age"`
	Score    float64  `form:"score"`
	Count    uint     `form:"count"`
	Active   bool     `form:"active"`
	Tags     []string `form:"tags"`
	Username string   `form:"user_name"`
	Ignored  string   `form:"-"`
}

type testMultipartStruct struct {
	Name        string        `form:"name"`
	Age         int           `form:"age"`
	Document    *FileHeader   `form:"document"`
	Attachments []*FileHeader `form:"attachments"`
}

// Query binding test structs
type testQueryStruct struct {
	Page     int      `query:"page"`
	Limit    int      `query:"limit"`
	Search   string   `query:"search"`
	Active   bool     `query:"active"`
	Tags     []string `query:"tags"`
	Category string   `query:"category"`
	Ignored  string   `query:"-"`
}

type testQueryEmbedded struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

type testQueryContainer struct {
	testQueryEmbedded
	Search string `query:"search"`
}

type testQueryPointers struct {
	Name   *string `query:"name"`
	Age    *int    `query:"age"`
	Active *bool   `query:"active"`
}

type testQuerySlices struct {
	IDs     []int     `query:"ids"`
	Scores  []float64 `query:"scores"`
	Enabled []bool    `query:"enabled"`
}

func TestBinder_Query(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantError  bool
		expected   func(t *testing.T, result *testQueryStruct)
		errContain string
	}{
		{
			name:      "basic fields",
			query:     "page=1&limit=20&search=hello",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				zhtest.AssertEqual(t, 1, result.Page)
				zhtest.AssertEqual(t, 20, result.Limit)
				zhtest.AssertEqual(t, "hello", result.Search)
			},
		},
		{
			name:      "boolean field",
			query:     "active=true",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				zhtest.AssertTrue(t, result.Active)
			},
		},
		{
			name:      "slice values",
			query:     "tags=go&tags=web&tags=api",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				expected := []string{"go", "web", "api"}
				zhtest.AssertLen(t, result.Tags, len(expected))
				for i, tag := range result.Tags {
					zhtest.AssertEqual(t, expected[i], tag)
				}
			},
		},
		{
			name:      "ignored field",
			query:     "ignored=shouldnotset",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				zhtest.AssertEmpty(t, result.Ignored)
			},
		},
		{
			name:      "empty query",
			query:     "",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				zhtest.AssertEqual(t, 0, result.Page)
				zhtest.AssertEmpty(t, result.Search)
			},
		},
		{
			name:       "invalid integer",
			query:      "page=abc",
			wantError:  true,
			errContain: "invalid integer",
		},
		{
			name:       "invalid boolean",
			query:      "active=notabool",
			wantError:  true,
			errContain: "invalid boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			var result testQueryStruct
			err := B.Query(req, &result)

			if tt.wantError {
				zhtest.AssertError(t, err)
				if tt.errContain != "" {
					zhtest.AssertErrorContains(t, err, tt.errContain)
				}
				return
			}

			zhtest.AssertNoError(t, err)

			tt.expected(t, &result)
		})
	}
}

func TestBinder_Query_EmbeddedStruct(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=2&limit=50&search=test", nil)

	var result testQueryContainer
	err := B.Query(req, &result)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, 2, result.Page)
	zhtest.AssertEqual(t, 50, result.Limit)
	zhtest.AssertEqual(t, "test", result.Search)
}

func TestBinder_Query_PointerFields(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected func(t *testing.T, result *testQueryPointers)
	}{
		{
			name:  "all fields provided",
			query: "name=John&age=30&active=true",
			expected: func(t *testing.T, result *testQueryPointers) {
				zhtest.AssertNotNil(t, result.Name)
				zhtest.AssertEqual(t, "John", *result.Name)
				zhtest.AssertNotNil(t, result.Age)
				zhtest.AssertEqual(t, 30, *result.Age)
				zhtest.AssertNotNil(t, result.Active)
				zhtest.AssertEqual(t, true, *result.Active)
			},
		},
		{
			name:  "some fields missing",
			query: "name=Jane",
			expected: func(t *testing.T, result *testQueryPointers) {
				zhtest.AssertNotNil(t, result.Name)
				zhtest.AssertEqual(t, "Jane", *result.Name)
				zhtest.AssertNil(t, result.Age)
				zhtest.AssertNil(t, result.Active)
			},
		},
		{
			name:  "empty values are nil for pointers",
			query: "name=&age=",
			expected: func(t *testing.T, result *testQueryPointers) {
				// Per design doc: empty string = not provided for pointers
				zhtest.AssertNil(t, result.Name)
				zhtest.AssertNil(t, result.Age)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			var result testQueryPointers
			err := B.Query(req, &result)
			zhtest.AssertNoError(t, err)

			tt.expected(t, &result)
		})
	}
}

func TestBinder_Query_SliceTypes(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected func(t *testing.T, result *testQuerySlices)
	}{
		{
			name:  "int slice",
			query: "ids=1&ids=2&ids=3",
			expected: func(t *testing.T, result *testQuerySlices) {
				expected := []int{1, 2, 3}
				zhtest.AssertLen(t, result.IDs, len(expected))
				for i, id := range result.IDs {
					zhtest.AssertEqual(t, expected[i], id)
				}
			},
		},
		{
			name:  "float64 slice",
			query: "scores=95.5&scores=87.2&scores=100",
			expected: func(t *testing.T, result *testQuerySlices) {
				expected := []float64{95.5, 87.2, 100}
				zhtest.AssertLen(t, result.Scores, len(expected))
				for i, score := range result.Scores {
					zhtest.AssertEqual(t, expected[i], score)
				}
			},
		},
		{
			name:  "bool slice",
			query: "enabled=true&enabled=false&enabled=1",
			expected: func(t *testing.T, result *testQuerySlices) {
				expected := []bool{true, false, true}
				zhtest.AssertLen(t, result.Enabled, len(expected))
				for i, val := range result.Enabled {
					zhtest.AssertEqual(t, expected[i], val)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			var result testQuerySlices
			err := B.Query(req, &result)
			zhtest.AssertNoError(t, err)

			tt.expected(t, &result)
		})
	}
}

func TestBinder_Query_ImplicitSnakeCase(t *testing.T) {
	type ImplicitNaming struct {
		UserName   string `query:"user_name"`
		FirstName  string // should map to first_name
		LastName   string // should map to last_name
		HTTPMethod string // should map to h_t_t_p_method
	}

	req := httptest.NewRequest(http.MethodGet, "/?user_name=johndoe&first_name=John&last_name=Doe&h_t_t_p_method=GET", nil)

	var result ImplicitNaming
	err := B.Query(req, &result)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "johndoe", result.UserName)
	zhtest.AssertEqual(t, "John", result.FirstName)
	zhtest.AssertEqual(t, "Doe", result.LastName)
	zhtest.AssertEqual(t, http.MethodGet, result.HTTPMethod)
}

func TestBinder_Query_InvalidDestination(t *testing.T) {
	tests := []struct {
		name string
		dst  interface{}
	}{
		{
			name: "nil pointer",
			dst:  nil,
		},
		{
			name: "non-pointer",
			dst:  testQueryStruct{},
		},
		{
			name: "pointer to non-struct",
			dst:  new(string),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?page=1", nil)

			var err error
			if tt.dst == nil {
				err = bindValues(req.URL.Query(), tt.dst, "query", false)
			} else {
				err = B.Query(req, tt.dst)
			}

			zhtest.AssertError(t, err)
		})
	}
}

func TestBinder_Query_URLEncodedValues(t *testing.T) {
	type URLValues struct {
		Search string `query:"search"`
		Path   string `query:"path"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?search=hello%20world&path=%2Ffoo%2Fbar", nil)

	var result URLValues
	err := B.Query(req, &result)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "hello world", result.Search)
	zhtest.AssertEqual(t, "/foo/bar", result.Path)
}

func TestBindEmbeddedStruct_NestedEmbedded(t *testing.T) {
	type DeepNested struct {
		Deep string `form:"deep"`
	}
	type Nested struct {
		DeepNested
		Middle string `form:"middle"`
	}
	type Container struct {
		Nested
		Top string `form:"top"`
	}

	values := url.Values{
		"top":    []string{"top_value"},
		"middle": []string{"middle_value"},
		"deep":   []string{"deep_value"},
	}

	var result Container
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "top_value", result.Top)
	zhtest.AssertEqual(t, "middle_value", result.Middle)
	zhtest.AssertEqual(t, "deep_value", result.Deep)
}

func TestBindSliceValue_InvalidElement(t *testing.T) {
	type WithIntSlice struct {
		IDs []int `form:"ids"`
	}

	values := url.Values{
		"ids": []string{"1", "not_a_number", "3"},
	}

	var result WithIntSlice
	err := bindValues(values, &result, "form", false)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "invalid integer")
}

func TestBindSliceValue_InvalidFloatSlice(t *testing.T) {
	type WithFloatSlice struct {
		Scores []float64 `form:"scores"`
	}

	values := url.Values{
		"scores": []string{"1.5", "invalid", "3.5"},
	}

	var result WithFloatSlice
	err := bindValues(values, &result, "form", false)
	zhtest.AssertError(t, err)
}

func TestBindSliceValue_InvalidBoolSlice(t *testing.T) {
	type WithBoolSlice struct {
		Flags []bool `form:"flags"`
	}

	values := url.Values{
		"flags": []string{"true", "not_a_bool", "false"},
	}

	var result WithBoolSlice
	err := bindValues(values, &result, "form", false)
	zhtest.AssertError(t, err)
}

func TestBindFileField_NonPointerFileHeader(t *testing.T) {
	type WithNonPointerFileHeader struct {
		Document FileHeader `form:"document"`
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "Test")
	fileWriter, _ := writer.CreateFormFile("document", "test.txt")
	_, _ = fileWriter.Write([]byte("content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(httpx.HeaderContentType, writer.FormDataContentType())

	var result WithNonPointerFileHeader
	err := B.MultipartForm(req, &result, 32<<20)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "must be a pointer")
}

func TestBindEmbeddedStruct_WithFiles(t *testing.T) {
	type FileContainer struct {
		Name     string      `form:"name"`
		Document *FileHeader `form:"document"`
	}

	type Wrapper struct {
		FileContainer
		Extra string `form:"extra"`
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "TestName")
	_ = writer.WriteField("extra", "ExtraValue")
	fileWriter, _ := writer.CreateFormFile("document", "embedded.txt")
	_, _ = fileWriter.Write([]byte("embedded content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(httpx.HeaderContentType, writer.FormDataContentType())

	var result Wrapper
	err := B.MultipartForm(req, &result, 32<<20)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "TestName", result.Name)
	zhtest.AssertEqual(t, "ExtraValue", result.Extra)
	zhtest.AssertNotNil(t, result.Document)
	zhtest.AssertEqual(t, "embedded.txt", result.Document.Filename)
}

func TestBindValues_UnexportedFields(t *testing.T) {
	type WithUnexported struct {
		Name    string `form:"name"`
		ignored string `form:"ignored"` // Unexported field should be skipped
	}

	values := url.Values{
		"name":    []string{"John"},
		"ignored": []string{"should not bind"},
	}

	var result WithUnexported
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "John", result.Name)
	zhtest.AssertEmpty(t, result.ignored)
}

func TestBindEmbeddedStruct_UnexportedField(t *testing.T) {
	type Inner struct {
		visible string // Unexported within embedded struct
		Value   string `form:"value"`
	}

	type Container struct {
		Inner
		Name string `form:"name"`
	}

	values := url.Values{
		"name":    []string{"container"},
		"value":   []string{"inner_value"},
		"visible": []string{"should_not_bind"},
	}

	var result Container
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	// visible is unexported so it should not be bound
	zhtest.AssertEmpty(t, result.visible)

	zhtest.AssertEqual(t, "container", result.Name)
	zhtest.AssertEqual(t, "inner_value", result.Value)
}

func TestBindEmbeddedStruct_IgnoredField(t *testing.T) {
	type Inner struct {
		Ignored string `form:"-"`
		Value   string `form:"value"`
	}

	type Container struct {
		Inner
	}

	values := url.Values{
		"ignored": []string{"should_not_set"},
		"value":   []string{"actual_value"},
	}

	var result Container
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEmpty(t, result.Ignored)
	zhtest.AssertEqual(t, "actual_value", result.Value)
}

func TestBindValues_EmptyTagName(t *testing.T) {
	type NoTag struct {
		UserName string // Will use snake_case: user_name
	}

	values := url.Values{
		"user_name": []string{"johndoe"},
	}

	var result NoTag
	err := bindValues(values, &result, "form", false)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "johndoe", result.UserName)
}

func TestBindEmbeddedStruct_ErrorPropagation(t *testing.T) {
	// Test that errors from nested embedded structs propagate correctly
	type DeepNested struct {
		Value int `form:"value"`
	}

	type Middle struct {
		DeepNested
	}

	type Container struct {
		Middle
	}

	values := url.Values{
		"value": []string{"not_a_number"},
	}

	var result Container
	err := bindValues(values, &result, "form", false)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "invalid integer")
}

func TestBinder_Form_ParseFormError(t *testing.T) {
	// Create a request with invalid content type that causes ParseForm to fail
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not=valid"))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)
	req.Body = &errReader{}

	var result testFormStruct
	err := B.Form(req, &result)
	zhtest.AssertError(t, err)
}

// errReader simulates a read error
type errReader struct{}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (e *errReader) Close() error {
	return nil
}

func TestBinder_MultipartForm_ParseError(t *testing.T) {
	// Request with malformed multipart data
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not valid multipart"))
	req.Header.Set(httpx.HeaderContentType, "multipart/form-data; boundary=xyz")

	var result testMultipartStruct
	err := B.MultipartForm(req, &result, 32<<20)
	zhtest.AssertError(t, err)
}

func TestBinder_MultipartForm_NoFormData(t *testing.T) {
	// Create a request that parses but has no multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "test")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(httpx.HeaderContentType, writer.FormDataContentType())

	// Manually parse then nil out the MultipartForm
	_ = req.ParseMultipartForm(32 << 20)
	req.MultipartForm = nil

	var result testMultipartStruct
	err := B.MultipartForm(req, &result, 32<<20)
	zhtest.AssertError(t, err)
}

func TestBindFilesToStruct_NoMultipartForm(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := BindMultipartFormFiles(req, &testMultipartStruct{})
	zhtest.AssertNoError(t, err)
}

func TestBindFilesToStruct_InvalidDestination(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, _ := writer.CreateFormFile("test", "file.txt")
	_, _ = fileWriter.Write([]byte("content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(httpx.HeaderContentType, writer.FormDataContentType())
	_ = req.ParseMultipartForm(32 << 20)

	tests := []struct {
		name string
		dst  interface{}
	}{
		{
			name: "non-pointer",
			dst:  testMultipartStruct{},
		},
		{
			name: "pointer to non-struct",
			dst:  new(string),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BindMultipartFormFiles(req, tt.dst)
			zhtest.AssertError(t, err)
		})
	}
}
