package cors

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestCORSConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 1, len(cfg.AllowedOrigins))
	zhtest.AssertEqual(t, "*", cfg.AllowedOrigins[0])
	zhtest.AssertEqual(t, 7, len(cfg.AllowedMethods))
	zhtest.AssertEqual(t, 5, len(cfg.AllowedHeaders))
	zhtest.AssertEqual(t, 0, len(cfg.ExposedHeaders))
	zhtest.AssertFalse(t, cfg.AllowCredentials)
	zhtest.AssertEqual(t, 86400, cfg.MaxAge)
	zhtest.AssertFalse(t, cfg.OptionsPassthrough)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

	// Test default method values
	expectedMethods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions}
	zhtest.AssertEqual(t, expectedMethods, cfg.AllowedMethods)

	// Test default header values
	expectedHeaders := []string{httpx.HeaderAccept, httpx.HeaderAuthorization, httpx.HeaderContentType, httpx.HeaderXCSRFToken, httpx.HeaderXRequestId}
	zhtest.AssertEqual(t, expectedHeaders, cfg.AllowedHeaders)
}

func TestCORSConfig_StructAssignment(t *testing.T) {
	t.Run("allowed origins", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []string
			expected []string
		}{
			{"multiple origins", []string{"https://example.com", "https://api.example.com", "http://localhost:3000"}, []string{"https://example.com", "https://api.example.com", "http://localhost:3000"}},
			{"wildcard origin", []string{"*"}, []string{"*"}},
			{"origin patterns", []string{"https://*.example.com", "http://localhost:*", "https://subdomain.*.com", "*"}, []string{"https://*.example.com", "http://localhost:*", "https://subdomain.*.com", "*"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := Config{
					AllowedOrigins: tt.input,
					AllowedMethods: DefaultConfig.AllowedMethods,
					AllowedHeaders: DefaultConfig.AllowedHeaders,
				}
				zhtest.AssertEqual(t, len(tt.expected), len(cfg.AllowedOrigins))
				zhtest.AssertEqual(t, tt.expected, cfg.AllowedOrigins)
			})
		}
	})

	t.Run("allowed methods", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPost, http.MethodDelete}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: methods,
			AllowedHeaders: DefaultConfig.AllowedHeaders,
		}
		zhtest.AssertEqual(t, 4, len(cfg.AllowedMethods))
		zhtest.AssertEqual(t, methods, cfg.AllowedMethods)
	})

	t.Run("all HTTP methods", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions, http.MethodConnect, http.MethodTrace}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: methods,
			AllowedHeaders: DefaultConfig.AllowedHeaders,
		}
		zhtest.AssertEqual(t, 9, len(cfg.AllowedMethods))
		zhtest.AssertEqual(t, methods, cfg.AllowedMethods)
	})

	t.Run("allowed headers", func(t *testing.T) {
		headers := []string{"Content-Type", "Authorization", "X-API-Key", "X-Client-Version"}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: DefaultConfig.AllowedMethods,
			AllowedHeaders: headers,
		}
		zhtest.AssertEqual(t, 4, len(cfg.AllowedHeaders))
		zhtest.AssertEqual(t, headers, cfg.AllowedHeaders)
	})

	t.Run("common headers", func(t *testing.T) {
		headers := []string{"Accept", "Accept-Language", "Authorization", "Cache-Control", "Content-Language", "Content-Type", "X-CSRF-Token", "X-Requested-With", "X-API-Key", "X-Client-Version"}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: DefaultConfig.AllowedMethods,
			AllowedHeaders: headers,
		}
		zhtest.AssertEqual(t, 10, len(cfg.AllowedHeaders))
		zhtest.AssertEqual(t, headers, cfg.AllowedHeaders)
	})

	t.Run("exposed headers", func(t *testing.T) {
		headers := []string{"X-Total-Count", "X-Rate-Limit", "Link", "ETag"}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: DefaultConfig.AllowedMethods,
			AllowedHeaders: DefaultConfig.AllowedHeaders,
			ExposedHeaders: headers,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExposedHeaders))
		zhtest.AssertEqual(t, headers, cfg.ExposedHeaders)
	})

	t.Run("boolean options", func(t *testing.T) {
		cfg := Config{
			AllowedOrigins:     DefaultConfig.AllowedOrigins,
			AllowedMethods:     DefaultConfig.AllowedMethods,
			AllowedHeaders:     DefaultConfig.AllowedHeaders,
			AllowCredentials:   true,
			OptionsPassthrough: true,
		}
		zhtest.AssertTrue(t, cfg.AllowCredentials)
		zhtest.AssertTrue(t, cfg.OptionsPassthrough)
	})

	t.Run("max age", func(t *testing.T) {
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: DefaultConfig.AllowedMethods,
			AllowedHeaders: DefaultConfig.AllowedHeaders,
			MaxAge:         3600,
		}
		zhtest.AssertEqual(t, 3600, cfg.MaxAge)
	})

	t.Run("excluded paths", func(t *testing.T) {
		excludedPaths := []string{"/health", "/metrics", "/internal", "/debug"}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: DefaultConfig.AllowedMethods,
			AllowedHeaders: DefaultConfig.AllowedHeaders,
			ExcludedPaths:  excludedPaths,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: DefaultConfig.AllowedMethods,
			AllowedHeaders: DefaultConfig.AllowedHeaders,
			IncludedPaths:  includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, includedPaths, cfg.IncludedPaths)
	})
}

