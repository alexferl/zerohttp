package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkBasicAuth_Methods compares different credential validation methods
func BenchmarkBasicAuth_Methods(b *testing.B) {
	b.Run("CredentialsMap", func(b *testing.B) {
		handler := BasicAuth(config.BasicAuthConfig{
			Credentials: map[string]string{
				"admin": "secret123",
				"user":  "password456",
			},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.SetBasicAuth("admin", "secret123")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})

	b.Run("ValidatorFunction", func(b *testing.B) {
		handler := BasicAuth(config.BasicAuthConfig{
			Validator: func(user, pass string) bool {
				return user == "admin" && pass == "secret123"
			},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.SetBasicAuth("admin", "secret123")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkBasicAuth_ConstantTimeComparison measures the overhead of constant-time comparison
func BenchmarkBasicAuth_ConstantTimeComparison(b *testing.B) {
	correctPass := "secret123"
	testPass := "secret123"
	wrongPass := "wrongpass"

	b.Run("StandardComparison", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = testPass == correctPass
		}
	})

	b.Run("ConstantTimeCompare_Correct", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			subtle.ConstantTimeCompare([]byte(testPass), []byte(correctPass))
		}
	})

	b.Run("ConstantTimeCompare_Wrong", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			subtle.ConstantTimeCompare([]byte(wrongPass), []byte(correctPass))
		}
	})
}

// BenchmarkBasicAuth_Scenarios measures different auth scenarios
func BenchmarkBasicAuth_Scenarios(b *testing.B) {
	scenarios := []struct {
		name   string
		user   string
		pass   string
		expect int
	}{
		{"Valid", "admin", "secret123", http.StatusOK},
		{"InvalidPassword", "admin", "wrongpass", http.StatusUnauthorized},
		{"InvalidUser", "unknown", "secret123", http.StatusUnauthorized},
		{"MissingAuth", "", "", http.StatusUnauthorized},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			handler := BasicAuth(config.BasicAuthConfig{
				Credentials: map[string]string{
					"admin": "secret123",
				},
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if s.user != "" {
				req.SetBasicAuth(s.user, s.pass)
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

// BenchmarkBasicAuth_UserCount measures performance with different user counts
func BenchmarkBasicAuth_UserCount(b *testing.B) {
	userCounts := []int{1, 10, 100}

	for _, count := range userCounts {
		b.Run(fmt.Sprintf("Users%d", count), func(b *testing.B) {
			creds := make(map[string]string, count)
			for i := range count {
				creds[fmt.Sprintf("user%d", i)] = fmt.Sprintf("pass%d", i)
			}

			handler := BasicAuth(config.BasicAuthConfig{
				Credentials: creds,
			})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.SetBasicAuth("user0", "pass0")

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}
