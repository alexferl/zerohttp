package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Apply RealIP middleware to extract client IP from proxy headers
	app.Use(middleware.RealIP())

	// This endpoint shows the extracted real client IP
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message":     "Hello!",
			"remote_addr": r.RemoteAddr,
			"real_ip":     r.RemoteAddr,
		})
	}))

	// This endpoint uses a custom IP extractor (X-Real-IP only)
	app.GET("/nginx", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message":     "Nginx style IP extraction",
			"remote_addr": r.RemoteAddr,
		})
	}),
		middleware.RealIP(config.RealIPConfig{
			IPExtractor: config.XRealIPExtractor,
		}),
	)

	// This endpoint uses RemoteAddr only (no proxy headers)
	app.GET("/direct", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message":     "Direct connection (no proxy headers)",
			"remote_addr": r.RemoteAddr,
		})
	}),
		middleware.RealIP(config.RealIPConfig{
			IPExtractor: config.RemoteAddrIPExtractor,
		}),
	)

	log.Fatal(app.Start())
}