func TestCORSConfig_MultipleFields(t *testing.T) {
	origins := []string{"https://example.com"}
	methods := []string{http.MethodGet, http.MethodPost}
	headers := []string{"Content-Type"}
	exposedHeaders := []string{"X-Total-Count"}
	excludedPaths := []string{"/health"}
	includedPaths := []string{"/api/public"}
	cfg := Config{
		AllowedOrigins:     origins,
		AllowedMethods:     methods,
		AllowedHeaders:     headers,
		ExposedHeaders:     exposedHeaders,
		AllowCredentials:   true,
		MaxAge:             7200,
		OptionsPassthrough: true,
		ExcludedPaths:      excludedPaths,
		IncludedPaths:      includedPaths,
	}

	zhtest.AssertEqual(t, origins, cfg.AllowedOrigins)
	zhtest.AssertEqual(t, methods, cfg.AllowedMethods)
	zhtest.AssertEqual(t, headers, cfg.AllowedHeaders)
	zhtest.AssertEqual(t, exposedHeaders, cfg.ExposedHeaders)
	zhtest.AssertTrue(t, cfg.AllowCredentials)
	zhtest.AssertEqual(t, 7200, cfg.MaxAge)
	zhtest.AssertTrue(t, cfg.OptionsPassthrough)
	zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	zhtest.AssertEqual(t, includedPaths, cfg.IncludedPaths)
}

func TestCORSConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := Config{
			AllowedOrigins: []string{},
			AllowedMethods: []string{},
			AllowedHeaders: []string{},
			ExposedHeaders: []string{},
			ExcludedPaths:  []string{},
			IncludedPaths:  []string{},
		}

		zhtest.AssertNotNil(t, cfg.AllowedOrigins)
		zhtest.AssertEqual(t, 0, len(cfg.AllowedOrigins))
		zhtest.AssertNotNil(t, cfg.AllowedMethods)
		zhtest.AssertEqual(t, 0, len(cfg.AllowedMethods))
		zhtest.AssertNotNil(t, cfg.AllowedHeaders)
		zhtest.AssertEqual(t, 0, len(cfg.AllowedHeaders))
		zhtest.AssertNotNil(t, cfg.ExposedHeaders)
		zhtest.AssertEqual(t, 0, len(cfg.ExposedHeaders))
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := Config{
			AllowedOrigins: nil,
			AllowedMethods: nil,
			AllowedHeaders: nil,
			ExposedHeaders: nil,
			ExcludedPaths:  nil,
			IncludedPaths:  nil,
		}

		zhtest.AssertNil(t, cfg.AllowedOrigins)
		zhtest.AssertNil(t, cfg.AllowedMethods)
		zhtest.AssertNil(t, cfg.AllowedHeaders)
		zhtest.AssertNil(t, cfg.ExposedHeaders)
		zhtest.AssertNil(t, cfg.ExcludedPaths)
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("max age boundary values", func(t *testing.T) {
		testCases := []int{0, 1, 3600, 86400, 604800, -1}
		for _, maxAge := range testCases {
			cfg := Config{
				AllowedOrigins: DefaultConfig.AllowedOrigins,
				AllowedMethods: DefaultConfig.AllowedMethods,
				AllowedHeaders: DefaultConfig.AllowedHeaders,
				MaxAge:         maxAge,
			}
			zhtest.AssertEqual(t, maxAge, cfg.MaxAge)
		}
	})

	t.Run("case sensitive headers", func(t *testing.T) {
		headers := []string{"Content-Type", "content-type", "CONTENT-TYPE", "Content-type"}
		cfg := Config{
			AllowedOrigins: DefaultConfig.AllowedOrigins,
			AllowedMethods: DefaultConfig.AllowedMethods,
			AllowedHeaders: headers,
		}
		zhtest.AssertEqual(t, 4, len(cfg.AllowedHeaders))
		for i, expectedHeader := range headers {
			zhtest.AssertEqual(t, expectedHeader, cfg.AllowedHeaders[i])
		}
	})
}
