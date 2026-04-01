package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

func main() {
	app := zh.New(
		zh.Config{
			Addr: "localhost:8080",
			TLS: zh.TLSConfig{
				Addr:         "localhost:8443",
				CertFile:     "cert.pem",
				KeyFile:      "key.pem",
				RedirectHTTP: true, // Redirect HTTP to HTTPS
			},
		},
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{
			"message": "Hello, World!",
			"tls":     r.TLS != nil,
		})
	}))

	log.Fatal(app.Start())
}
