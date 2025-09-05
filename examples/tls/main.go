package main

import (
	"log"
	"net"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

func httpsRedirectMiddleware(httpsPort string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil {
				host, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					// No port in Host, use as-is
					host = r.Host
				}

				target := "https://" + host + ":" + httpsPort + r.RequestURI
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	app := zh.New(
		config.WithAddr("localhost:8080"),
		config.WithTLSAddr("localhost:8443"),
		config.WithCertFile("cert.pem"),
		config.WithKeyFile("key.pem"),
	)

	// Add redirect middleware with custom HTTPS port
	app.Use(httpsRedirectMiddleware("8443"))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{
			"message": "Hello, World!",
			"tls":     r.TLS != nil,
		})
	}))

	log.Fatal(app.Start())
}
