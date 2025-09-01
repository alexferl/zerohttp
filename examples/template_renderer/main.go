package main

import (
	"embed"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

//go:embed templates/*.html
var templatesFS embed.FS

type PageData struct {
	Title       string
	Message     string
	Description string
}

func main() {
	tm := zh.NewTemplateManager(templatesFS, "templates/*.html")

	app := zh.New()

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := PageData{
			Title:       "Welcome",
			Message:     "Hello, World!",
			Description: "This is a simple zerohttp example app.",
		}
		return tm.Render(w, http.StatusOK, "index.html", data)
	}))

	app.GET("/about", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := PageData{
			Title:       "About",
			Message:     "About Our App",
			Description: "Built with zerohttp framework.",
		}
		return tm.Render(w, http.StatusOK, "index.html", data)
	}))

	app.NotFound(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := PageData{
			Title:       "404 - Page Not Found",
			Message:     "Page Not Found",
			Description: "The page you're looking for doesn't exist.",
		}
		return tm.Render(w, http.StatusNotFound, "404.html", data)
	}))

	log.Fatal(app.Start())
}
