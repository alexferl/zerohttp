package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkHMACSigner_BuildCanonicalRequest measures canonical request building
func BenchmarkHMACSigner_BuildCanonicalRequest(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes-long!!")
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Host = "example.com"
		signedHeaders := []string{"host", "x-timestamp"}
		bodyHash := "UNSIGNED-PAYLOAD"

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = signer.buildCanonicalRequest(req, signedHeaders, bodyHash)
		}
	})

	b.Run("WithQueryParams", func(b *testing.B) {
		signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes-long!!")
		req := httptest.NewRequest(http.MethodGet, "/api/test?foo=bar&baz=qux", nil)
		req.Host = "example.com"
		signedHeaders := []string{"host", "x-timestamp"}
		bodyHash := "UNSIGNED-PAYLOAD"

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = signer.buildCanonicalRequest(req, signedHeaders, bodyHash)
		}
	})

	b.Run("WithContentType", func(b *testing.B) {
		signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes-long!!")
		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		req.Host = "example.com"
		req.Header.Set("Content-Type", "application/json")
		signedHeaders := []string{"host", "x-timestamp", "content-type"}
		bodyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = signer.buildCanonicalRequest(req, signedHeaders, bodyHash)
		}
	})
}

// BenchmarkHMACSigner_GenerateSignature measures full signature generation
func BenchmarkHMACSigner_GenerateSignature(b *testing.B) {
	tests := []struct {
		name   string
		algo   config.HMACHashAlgorithm
		secret string
	}{
		{"SHA256", config.HMACSHA256, "test-secret-that-is-32-bytes-long!!"},
		{"SHA384", config.HMACSHA384, "test-secret-that-is-48-bytes-long-for-sha384-algo!"},
		{"SHA512", config.HMACSHA512, "test-secret-that-is-64-bytes-long-for-the-sha512-algorithm-use!!"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			signer := NewHMACSignerWithAlgorithm("test-key", tt.secret, tt.algo)
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Host = "example.com"

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
				r.Host = "example.com"
				_, _ = signer.GenerateSignature(r, time.Now().UTC())
			}
		})
	}
}
