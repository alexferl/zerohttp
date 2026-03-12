package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkCSRF_TokenGeneration measures token generation performance
func BenchmarkCSRF_TokenGeneration(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = generateToken(key)
	}
}

// BenchmarkCSRF_TokenValidation measures token validation/comparison
func BenchmarkCSRF_TokenValidation(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")
	token, _ := generateToken(key)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = compareTokens(token, token, key)
	}
}

// BenchmarkCSRF_TokenComparisonScenarios different validation scenarios
func BenchmarkCSRF_TokenComparisonScenarios(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")
	validToken, _ := generateToken(key)
	invalidToken, _ := generateToken(key)

	b.Run("ValidToken", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = compareTokens(validToken, validToken, key)
		}
	})

	b.Run("InvalidToken", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = compareTokens(validToken, invalidToken, key)
		}
	})

	b.Run("MalformedToken", func(b *testing.B) {
		malformed := "not-a-valid-token"
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = compareTokens(validToken, malformed, key)
		}
	})
}

// BenchmarkCSRF_ExtractToken measures token extraction from different sources
func BenchmarkCSRF_ExtractToken(b *testing.B) {
	b.Run("Header", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-CSRF-Token", "test-token-value")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = extractToken(req, "header", "X-CSRF-Token")
		}
	})

	b.Run("Query", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/?csrf_token=test-token-value", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = extractToken(req, "query", "csrf_token")
		}
	})

	b.Run("Form", func(b *testing.B) {
		formData := url.Values{"csrf_token": []string{"test-token-value"}}
		body := strings.NewReader(formData.Encode())
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = extractToken(req, "form", "csrf_token")
		}
	})
}

// BenchmarkCSRF_Middleware measures full middleware overhead
func BenchmarkCSRF_Middleware(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")

	b.Run("GET_ExemptMethod", func(b *testing.B) {
		handler := CSRF(config.CSRFConfig{
			HMACKey:       key,
			ExemptMethods: []string{http.MethodGet},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("GET_NewToken", func(b *testing.B) {
		handler := CSRF(config.CSRFConfig{
			HMACKey:       key,
			ExemptMethods: []string{}, // GET is not exempt, will generate token
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("POST_ValidToken", func(b *testing.B) {
		handler := CSRF(config.CSRFConfig{HMACKey: key})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Pre-generate a valid token
		token, _ := generateToken(key)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("X-CSRF-Token", token)
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("POST_InvalidToken", func(b *testing.B) {
		handler := CSRF(config.CSRFConfig{HMACKey: key})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		validToken, _ := generateToken(key)
		invalidToken, _ := generateToken(key)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("X-CSRF-Token", invalidToken)
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: validToken})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("POST_MissingToken", func(b *testing.B) {
		handler := CSRF(config.CSRFConfig{HMACKey: key})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkCSRF_TokenFormatValidation measures token format validation
func BenchmarkCSRF_TokenFormatValidation(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")
	validToken, _ := generateToken(key)

	cases := []struct {
		name  string
		token string
	}{
		{"Valid", validToken},
		{"Empty", ""},
		{"InvalidBase64", "!!!not-base64!!!"},
		{"TooShort", base64.RawURLEncoding.EncodeToString([]byte("short"))},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_ = validateTokenFormat(tc.token)
			}
		})
	}
}

// BenchmarkCSRF_ParseTokenLookup measures token lookup parsing
func BenchmarkCSRF_ParseTokenLookup(b *testing.B) {
	lookups := []string{
		"header:X-CSRF-Token",
		"form:csrf_token",
		"query:csrf_token",
		"invalid", // falls back to default
	}

	for _, lookup := range lookups {
		name := strings.ReplaceAll(lookup, ":", "_")
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = parseTokenLookup(lookup)
			}
		})
	}
}

// BenchmarkCSRF_SetCookie measures cookie setting overhead
func BenchmarkCSRF_SetCookie(b *testing.B) {
	cfg := config.CSRFConfig{
		CookieName:     "csrf_token",
		CookieMaxAge:   86400,
		CookiePath:     "/",
		CookieDomain:   "",
		CookieSecure:   boolPtr(true),
		CookieSameSite: http.SameSiteStrictMode,
	}
	token := "test-csrf-token-value"

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		w := httptest.NewRecorder()
		setCSRFCookie(w, cfg, token)
	}
}

// BenchmarkCSRF_ExemptPaths measures exempt path checking
func BenchmarkCSRF_ExemptPaths(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")
	exemptPaths := []string{"/api/*", "/health", "/static/*"}

	handler := CSRF(config.CSRFConfig{
		HMACKey:     key,
		ExemptPaths: exemptPaths,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	b.Run("ExemptPath", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/health", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("NonExemptPath", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkCSRF_Concurrent measures concurrent token operations
func BenchmarkCSRF_Concurrent(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")

	b.Run("TokenGeneration", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = generateToken(key)
			}
		})
	})

	b.Run("TokenValidation", func(b *testing.B) {
		token, _ := generateToken(key)

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = compareTokens(token, token, key)
			}
		})
	})

	b.Run("Middleware_GET", func(b *testing.B) {
		handler := CSRF(config.CSRFConfig{
			HMACKey:       key,
			ExemptMethods: []string{http.MethodGet},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
			}
		})
	})

	b.Run("Middleware_POST_Valid", func(b *testing.B) {
		handler := CSRF(config.CSRFConfig{HMACKey: key})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		token, _ := generateToken(key)

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.Header.Set("X-CSRF-Token", token)
				req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
			}
		})
	})
}

// BenchmarkCSRF_HMACComparison compares HMAC operations at different levels
func BenchmarkCSRF_HMACComparison(b *testing.B) {
	key := []byte("test-key-32-bytes-long-for-hmac!!")
	data := make([]byte, 32)
	_, _ = rand.Read(data)

	b.Run("HMAC_Generate", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			mac := hmac.New(sha256.New, key)
			mac.Write(data)
			_ = mac.Sum(nil)
		}
	})

	b.Run("HMAC_Verify", func(b *testing.B) {
		mac := hmac.New(sha256.New, key)
		mac.Write(data)
		signature := mac.Sum(nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			mac2 := hmac.New(sha256.New, key)
			mac2.Write(data)
			expected := mac2.Sum(nil)
			_ = subtle.ConstantTimeCompare(signature, expected)
		}
	})

	b.Run("Base64_Encode", func(b *testing.B) {
		combined := append(data, make([]byte, sha256.Size)...)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = base64.RawURLEncoding.EncodeToString(combined)
		}
	})

	b.Run("Base64_Decode", func(b *testing.B) {
		combined := append(data, make([]byte, sha256.Size)...)
		encoded := base64.RawURLEncoding.EncodeToString(combined)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_, _ = base64.RawURLEncoding.DecodeString(encoded)
		}
	})
}

// boolPtr is a helper to get a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}
