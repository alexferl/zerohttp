package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zerohttp.New()

	app.Use(middleware.Compress())

	app.GET("/", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Compression Demo</title></head>
<body>
<h1>Hello, Compressed World!</h1>
<p>This response is automatically compressed if the client supports it.</p>
</body>
</html>`))
		return err
	}))

	app.GET("/api/data", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zerohttp.R.JSON(w, http.StatusOK, map[string]any{
			"message":   "This JSON response is compressed",
			"timestamp": time.Now().Unix(),
		})
	}))

	log.Fatal(app.Start())
}
