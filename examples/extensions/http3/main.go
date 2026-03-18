package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/quic-go/quic-go/http3"
)

func main() {
	certFile, keyFile := "localhost+2.pem", "localhost+2-key.pem"

	app := zh.New(
		config.Config{
			TLS: config.TLSConfig{
				Addr:     ":8443",
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
	)

	// Add Alt-Svc header middleware to advertise HTTP/3
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(httpx.HeaderAltSvc, `h3=":8443"; ma=86400`)
			next.ServeHTTP(w, r)
		})
	})

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, err := w.Write([]byte("Hello over HTTP/3!\n"))
		return err
	}))

	// Create HTTP/3 server using quic-go
	h3Server := &http3.Server{
		Addr:    ":8443",
		Handler: app,
	}
	app.SetHTTP3Server(h3Server)

	// Start HTTPS server - HTTP/3 starts automatically!
	log.Fatal(app.StartTLS(certFile, keyFile))
}
