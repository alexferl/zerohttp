package main

import (
	"fmt"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

// LoginForm demonstrates simple form binding with basic types
type LoginForm struct {
	Username string `form:"username"`
	Password string `form:"password"`
	Remember bool   `form:"remember"`
}

// SearchForm demonstrates query parameter binding and slices
type SearchForm struct {
	Query    string   `form:"q"`
	Category string   `form:"category"`
	Tags     []string `form:"tags"`
	Page     int      `form:"page"`
	Limit    int      `form:"limit"`
}

// ContactForm demonstrates multipart form with mixed data
type ContactForm struct {
	Name    string         `form:"name"`
	Email   string         `form:"email"`
	Subject string         `form:"subject"`
	Message string         `form:"message"`
	Avatar  *zh.FileHeader `form:"avatar"`
}

// UploadForm demonstrates multiple file uploads
type UploadForm struct {
	Description string           `form:"description"`
	Documents   []*zh.FileHeader `form:"documents"`
}

func main() {
	app := zh.New()

	// Routes
	app.GET("/", zh.HandlerFunc(indexHandler))
	app.POST("/login", zh.HandlerFunc(loginHandler))
	app.GET("/search", zh.HandlerFunc(searchHandler))
	app.GET("/contact", zh.HandlerFunc(contactFormHandler))
	app.POST("/contact", zh.HandlerFunc(contactHandler))
	app.GET("/upload", zh.HandlerFunc(uploadFormHandler))
	app.POST("/upload", zh.HandlerFunc(uploadHandler))

	log.Println("Server starting on :8080")
	log.Println("Try these endpoints:")
	log.Println("  GET  /          - Overview")
	log.Println("  POST /login     - Form binding (application/x-www-form-urlencoded)")
	log.Println("  GET  /search    - Query binding with slices")
	log.Println("  GET  /contact   - Multipart form page")
	log.Println("  POST /contact   - Multipart form with single file")
	log.Println("  GET  /upload    - Multiple file upload page")
	log.Println("  POST /upload    - Multipart form with multiple files")
	log.Fatal(app.Start())
}

func indexHandler(w http.ResponseWriter, r *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Form Binding Examples</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 40px auto; padding: 20px; }
        h1 { color: #333; }
        h2 { color: #555; margin-top: 30px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { margin: 20px 0; padding: 15px; background: #f9f9f9; border-left: 4px solid #007bff; }
        .method { font-weight: bold; color: #007bff; }
    </style>
</head>
<body>
    <h1>Form/Multipart Binding Examples</h1>
    <p>This example demonstrates zerohttp's form binding capabilities using <code>Bind.Form()</code> and <code>Bind.MultipartForm()</code>.</p>

    <h2>Available Endpoints</h2>

    <div class="endpoint">
        <span class="method">POST</span> <code>/login</code>
        <p>Simple form binding with basic types (string, bool).</p>
        <pre>curl -X POST http://localhost:8080/login \
  -d "username=johndoe" \
  -d "password=secret" \
  -d "remember=true"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <code>/search</code>
        <p>Query parameter binding with slices.</p>
        <pre>curl "http://localhost:8080/search?q=golang&category=tech&tags=api&tags=web&page=1&limit=10"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET/POST</span> <code>/contact</code>
        <p>Multipart form with single file upload. Visit in browser or use:</p>
        <pre>curl -X POST http://localhost:8080/contact \
  -F "name=John Doe" \
  -F "email=john@example.com" \
  -F "subject=Hello" \
  -F "message=This is a test" \
  -F "avatar=@/path/to/avatar.png"</pre>
    </div>

    <div class="endpoint">
        <span class="method">GET/POST</span> <code>/upload</code>
        <p>Multiple file upload. Visit in browser or use:</p>
        <pre>curl -X POST http://localhost:8080/upload \
  -F "description=My documents" \
  -F "documents=@file1.pdf" \
  -F "documents=@file2.pdf" \
  -F "documents=@file3.pdf"</pre>
    </div>

    <h2>Features Demonstrated</h2>
    <ul>
        <li><strong>Bind.Form()</strong> - <code>application/x-www-form-urlencoded</code> parsing</li>
        <li><strong>Bind.MultipartForm()</strong> - <code>multipart/form-data</code> with file uploads</li>
        <li><strong>Struct tags</strong> - <code>form:"fieldname"</code> for custom field names</li>
        <li><strong>Type conversion</strong> - Automatic string to int, bool, float conversion</li>
        <li><strong>Slice binding</strong> - Multiple values with same field name</li>
        <li><strong>File uploads</strong> - Single and multiple file handling</li>
    </ul>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	var form LoginForm
	if err := zh.B.Form(r, &form); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	// In a real app, you'd validate credentials here
	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message":  "Login form received",
		"username": form.Username,
		"remember": form.Remember,
		"note":     "Password would be validated in production",
	})
}

func searchHandler(w http.ResponseWriter, r *http.Request) error {
	var form SearchForm
	// Form() parses both POST body and query string
	if err := zh.B.Form(r, &form); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	// Set defaults
	if form.Page < 1 {
		form.Page = 1
	}
	if form.Limit < 1 {
		form.Limit = 20
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"query":    form.Query,
		"category": form.Category,
		"tags":     form.Tags,
		"page":     form.Page,
		"limit":    form.Limit,
	})
}

