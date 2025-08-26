package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

func main() {
	app := zh.New(
		config.WithAddr(":8080"),
		config.WithTLSAddr(":8443"),
		config.WithCertFile("cert.pem"),
		config.WithKeyFile("key.pem"),
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{
			"message": "Hello, World!",
			"tls":     r.TLS != nil,
		})
	}))

	log.Fatal(app.Start())
}
