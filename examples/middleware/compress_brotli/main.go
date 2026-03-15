package main

import (
	"io"
	"log"
	"net/http"

	"github.com/andybalholm/brotli"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

// BrotliEncoder implements config.CompressionEncoder
type BrotliEncoder struct{}

func (e BrotliEncoder) Encode(w io.Writer, level int) io.Writer {
	// brotli levels are 0-11, map gzip 1-9 to brotli range
	if level < 0 {
		level = 4
	} else if level > 11 {
		level = 11
	}
	return brotli.NewWriterLevel(w, level)
}

func (e BrotliEncoder) Encoding() string {
	return "br"
}

// BrotliProvider implements config.CompressionProvider
type BrotliProvider struct{}

func (p BrotliProvider) GetEncoder(encoding string) config.CompressionEncoder {
	if encoding == "br" {
		return BrotliEncoder{}
	}
	return nil
}

func main() {
	app := zerohttp.New()

	app.Use(middleware.Compress(config.CompressConfig{
		Level:      6,
		Algorithms: []config.CompressionAlgorithm{"br", config.Gzip, config.Deflate},
		Provider:   BrotliProvider{},
	}))

	app.GET("/", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Brotli Compression Demo</title></head>
<body>
<h1>Hello, Brotli Compressed World!</h1>
<p>Brotli typically provides 20-26% better compression than gzip.</p>
</body>
</html>`))
		return err
	}))

	log.Fatal(app.Start())
}