func contactFormHandler(w http.ResponseWriter, r *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Contact Form</title>
    <style>
        body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 20px; }
        label { display: block; margin: 15px 0 5px; }
        input, textarea { width: 100%; padding: 8px; box-sizing: border-box; }
        button { margin-top: 20px; padding: 10px 20px; }
    </style>
</head>
<body>
    <h1>Contact Form</h1>
    <form method="POST" enctype="multipart/form-data">
        <label>Name: <input type="text" name="name" required></label>
        <label>Email: <input type="email" name="email" required></label>
        <label>Subject: <input type="text" name="subject" required></label>
        <label>Message: <textarea name="message" rows="5" required></textarea></label>
        <label>Avatar: <input type="file" name="avatar" accept="image/*"></label>
        <button type="submit">Submit</button>
    </form>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func contactHandler(w http.ResponseWriter, r *http.Request) error {
	var form ContactForm
	if err := zh.B.MultipartForm(r, &form, 32<<20); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	response := zh.M{
		"message": "Contact form received",
		"data": zh.M{
			"name":    form.Name,
			"email":   form.Email,
			"subject": form.Subject,
			"message": form.Message,
		},
	}

	// Handle optional file upload
	if form.Avatar != nil {
		response["avatar"] = zh.M{
			"filename": form.Avatar.Filename,
			"size":     form.Avatar.Size,
			"header":   form.Avatar.Header,
		}

		// Example: Read file content (in production, save to disk or cloud storage)
		content, err := form.Avatar.ReadAll()
		if err == nil {
			response["avatar"].(zh.M)["content_length"] = len(content)
		}
	}

	return zh.R.JSON(w, http.StatusOK, response)
}

func uploadFormHandler(w http.ResponseWriter, r *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Multi-File Upload</title>
    <style>
        body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 20px; }
        label { display: block; margin: 15px 0 5px; }
        input, textarea { width: 100%; padding: 8px; box-sizing: border-box; }
        button { margin-top: 20px; padding: 10px 20px; }
    </style>
</head>
<body>
    <h1>Multi-File Upload</h1>
    <form method="POST" enctype="multipart/form-data">
        <label>Description: <textarea name="description" rows="3"></textarea></label>
        <label>Documents: <input type="file" name="documents" multiple></label>
        <button type="submit">Upload</button>
    </form>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) error {
	var form UploadForm
	if err := zh.B.MultipartForm(r, &form, 32<<20); err != nil {
		return zh.NewProblemDetail(http.StatusBadRequest, err.Error()).Render(w)
	}

	files := make([]zh.M, len(form.Documents))
	for i, doc := range form.Documents {
		files[i] = zh.M{
			"filename": doc.Filename,
			"size":     doc.Size,
			"header":   doc.Header,
		}
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message":     fmt.Sprintf("Received %d file(s)", len(form.Documents)),
		"description": form.Description,
		"files":       files,
	})
}
