package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

var csp = "default-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self'; style-src 'unsafe-inline'; font-src 'self'; frame-ancestors 'self'; form-action 'self';"

type PageData struct {
	Message     string
	Description string
}

func main() {
	tm := NewTemplTemplateManager()

	tm.RegisterComponent("home", ComponentFactory(HomePage))
	tm.RegisterComponent("404", ComponentFactory(NotFoundPage))

	app := zh.New(
		config.WithSecurityHeadersOptions(
			config.WithSecurityHeadersCSP(csp),
		),
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := PageData{
			Message:     "Hello from Templ!",
			Description: "This is a zerohttp example using Templ components.",
		}
		return tm.Render(w, r, http.StatusOK, "home", data)
	}))

	app.GET("/about", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := PageData{
			Message:     "About Our App",
			Description: "Built with zerohttp and Templ for type-safe HTML rendering.",
		}
		return tm.Render(w, r, http.StatusOK, "home", data)
	}))

	app.NotFound(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := PageData{
			Message:     "Page Not Found",
			Description: "The page you requested could not be found.",
		}
		return tm.Render(w, r, http.StatusNotFound, "404", data)
	}))

	log.Fatal(app.Start())
}
