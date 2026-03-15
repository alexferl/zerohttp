package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

//go:embed templates/*.html
var templatesFS embed.FS

func main() {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.html"))

	app := zh.New()

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := zh.M{"Title": "Home", "Message": "Hello, World!"}
		return zh.R.Template(w, 200, tmpl, "index.html", data)
	}))

	app.NotFound(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := zh.M{"Title": "404", "Message": "Page Not Found"}
		return zh.R.Template(w, 404, tmpl, "404.html", data)
	}))

	log.Fatal(app.Start())
}
