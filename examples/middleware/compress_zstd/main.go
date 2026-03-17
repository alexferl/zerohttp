package main

import (
	"io"
	"log"
	"net/http"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/klauspost/compress/zstd"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

// ZstdEncoder implements config.CompressionEncoder
type ZstdEncoder struct{}

func (e ZstdEncoder) Encode(w io.Writer, level int) io.Writer {
	// zstd levels are 1-22 (SpeedFastest to SpeedBest)
	// Map standard 1-9 to zstd range
	zstdLevel := zstd.SpeedDefault
	switch {
	case level <= 1:
		zstdLevel = zstd.SpeedFastest
	case level <= 3:
		zstdLevel = zstd.SpeedDefault
	case level <= 6:
		zstdLevel = zstd.SpeedBetterCompression
	default:
		zstdLevel = zstd.SpeedBestCompression
	}

	encoder, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstdLevel))
	if err != nil {
		// Fall back to default on error
		encoder, _ = zstd.NewWriter(w)
	}
	return encoder
}

func (e ZstdEncoder) Encoding() string {
	return "zstd"
}

// ZstdProvider implements config.CompressionProvider
type ZstdProvider struct{}

func (p ZstdProvider) GetEncoder(encoding string) config.CompressionEncoder {
	if encoding == "zstd" {
		return ZstdEncoder{}
	}
	return nil
}

func main() {
	app := zerohttp.New()

	app.Use(middleware.Compress(config.CompressConfig{
		Level:      6,
		Algorithms: []config.CompressionAlgorithm{"zstd", config.Gzip, config.Deflate},
		Provider:   ZstdProvider{},
	}))

	app.GET("/", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Zstd Compression Demo</title></head>
<body>
<h1>Hello, Zstd Compressed World!</h1>
<p>Zstd provides excellent compression ratios with very fast decompression.</p>
</body>
</html>`))
		return err
	}))

	log.Fatal(app.Start())
}
