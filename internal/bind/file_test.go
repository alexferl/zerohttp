package bind

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestFileHeader_Open(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, _ := writer.CreateFormFile("test", "hello.txt")
	_, _ = fileWriter.Write([]byte("Hello"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(httpx.HeaderContentType, writer.FormDataContentType())

	zhtest.AssertNoError(t, req.ParseMultipartForm(32<<20))

	fileHeaders := req.MultipartForm.File["test"]
	zhtest.AssertGreater(t, len(fileHeaders), 0)

	file, err := fileHeaders[0].Open()
	zhtest.AssertNoError(t, err)
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "Hello", string(content))
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
	zhtest.AssertNoError(t, err)
	defer func() {
		if err := form.RemoveAll(); err != nil {
			t.Logf("failed to remove temp files: %v", err)
		}
	}()

	files := form.File["test"]
	zhtest.AssertGreater(t, len(files), 0)

	// Open the file directly
	file, err := files[0].Open()
	zhtest.AssertNoError(t, err)

	// Create FileHeader directly
	fh := &FileHeader{
		Filename: files[0].Filename,
		Size:     files[0].Size,
		Header:   files[0].Header,
		File:     file,
	}

	content, err := fh.ReadAll()
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "Hello, World!", string(content))
}

func TestFileHeader_Open_AfterClose(t *testing.T) {
	fh := &FileHeader{
		Filename: "test.txt",
		Size:     5,
		File:     nil, // Simulate closed file
	}

	_, err := fh.Open()
	zhtest.AssertError(t, err)
	zhtest.AssertTrue(t, strings.Contains(err.Error(), "no longer available"))
}

func TestFileHeader_ReadAll_OpenError(t *testing.T) {
	// Create a FileHeader with no internal file reference
	fh := &FileHeader{
		Filename: "test.txt",
		Size:     5,
		File:     nil, // Simulate closed/unavailable file
	}

	_, err := fh.ReadAll()
	zhtest.AssertError(t, err)
	zhtest.AssertTrue(t, strings.Contains(err.Error(), "no longer available"))
}
