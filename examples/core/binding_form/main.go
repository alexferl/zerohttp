package main

import (
	"fmt"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

type LoginForm struct {
	Username string `form:"username"`
	Password string `form:"password"`
	Remember bool   `form:"remember"`
}

type SearchForm struct {
	Query    string   `form:"q"`
	Category string   `form:"category"`
	Tags     []string `form:"tags"`
	Page     int      `form:"page"`
	Limit    int      `form:"limit"`
}

type ContactForm struct {
	Name    string         `form:"name"`
	Email   string         `form:"email"`
	Subject string         `form:"subject"`
	Message string         `form:"message"`
	Avatar  *zh.FileHeader `form:"avatar"`
}

type UploadForm struct {
	Description string           `form:"description"`
	Documents   []*zh.FileHeader `form:"documents"`
}

func main() {
	app := zh.New()

	app.GET("/", zh.HandlerFunc(indexHandler))
	app.POST("/login", zh.HandlerFunc(loginHandler))
	app.GET("/search", zh.HandlerFunc(searchHandler))
	app.GET("/contact", zh.HandlerFunc(contactFormHandler))
	app.POST("/contact", zh.HandlerFunc(contactHandler))
	app.GET("/upload", zh.HandlerFunc(uploadFormHandler))
	app.POST("/upload", zh.HandlerFunc(uploadHandler))

	log.Fatal(app.Start())
}

func indexHandler(w http.ResponseWriter, _ *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Form Binding Examples</title>
    <style>
        body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 20px; }
        a { display: block; margin: 10px 0; padding: 15px; background: #f4f4f4; border-radius: 5px; text-decoration: none; color: #007bff; }
        a:hover { background: #e9e9e9; }
    </style>
</head>
<body>
    <h1>Form Binding Examples</h1>
    <a href="/contact">Contact Form (single file upload)</a>
    <a href="/upload">Multi-File Upload</a>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	var form LoginForm
	if err := zh.B.Form(r, &form); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message":  "Login form received",
		"username": form.Username,
		"remember": form.Remember,
	})
}

func searchHandler(w http.ResponseWriter, r *http.Request) error {
	var form SearchForm
	if err := zh.B.Form(r, &form); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

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

func contactFormHandler(w http.ResponseWriter, _ *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Contact Form</title>
    <style>
        body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 20px; }
        .field { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, textarea { width: 100%; padding: 8px; box-sizing: border-box; }
        button { padding: 10px 20px; }
    </style>
</head>
<body>
    <h1>Contact Form</h1>
    <form method="POST" enctype="multipart/form-data">
        <div class="field">
            <label>Name:</label>
            <input type="text" name="name" required>
        </div>
        <div class="field">
            <label>Email:</label>
            <input type="email" name="email" required>
        </div>
        <div class="field">
            <label>Subject:</label>
            <input type="text" name="subject" required>
        </div>
        <div class="field">
            <label>Message:</label>
            <textarea name="message" rows="5" required></textarea>
        </div>
        <div class="field">
            <label>Avatar:</label>
            <input type="file" name="avatar" accept="image/*">
        </div>
        <button type="submit">Submit</button>
    </form>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func contactHandler(w http.ResponseWriter, r *http.Request) error {
	var form ContactForm
	if err := zh.B.MultipartForm(r, &form, 32<<20); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
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

	if form.Avatar != nil {
		response["avatar"] = zh.M{
			"filename": form.Avatar.Filename,
			"size":     form.Avatar.Size,
		}

		content, err := form.Avatar.ReadAll()
		if err == nil {
			response["avatar"].(zh.M)["content_length"] = len(content)
		}
	}

	return zh.R.JSON(w, http.StatusOK, response)
}

func uploadFormHandler(w http.ResponseWriter, _ *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Multi-File Upload</title>
    <style>
        body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 20px; }
        .field { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, textarea { width: 100%; padding: 8px; box-sizing: border-box; }
        button { padding: 10px 20px; }
    </style>
</head>
<body>
    <h1>Multi-File Upload</h1>
    <form method="POST" enctype="multipart/form-data">
        <div class="field">
            <label>Description:</label>
            <textarea name="description" rows="3"></textarea>
        </div>
        <div class="field">
            <label>Documents:</label>
            <input type="file" name="documents" multiple>
        </div>
        <button type="submit">Upload</button>
    </form>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) error {
	var form UploadForm
	if err := zh.B.MultipartForm(r, &form, 32<<20); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(http.StatusBadRequest, err.Error()))
	}

	files := make([]zh.M, len(form.Documents))
	for i, doc := range form.Documents {
		files[i] = zh.M{
			"filename": doc.Filename,
			"size":     doc.Size,
		}
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message":     fmt.Sprintf("Received %d file(s)", len(form.Documents)),
		"description": form.Description,
		"files":       files,
	})
}
