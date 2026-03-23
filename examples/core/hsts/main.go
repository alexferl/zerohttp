package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/securityheaders"
)

func main() {
	app := zh.New(
		zh.Config{
			Addr: "localhost:8080",
			TLS: zh.TLSConfig{
				Addr:     "localhost:8443",
				CertFile: "cert.pem",
				KeyFile:  "key.pem",
			},
			// Enable HSTS with custom configuration
			SecurityHeaders: securityheaders.Config{
				StrictTransportSecurity: securityheaders.StrictTransportSecurity{
					MaxAge:            31536000, // 1 year in seconds
					ExcludeSubdomains: false,    // Apply to subdomains too
					PreloadEnabled:    false,    // Set to true only if submitting to hstspreload.org
				},
			},
		},
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message": "HSTS enabled!",
			"tls":     r.TLS != nil,
		})
	}))

	log.Fatal(app.Start())
}
