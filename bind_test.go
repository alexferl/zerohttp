package zerohttp

import (
	"bytes"
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
				err = bindValues(url.Values{"name": []string{"test"}}, tt.dst, false)
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
	err := bindValues(values, &result, false)
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
	err := bindValues(values, &result, false)
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
	err := bindValues(values, &result, false)
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
