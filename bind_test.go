package zerohttp

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Verify valid cases
			if tt.name == "valid JSON" {
				if result.Name == nil || *result.Name != "John" {
					t.Errorf("expected name 'John', got %v", result.Name)
				}
				if result.Age == nil || *result.Age != 30 {
					t.Errorf("expected age 30, got %v", result.Age)
				}
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
				if result.Name != "John" {
					t.Errorf("expected Name='John', got %q", result.Name)
				}
				if result.Email != "john@example.com" {
					t.Errorf("expected Email='john@example.com', got %q", result.Email)
				}
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
				if result.Age != 30 {
					t.Errorf("expected Age=30, got %d", result.Age)
				}
				if result.Score != 95.5 {
					t.Errorf("expected Score=95.5, got %f", result.Score)
				}
				if result.Count != 100 {
					t.Errorf("expected Count=100, got %d", result.Count)
				}
				if !result.Active {
					t.Errorf("expected Active=true, got %v", result.Active)
				}
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
				if len(result.Tags) != len(expected) {
					t.Errorf("expected %d tags, got %d", len(expected), len(result.Tags))
					return
				}
				for i, tag := range result.Tags {
					if tag != expected[i] {
						t.Errorf("expected tag[%d]=%q, got %q", i, expected[i], tag)
					}
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
				if result.Username != "johndoe" {
					t.Errorf("expected Username='johndoe', got %q", result.Username)
				}
			},
		},
		{
			name: "ignored field",
			formData: url.Values{
				"ignored": []string{"should not be set"},
			},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				if result.Ignored != "" {
					t.Errorf("expected Ignored to be empty, got %q", result.Ignored)
				}
			},
		},
		{
			name:      "empty form",
			formData:  url.Values{},
			wantError: false,
			expected: func(t *testing.T, result *testFormStruct) {
				// All fields should remain at zero values
				if result.Name != "" {
					t.Errorf("expected empty Name, got %q", result.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			var result testFormStruct
			err := B.Form(req, &result)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

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
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			var result testFormStruct
			err := B.Form(req, &result)

			if err == nil {
				t.Fatal("expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error to contain %q, got %v", tt.errMsg, err)
			}
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
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			var err error
			if tt.dst == nil {
				err = bindValues(url.Values{"name": []string{"test"}}, tt.dst, "form", false)
			} else {
				err = B.Form(req, tt.dst)
			}

			if err == nil {
				t.Fatal("expected error but got none")
			}
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
				if result.Name != "John" {
					t.Errorf("expected Name='John', got %q", result.Name)
				}
				if result.Age != 30 {
					t.Errorf("expected Age=30, got %d", result.Age)
				}
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
				if result.Document == nil {
					t.Fatal("expected Document to be set")
				}
				if result.Document.Filename != "test.txt" {
					t.Errorf("expected Filename='test.txt', got %q", result.Document.Filename)
				}
				if result.Document.Size != 13 {
					t.Errorf("expected Size=13, got %d", result.Document.Size)
				}
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
				if len(result.Attachments) != 3 {
					t.Errorf("expected 3 attachments, got %d", len(result.Attachments))
					return
				}
				expectedNames := []string{"file0.txt", "file1.txt", "file2.txt"}
				for i, fh := range result.Attachments {
					if fh.Filename != expectedNames[i] {
						t.Errorf("expected attachment[%d].Filename=%q, got %q", i, expectedNames[i], fh.Filename)
					}
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
				if result.Name != "Mixed Content" {
					t.Errorf("expected Name='Mixed Content', got %q", result.Name)
				}
				if result.Age != 25 {
					t.Errorf("expected Age=25, got %d", result.Age)
				}
				if result.Document == nil {
					t.Fatal("expected Document to be set")
				}
				if result.Document.Filename != "mixed.txt" {
					t.Errorf("expected Filename='mixed.txt', got %q", result.Document.Filename)
				}
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
			req.Header.Set("Content-Type", writer.FormDataContentType())

			var result testMultipartStruct
			err := B.MultipartForm(req, &result, 32<<20)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			tt.expected(t, &result)

			// Cleanup: close any open file handles
			if result.Document != nil && result.Document.file != nil {
				_ = result.Document.file.Close()
			}
			for _, fh := range result.Attachments {
				if fh != nil && fh.file != nil {
					_ = fh.file.Close()
				}
			}
		})
	}
}

func TestFileHeader_Open(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, _ := writer.CreateFormFile("test", "hello.txt")
	_, _ = fileWriter.Write([]byte("Hello"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatalf("failed to parse multipart form: %v", err)
	}

	fileHeaders := req.MultipartForm.File["test"]
	if len(fileHeaders) == 0 {
		t.Fatal("no files found")
	}

	file, err := fileHeaders[0].Open()
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "Hello" {
		t.Errorf("expected content 'Hello', got %q", string(content))
	}
}

func TestFileHeader_ReadAll(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, _ := writer.CreateFormFile("test", "hello.txt")
	_, _ = fileWriter.Write([]byte("Hello, World!"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var result struct {
		File *FileHeader `form:"test"`
	}

	err := B.MultipartForm(req, &result, 32<<20)
	if err != nil {
		t.Fatalf("failed to bind multipart form: %v", err)
	}

	if result.File == nil {
		t.Fatal("expected File to be set")
	}

	content, err := result.File.ReadAll()
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("expected content 'Hello, World!', got %q", string(content))
	}
}

func TestFileHeader_Open_AfterClose(t *testing.T) {
	fh := &FileHeader{
		Filename: "test.txt",
		Size:     5,
		file:     nil, // Simulate closed file
	}

	_, err := fh.Open()
	if err == nil {
		t.Fatal("expected error when opening closed file")
	}

	if !strings.Contains(err.Error(), "no longer available") {
		t.Errorf("expected 'no longer available' error, got %v", err)
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Name", "name"},
		{"UserName", "user_name"},
		{"EmailAddress", "email_address"},
		{"ID", "i_d"},
		{"Simple", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := camelToSnake(tt.input)
			if result != tt.expected {
				t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, result, tt.expected)
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "John" {
		t.Errorf("expected Name='John', got %q", result.Name)
	}
	if result.Email != "john@example.com" {
		t.Errorf("expected Email='john@example.com', got %q", result.Email)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name == nil || *result.Name != "John" {
		t.Errorf("expected Name='John', got %v", result.Name)
	}
	if result.Age == nil || *result.Age != 30 {
		t.Errorf("expected Age=30, got %v", result.Age)
	}
	if result.Score == nil || *result.Score != 95.5 {
		t.Errorf("expected Score=95.5, got %v", result.Score)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []int{1, 2, 3}
	if len(result.IDs) != len(expected) {
		t.Fatalf("expected %d IDs, got %d", len(expected), len(result.IDs))
	}

	for i, id := range result.IDs {
		if id != expected[i] {
			t.Errorf("expected IDs[%d]=%d, got %d", i, expected[i], id)
		}
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
				if result.Page != 1 {
					t.Errorf("expected Page=1, got %d", result.Page)
				}
				if result.Limit != 20 {
					t.Errorf("expected Limit=20, got %d", result.Limit)
				}
				if result.Search != "hello" {
					t.Errorf("expected Search='hello', got %q", result.Search)
				}
			},
		},
		{
			name:      "boolean field",
			query:     "active=true",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				if !result.Active {
					t.Errorf("expected Active=true, got %v", result.Active)
				}
			},
		},
		{
			name:      "slice values",
			query:     "tags=go&tags=web&tags=api",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				expected := []string{"go", "web", "api"}
				if len(result.Tags) != len(expected) {
					t.Errorf("expected %d tags, got %d", len(expected), len(result.Tags))
					return
				}
				for i, tag := range result.Tags {
					if tag != expected[i] {
						t.Errorf("expected tag[%d]=%q, got %q", i, expected[i], tag)
					}
				}
			},
		},
		{
			name:      "ignored field",
			query:     "ignored=shouldnotset",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				if result.Ignored != "" {
					t.Errorf("expected Ignored to be empty, got %q", result.Ignored)
				}
			},
		},
		{
			name:      "empty query",
			query:     "",
			wantError: false,
			expected: func(t *testing.T, result *testQueryStruct) {
				if result.Page != 0 {
					t.Errorf("expected Page=0, got %d", result.Page)
				}
				if result.Search != "" {
					t.Errorf("expected Search='', got %q", result.Search)
				}
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
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("expected error to contain %q, got %v", tt.errContain, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			tt.expected(t, &result)
		})
	}
}

func TestBinder_Query_EmbeddedStruct(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=2&limit=50&search=test", nil)

	var result testQueryContainer
	err := B.Query(req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Page != 2 {
		t.Errorf("expected Page=2, got %d", result.Page)
	}
	if result.Limit != 50 {
		t.Errorf("expected Limit=50, got %d", result.Limit)
	}
	if result.Search != "test" {
		t.Errorf("expected Search='test', got %q", result.Search)
	}
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
				if result.Name == nil || *result.Name != "John" {
					t.Errorf("expected Name='John', got %v", result.Name)
				}
				if result.Age == nil || *result.Age != 30 {
					t.Errorf("expected Age=30, got %v", result.Age)
				}
				if result.Active == nil || *result.Active != true {
					t.Errorf("expected Active=true, got %v", result.Active)
				}
			},
		},
		{
			name:  "some fields missing",
			query: "name=Jane",
			expected: func(t *testing.T, result *testQueryPointers) {
				if result.Name == nil || *result.Name != "Jane" {
					t.Errorf("expected Name='Jane', got %v", result.Name)
				}
				if result.Age != nil {
					t.Errorf("expected Age=nil, got %v", result.Age)
				}
				if result.Active != nil {
					t.Errorf("expected Active=nil, got %v", result.Active)
				}
			},
		},
		{
			name:  "empty values are nil for pointers",
			query: "name=&age=",
			expected: func(t *testing.T, result *testQueryPointers) {
				// Per design doc: empty string = not provided for pointers
				if result.Name != nil {
					t.Errorf("expected Name=nil for empty value, got %q", *result.Name)
				}
				if result.Age != nil {
					t.Errorf("expected Age=nil for empty value, got %d", *result.Age)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			var result testQueryPointers
			err := B.Query(req, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

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
				if len(result.IDs) != len(expected) {
					t.Fatalf("expected %d IDs, got %d", len(expected), len(result.IDs))
				}
				for i, id := range result.IDs {
					if id != expected[i] {
						t.Errorf("expected IDs[%d]=%d, got %d", i, expected[i], id)
					}
				}
			},
		},
		{
			name:  "float64 slice",
			query: "scores=95.5&scores=87.2&scores=100",
			expected: func(t *testing.T, result *testQuerySlices) {
				expected := []float64{95.5, 87.2, 100}
				if len(result.Scores) != len(expected) {
					t.Fatalf("expected %d scores, got %d", len(expected), len(result.Scores))
				}
				for i, score := range result.Scores {
					if score != expected[i] {
						t.Errorf("expected Scores[%d]=%f, got %f", i, expected[i], score)
					}
				}
			},
		},
		{
			name:  "bool slice",
			query: "enabled=true&enabled=false&enabled=1",
			expected: func(t *testing.T, result *testQuerySlices) {
				expected := []bool{true, false, true}
				if len(result.Enabled) != len(expected) {
					t.Fatalf("expected %d enabled values, got %d", len(expected), len(result.Enabled))
				}
				for i, val := range result.Enabled {
					if val != expected[i] {
						t.Errorf("expected Enabled[%d]=%v, got %v", i, expected[i], val)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			var result testQuerySlices
			err := B.Query(req, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.UserName != "johndoe" {
		t.Errorf("expected UserName='johndoe', got %q", result.UserName)
	}
	if result.FirstName != "John" {
		t.Errorf("expected FirstName='John', got %q", result.FirstName)
	}
	if result.LastName != "Doe" {
		t.Errorf("expected LastName='Doe', got %q", result.LastName)
	}
	if result.HTTPMethod != "GET" {
		t.Errorf("expected HTTPMethod='GET', got %q", result.HTTPMethod)
	}
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

			if err == nil {
				t.Fatal("expected error but got none")
			}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Search != "hello world" {
		t.Errorf("expected Search='hello world', got %q", result.Search)
	}
	if result.Path != "/foo/bar" {
		t.Errorf("expected Path='/foo/bar', got %q", result.Path)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Top != "top_value" {
		t.Errorf("expected Top='top_value', got %q", result.Top)
	}
	if result.Middle != "middle_value" {
		t.Errorf("expected Middle='middle_value', got %q", result.Middle)
	}
	if result.Deep != "deep_value" {
		t.Errorf("expected Deep='deep_value', got %q", result.Deep)
	}
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
	if err == nil {
		t.Fatal("expected error for invalid slice element")
	}
	if !strings.Contains(err.Error(), "invalid integer") {
		t.Errorf("expected error to contain 'invalid integer', got %v", err)
	}
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
	if err == nil {
		t.Fatal("expected error for invalid float slice element")
	}
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
	if err == nil {
		t.Fatal("expected error for invalid bool slice element")
	}
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
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var result WithNonPointerFileHeader
	err := B.MultipartForm(req, &result, 32<<20)
	if err == nil {
		t.Fatal("expected error for non-pointer FileHeader")
	}
	if !strings.Contains(err.Error(), "must be a pointer") {
		t.Errorf("expected error to contain 'must be a pointer', got %v", err)
	}
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
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var result Wrapper
	err := B.MultipartForm(req, &result, 32<<20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "TestName" {
		t.Errorf("expected Name='TestName', got %q", result.Name)
	}
	if result.Extra != "ExtraValue" {
		t.Errorf("expected Extra='ExtraValue', got %q", result.Extra)
	}
	if result.Document == nil {
		t.Fatal("expected Document to be set")
	}
	if result.Document.Filename != "embedded.txt" {
		t.Errorf("expected Filename='embedded.txt', got %q", result.Document.Filename)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "John" {
		t.Errorf("expected Name='John', got %q", result.Name)
	}
	if result.ignored != "" {
		t.Errorf("expected ignored to remain empty, got %q", result.ignored)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// visible is unexported so it should not be bound
	if result.visible != "" {
		t.Errorf("expected visible to remain empty (unexported), got %q", result.visible)
	}

	if result.Name != "container" {
		t.Errorf("expected Name='container', got %q", result.Name)
	}
	if result.Value != "inner_value" {
		t.Errorf("expected Value='inner_value', got %q", result.Value)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Ignored != "" {
		t.Errorf("expected Ignored to remain empty, got %q", result.Ignored)
	}
	if result.Value != "actual_value" {
		t.Errorf("expected Value='actual_value', got %q", result.Value)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.UserName != "johndoe" {
		t.Errorf("expected UserName='johndoe', got %q", result.UserName)
	}
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
	if err == nil {
		t.Fatal("expected error for invalid int conversion in nested embedded struct")
	}
	if !strings.Contains(err.Error(), "invalid integer") {
		t.Errorf("expected 'invalid integer' error, got %v", err)
	}
}

func TestFileHeader_ReadAll_OpenError(t *testing.T) {
	// Create a FileHeader with no internal file reference
	fh := &FileHeader{
		Filename: "test.txt",
		Size:     5,
		file:     nil, // Simulate closed/unavailable file
	}

	_, err := fh.ReadAll()
	if err == nil {
		t.Fatal("expected error when file is not available")
	}
	if !strings.Contains(err.Error(), "no longer available") {
		t.Errorf("expected 'no longer available' error, got %v", err)
	}
}

func TestBinder_Form_ParseFormError(t *testing.T) {
	// Create a request with invalid content type that causes ParseForm to fail
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = &errReader{}

	var result testFormStruct
	err := B.Form(req, &result)
	if err == nil {
		t.Fatal("expected error for invalid form data")
	}
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
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")

	var result testMultipartStruct
	err := B.MultipartForm(req, &result, 32<<20)
	if err == nil {
		t.Fatal("expected error for invalid multipart form")
	}
}

func TestBinder_MultipartForm_NoFormData(t *testing.T) {
	// Create a request that parses but has no multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "test")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Manually parse then nil out the MultipartForm
	_ = req.ParseMultipartForm(32 << 20)
	req.MultipartForm = nil

	var result testMultipartStruct
	err := B.MultipartForm(req, &result, 32<<20)
	if err == nil {
		t.Fatal("expected error when MultipartForm is nil")
	}
}

func TestBindFilesToStruct_NoMultipartForm(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := BindMultipartFormFiles(req, &testMultipartStruct{})
	if err != nil {
		t.Errorf("expected no error when no multipart form, got %v", err)
	}
}

func TestBindFilesToStruct_InvalidDestination(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, _ := writer.CreateFormFile("test", "file.txt")
	_, _ = fileWriter.Write([]byte("content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
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
			if err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}
