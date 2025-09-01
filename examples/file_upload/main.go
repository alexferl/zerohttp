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
	"github.com/alexferl/zerohttp/config"
)

const (
	uploadDir   = "./uploads"
	maxFileSize = 10 << 20 // 10 MB per file
	maxFormSize = 32 << 20 // 32 MB total
)

func main() {
	app := zh.New(
		config.WithRequestBodySizeOptions(
			config.WithRequestBodySizeMaxBytes(maxFormSize),
		),
	)

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	app.GET("/", zh.HandlerFunc(uploadFormHandler))
	app.POST("/upload", zh.HandlerFunc(uploadHandler))
	app.GET("/files/{filename}", zh.HandlerFunc(downloadHandler))

	log.Printf("Server starting on :8080")
	log.Fatal(app.Start())
}

func uploadFormHandler(w http.ResponseWriter, r *http.Request) error {
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
	if err := r.ParseMultipartForm(maxFormSize); err != nil {
		problem := zh.NewProblemDetail(400, "Failed to parse form")
		problem.Set("error", err.Error())
		return problem.Render(w)
	}

	description := r.FormValue("description")
	files := r.MultipartForm.File["files"]

	if len(files) == 0 {
		return zh.NewProblemDetail(400, "No files uploaded").Render(w)
	}

	var uploadedFiles []zh.M
	var errors []string

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", fileHeader.Filename, err))
			continue
		}

		// Use anonymous function to handle defer properly in loop
		func() {
			defer func() {
				if err := file.Close(); err != nil {
					errors = append(errors, fmt.Sprintf("%s: failed to close - %v", fileHeader.Filename, err))
				}
			}()

			// Check file size
			if fileHeader.Size > maxFileSize {
				errors = append(errors, fmt.Sprintf("%s: file too large", fileHeader.Filename))
				return
			}

			filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileHeader.Filename)
			destPath := filepath.Join(uploadDir, filename)

			dest, err := os.Create(destPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: save failed", fileHeader.Filename))
				return
			}

			_, err = io.Copy(dest, file)
			if err != nil {
				_ = dest.Close()        // Try to close before cleanup
				_ = os.Remove(destPath) // Remove failed upload
				errors = append(errors, fmt.Sprintf("%s: copy failed", fileHeader.Filename))
				return
			}

			if err := dest.Close(); err != nil {
				_ = os.Remove(destPath) // Remove on close failure
				errors = append(errors, fmt.Sprintf("%s: failed to close destination - %v", fileHeader.Filename, err))
				return
			}

			// Success - file saved successfully, add to results
			uploadedFiles = append(uploadedFiles, zh.M{
				"filename":     filename,
				"original":     fileHeader.Filename,
				"size":         fileHeader.Size,
				"download_url": fmt.Sprintf("/files/%s", filename),
			})
		}()
	}

	response := zh.M{
		"message":     fmt.Sprintf("Uploaded %d of %d files", len(uploadedFiles), len(files)),
		"files":       uploadedFiles,
		"description": description,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := 201
	if len(uploadedFiles) == 0 {
		statusCode = 400 // All failed
	} else if len(errors) > 0 {
		statusCode = 207 // Partial success
	}

	return zh.R.JSON(w, statusCode, response)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) error {
	filename := filepath.Base(r.PathValue("filename")) // Security: prevent traversal
	filePath := filepath.Join(uploadDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return zh.NewProblemDetail(404, "File not found").Render(w)
	}

	return zh.R.File(w, r, filePath)
}
