package main

import (
	"embed"
	"io"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/flosch/pongo2/v6"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Pongo2Renderer struct {
	set *pongo2.TemplateSet
}

func NewPongo2Renderer(templateFS embed.FS) *Pongo2Renderer {
	// Create a custom loader for embed.FS
	loader := &EmbedFSLoader{fs: templateFS}
	set := pongo2.NewSet("embedded", loader)
	return &Pongo2Renderer{set: set}
}

func (pr *Pongo2Renderer) Render(w http.ResponseWriter, status int, name string, ctx pongo2.Context) error {
	tpl, err := pr.set.FromFile(name)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	return tpl.ExecuteWriter(ctx, w)
}

// Custom loader for embed.FS
type EmbedFSLoader struct {
	fs embed.FS
}

func (loader *EmbedFSLoader) Get(path string) (io.Reader, error) {
	return loader.fs.Open("templates/" + path)
}

func (loader *EmbedFSLoader) Abs(base, name string) string {
	return name
}

func main() {
	renderer := NewPongo2Renderer(templatesFS)

	app := zh.New()
	app.Use()

	// Home route
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ctx := pongo2.Context{
			"title":       "Welcome",
			"message":     "Hello from Pongo2!",
			"description": "This is a zerohttp example using Pongo2 templates.",
		}
		return renderer.Render(w, http.StatusOK, "index.html", ctx)
	}))

	// About route
	app.GET("/about", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ctx := pongo2.Context{
			"title":       "About",
			"message":     "About Our App",
			"description": "Built with zerohttp and Pongo2 templates.",
		}
		return renderer.Render(w, http.StatusOK, "about.html", ctx)
	}))

	// 404 handler
	app.NotFound(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ctx := pongo2.Context{
			"title":       "404 - Not Found",
			"message":     "Page Not Found",
			"description": "The page you requested could not be found.",
		}
		return renderer.Render(w, http.StatusNotFound, "404.html", ctx)
	}))

	log.Println("Server starting on :8080")
	log.Fatal(app.Start())
}
