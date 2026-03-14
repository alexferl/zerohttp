package bind

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

	// Parse multipart form directly (no binder)
	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("failed to read form: %v", err)
	}
	defer func() {
		if err := form.RemoveAll(); err != nil {
			t.Logf("failed to remove temp files: %v", err)
		}
	}()

	files := form.File["test"]
	if len(files) == 0 {
		t.Fatal("expected file in form")
	}

	// Open the file directly
	file, err := files[0].Open()
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}

	// Create FileHeader directly
	fh := &FileHeader{
		Filename: files[0].Filename,
		Size:     files[0].Size,
		Header:   files[0].Header,
		File:     file,
	}

	content, err := fh.ReadAll()
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
		File:     nil, // Simulate closed file
	}

	_, err := fh.Open()
	if err == nil {
		t.Fatal("expected error when opening closed file")
	}

	if !strings.Contains(err.Error(), "no longer available") {
		t.Errorf("expected 'no longer available' error, got %v", err)
	}
}

func TestFileHeader_ReadAll_OpenError(t *testing.T) {
	// Create a FileHeader with no internal file reference
	fh := &FileHeader{
		Filename: "test.txt",
		Size:     5,
		File:     nil, // Simulate closed/unavailable file
	}

	_, err := fh.ReadAll()
	if err == nil {
		t.Fatal("expected error when file is not available")
	}
	if !strings.Contains(err.Error(), "no longer available") {
		t.Errorf("expected 'no longer available' error, got %v", err)
	}
}
