package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkJWT_HS256_Generate measures token generation performance
func BenchmarkJWT_HS256_Generate(b *testing.B) {
	store := NewHS256TokenStore([]byte("this-is-a-32-byte-secret-key-for-tests!!"), HS256Options{
		Issuer: "test-app",
	})

	claims := HS256Claims{
		"sub":   "user123",
		"name":  "Test User",
		"email": "test@example.com",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = store.Generate(claims, config.AccessToken)
	}
}

// BenchmarkJWT_HS256_Validate measures token validation performance
func BenchmarkJWT_HS256_Validate(b *testing.B) {
	store := NewHS256TokenStore([]byte("this-is-a-32-byte-secret-key-for-tests!!"), HS256Options{
		Issuer: "test-app",
	})

	claims := HS256Claims{
		"sub":   "user123",
		"name":  "Test User",
		"email": "test@example.com",
	}

	token, _ := store.Generate(claims, config.AccessToken)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = store.Validate(token)
	}
}

// BenchmarkJWT_HS256_ClaimSizes measures performance with different claim sizes
func BenchmarkJWT_HS256_ClaimSizes(b *testing.B) {
	store := NewHS256TokenStore([]byte("this-is-a-32-byte-secret-key-for-tests!!"), HS256Options{})

	sizes := []struct {
		name   string
		claims HS256Claims
	}{
		{
			name:   "Minimal",
			claims: HS256Claims{"sub": "user123"},
		},
		{
			name: "Small",
			claims: HS256Claims{
				"sub":   "user123",
				"name":  "Test User",
				"email": "test@example.com",
			},
		},
		{
			name: "Medium",
			claims: HS256Claims{
				"sub":         "user123",
				"name":        "Test User",
				"email":       "test@example.com",
				"role":        "admin",
				"org":         "acme",
				"department":  "engineering",
				"permissions": []string{"read", "write", "delete"},
			},
		},
		{
			name: "Large",
			claims: func() HS256Claims {
				c := HS256Claims{
					"sub":   "user123",
					"name":  "Test User",
					"email": "test@example.com",
				}
				for i := range 20 {
					c[fmt.Sprintf("custom_claim_%d", i)] = fmt.Sprintf("value_%d", i)
				}
				return c
			}(),
		},
	}

	for _, s := range sizes {
		b.Run(s.name+"/Generate", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = store.Generate(s.claims, config.AccessToken)
			}
		})

		token, _ := store.Generate(s.claims, config.AccessToken)

		b.Run(s.name+"/Validate", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = store.Validate(token)
			}
		})
	}
}

// BenchmarkJWT_AuthMiddleware measures the full JWT middleware
func BenchmarkJWT_AuthMiddleware(b *testing.B) {
	store := NewHS256TokenStore([]byte("this-is-a-32-byte-secret-key-for-tests!!"), HS256Options{})

	handler := JWTAuth(config.JWTAuthConfig{
		TokenStore:     store,
		RequiredClaims: []string{"sub"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims := HS256Claims{"sub": "user123", "name": "Test User"}
	token, _ := store.Generate(claims, config.AccessToken)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkJWT_AuthMiddleware_Scenarios measures different JWT auth scenarios
func BenchmarkJWT_AuthMiddleware_Scenarios(b *testing.B) {
	store := NewHS256TokenStore([]byte("this-is-a-32-byte-secret-key-for-tests!!"), HS256Options{})

	validClaims := HS256Claims{"sub": "user123", "name": "Test User"}
	validToken, _ := store.Generate(validClaims, config.AccessToken)

	scenarios := []struct {
		name  string
		token string
	}{
		{"Valid", validToken},
		{"Missing", ""},
		{"InvalidFormat", "invalid-token"},
		{"InvalidSignature", validToken[:len(validToken)-10] + "tampered!!"},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			handler := JWTAuth(config.JWTAuthConfig{
				TokenStore: store,
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if s.token != "" {
				req.Header.Set("Authorization", "Bearer "+s.token)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkJWT_ClaimsExtraction measures claims extraction performance
func BenchmarkJWT_ClaimsExtraction(b *testing.B) {
	store := NewHS256TokenStore([]byte("this-is-a-32-byte-secret-key-for-tests!!"), HS256Options{
		Issuer:   "test-app",
		Audience: "test-api",
	})

	claims := HS256Claims{
		"sub":   "user123",
		"iss":   "test-app",
		"aud":   "test-api",
		"jti":   "token-id-123",
		"scope": "read write delete",
	}

	token, _ := store.Generate(claims, config.AccessToken)
	validatedClaims, _ := store.Validate(token)

	jwtClaims := JWTClaims{claims: validatedClaims}

	b.Run("Subject", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = jwtClaims.Subject()
		}
	})

	b.Run("Issuer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = jwtClaims.Issuer()
		}
	})

	b.Run("Audience", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = jwtClaims.Audience()
		}
	})

	b.Run("Scopes", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = jwtClaims.Scopes()
		}
	})

	b.Run("HasScope", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = jwtClaims.HasScope("write")
		}
	})
}

// BenchmarkJWT_RequiredClaims measures required claims validation overhead
func BenchmarkJWT_RequiredClaims(b *testing.B) {
	store := NewHS256TokenStore([]byte("this-is-a-32-byte-secret-key-for-tests!!"), HS256Options{})

	claims := HS256Claims{
		"sub":  "user123",
		"name": "Test User",
		"org":  "acme",
	}
	token, _ := store.Generate(claims, config.AccessToken)

	requiredClaimsCounts := []int{1, 3, 5}

	for _, count := range requiredClaimsCounts {
		b.Run(fmt.Sprintf("Claims%d", count), func(b *testing.B) {
			required := make([]string, count)
			for i := range count {
				required[i] = []string{"sub", "name", "org", "email", "role"}[i]
			}

			handler := JWTAuth(config.JWTAuthConfig{
				TokenStore:     store,
				RequiredClaims: required,
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}
