package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware/requestbodysize"
)

const (
	uploadDir   = "./uploads"
	maxFileSize = 10 << 20 // 10 MB per file
	maxFormSize = 32 << 20 // 32 MB total
)

type UploadForm struct {
	Description string           `form:"description"`
	Files       []*zh.FileHeader `form:"files"`
}

func main() {
	app := zh.New(zh.Config{
		RequestBodySize: requestbodysize.Config{
			MaxBytes: maxFormSize,
		},
	})

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	app.GET("/", zh.HandlerFunc(uploadFormHandler))
	app.POST("/upload", zh.HandlerFunc(uploadHandler))
	app.GET("/files/{filename}", zh.HandlerFunc(downloadHandler))

	log.Fatal(app.Start())
}

func uploadFormHandler(w http.ResponseWriter, _ *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<body>
    <form action="/upload" method="POST" enctype="multipart/form-data">
        <input type="file" name="files" multiple required><br><br>
        <input type="text" name="description" placeholder="Description"><br><br>
        <input type="submit" value="Upload">
    </form>
</body>
</html>`
	return zh.R.HTML(w, 200, html)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) error {
	var form UploadForm

	if err := zh.B.MultipartForm(r, &form, maxFormSize); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Failed to parse form"))
	}

	if len(form.Files) == 0 {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "No files uploaded"))
	}

	var uploadedFiles []zh.M
	var errors []string

	for _, fileHeader := range form.Files {
		if fileHeader.Size > maxFileSize {
			errors = append(errors, fmt.Sprintf("%s: file too large", fileHeader.Filename))
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", fileHeader.Filename, err))
			continue
		}

		func() {
			defer func() { _ = file.Close() }()

			filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileHeader.Filename)
			destPath := filepath.Join(uploadDir, filename)

			dest, err := os.Create(destPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: save failed", fileHeader.Filename))
				return
			}

			if _, err = io.Copy(dest, file); err != nil {
				_ = dest.Close()
				_ = os.Remove(destPath)
				errors = append(errors, fmt.Sprintf("%s: copy failed", fileHeader.Filename))
				return
			}

			_ = dest.Close()

			uploadedFiles = append(uploadedFiles, zh.M{
				"filename":     filename,
				"original":     fileHeader.Filename,
				"size":         fileHeader.Size,
				"download_url": fmt.Sprintf("/files/%s", filename),
			})
		}()
	}

	response := zh.M{
		"message":     fmt.Sprintf("Uploaded %d of %d files", len(uploadedFiles), len(form.Files)),
		"files":       uploadedFiles,
		"description": form.Description,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := 201
	if len(uploadedFiles) == 0 {
		statusCode = 400
	} else if len(errors) > 0 {
		statusCode = 207
	}

	// Set Location header for single file upload
	if len(uploadedFiles) == 1 {
		w.Header().Set(httpx.HeaderLocation, uploadedFiles[0]["download_url"].(string))
	}

	return zh.R.JSON(w, statusCode, response)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) error {
	filename := filepath.Base(r.PathValue("filename"))
	filePath := filepath.Join(uploadDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(404, "File not found"))
	}

	return zh.R.File(w, r, filePath)
}
