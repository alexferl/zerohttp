package hmacauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

// BenchmarkHMAC_SignRequest measures HMAC request signing performance
func BenchmarkHMAC_SignRequest(b *testing.B) {
	algorithms := []HashAlgorithm{
		SHA256,
		SHA384,
		SHA512,
	}

	for _, algo := range algorithms {
		b.Run(string(algo), func(b *testing.B) {
			secret := strings.Repeat("a", 64) // 64 bytes for SHA512
			signer := NewSignerWithAlgorithm("access-key", secret, algo)

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Host = "example.com"

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				// Need fresh request each time since body is consumed
				r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
				r.Host = "example.com"
				_ = signer.SignRequest(r)
			}
		})
	}
}

// BenchmarkHMAC_VerifySignature measures signature verification performance
func BenchmarkHMAC_VerifySignature(b *testing.B) {
	algorithms := []HashAlgorithm{
		SHA256,
		SHA384,
		SHA512,
	}

	for _, algo := range algorithms {
		b.Run(string(algo), func(b *testing.B) {
			secret := strings.Repeat("a", 64)
			signer := NewSignerWithAlgorithm("access-key", secret, algo)

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Host = "example.com"
			_ = signer.SignRequest(req)

			authHeader := req.Header.Get(httpx.HeaderAuthorization)
			timestamp := req.Header.Get("X-Timestamp")

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				// Verify by computing signature
				_ = authHeader
				canonical := "GET\n/api/test\n\nhost:example.com\nx-timestamp:" + timestamp + "\n\nUNSIGNED-PAYLOAD"
				_ = computeHMACSignature(secret, canonical, algo)
			}
		})
	}
}

// BenchmarkHMAC_AuthMiddleware measures the full HMAC auth middleware
func BenchmarkHMAC_AuthMiddleware(b *testing.B) {
	secret := strings.Repeat("a", 32)

	credentialStore := func(accessKeyID string) []string {
		if accessKeyID == "test-key" {
			return []string{secret}
		}
		return nil
	}

	handler := New(Config{
		CredentialStore: credentialStore,
		Algorithm:       SHA256,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewSigner("test-key", secret)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Host = "example.com"
	_ = signer.SignRequest(req)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkHMAC_AuthMiddleware_Scenarios measures different HMAC auth scenarios
func BenchmarkHMAC_AuthMiddleware_Scenarios(b *testing.B) {
	secret := strings.Repeat("a", 32)

	credentialStore := func(accessKeyID string) []string {
		if accessKeyID == "test-key" {
			return []string{secret}
		}
		return nil
	}

	scenarios := []struct {
		name         string
		setupRequest func(*http.Request)
	}{
		{
			name: "Valid",
			setupRequest: func(req *http.Request) {
				signer := NewSigner("test-key", secret)
				req.Host = "example.com"
				_ = signer.SignRequest(req)
			},
		},
		{
			name: "MissingAuth",
			setupRequest: func(req *http.Request) {
				// Don't add auth header
			},
		},
		{
			name: "InvalidSignature",
			setupRequest: func(req *http.Request) {
				signer := NewSigner("test-key", secret)
				req.Host = "example.com"
				_ = signer.SignRequest(req)
				req.Header.Set(httpx.HeaderAuthorization, req.Header.Get(httpx.HeaderAuthorization)+"tampered")
			},
		},
		{
			name: "ExpiredTimestamp",
			setupRequest: func(req *http.Request) {
				req.Header.Set(httpx.HeaderAuthorization, "HMAC-SHA256 Credential=test-key/2020-01-01T00:00:00Z, SignedHeaders=host;x-timestamp, Signature=aW52YWxpZA==")
				req.Header.Set(httpx.HeaderXTimestamp, "2020-01-01T00:00:00Z")
			},
		},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			handler := New(Config{
				CredentialStore: credentialStore,
				Algorithm:       SHA256,
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
				s.setupRequest(req)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkHMAC_PresignedURL measures presigned URL generation and validation
func BenchmarkHMAC_PresignedURL(b *testing.B) {
	b.Run("Generate", func(b *testing.B) {
		secret := strings.Repeat("a", 32)
		signer := NewSigner("test-key", secret)

		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		req.Host = "example.com"

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			r := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
			r.Host = "example.com"
			_, _ = signer.PresignURL(r, time.Hour)
		}
	})

	b.Run("Validate", func(b *testing.B) {
		secret := strings.Repeat("a", 32)
		signer := NewSigner("test-key", secret)

		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		req.Host = "example.com"
		presignedURL, _ := signer.PresignURL(req, time.Hour)

		parsedReq := httptest.NewRequest(http.MethodGet, presignedURL, nil)
		parsedReq.Host = "example.com"

		credentialStore := func(accessKeyID string) []string {
			if accessKeyID == "test-key" {
				return []string{secret}
			}
			return nil
		}

		handler := New(Config{
			CredentialStore:    credentialStore,
			Algorithm:          SHA256,
			AllowPresignedURLs: true,
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, parsedReq)
		}
	})
}

// BenchmarkHMAC_CanonicalRequest measures canonical request building
func BenchmarkHMAC_CanonicalRequest(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Host = "example.com"

		parsed := &parsedAuth{
			Headers: []string{"host", "x-timestamp"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = buildCanonicalRequest(req, parsed, nil, nil, "UNSIGNED-PAYLOAD", "X-Timestamp")
		}
	})

	b.Run("WithQueryParams", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/api/test?foo=bar&baz=qux", nil)
		req.Host = "example.com"

		parsed := &parsedAuth{
			Headers: []string{"host", "x-timestamp"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = buildCanonicalRequest(req, parsed, nil, nil, "UNSIGNED-PAYLOAD", "X-Timestamp")
		}
	})

	b.Run("WithBody", func(b *testing.B) {
		parsed := &parsedAuth{
			Headers: []string{"host", "x-timestamp", "content-type"},
		}

		bodyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			body := strings.NewReader(`{"key":"value"}`)
			r := httptest.NewRequest(http.MethodPost, "/api/test", body)
			r.Host = "example.com"
			r.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
			_ = buildCanonicalRequest(r, parsed, nil, nil, bodyHash, "X-Timestamp")
		}
	})
}

// BenchmarkHMAC_ParseAuthorizationHeader measures authorization header parsing
func BenchmarkHMAC_ParseAuthorizationHeader(b *testing.B) {
	header := "HMAC-SHA256 Credential=test-key/2024-01-15T10:30:00Z, SignedHeaders=host;x-timestamp, Signature=AbCdEf123="

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = parseAuthorizationHeader(header, "X-Timestamp")
	}
}
