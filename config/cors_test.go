package config

import (
	"net/http"
	"reflect"
	"testing"
)

func TestCORSConfig_DefaultValues(t *testing.T) {
	cfg := DefaultCORSConfig
	if len(cfg.AllowedOrigins) != 1 {
		t.Errorf("expected 1 default allowed origin, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected default allowed origin = '*', got %s", cfg.AllowedOrigins[0])
	}
	if len(cfg.AllowedMethods) != 7 {
		t.Errorf("expected 7 default allowed methods, got %d", len(cfg.AllowedMethods))
	}
	if len(cfg.AllowedHeaders) != 5 {
		t.Errorf("expected 5 default allowed headers, got %d", len(cfg.AllowedHeaders))
	}
	if len(cfg.ExposedHeaders) != 0 {
		t.Errorf("expected default exposed headers to be empty, got %d headers", len(cfg.ExposedHeaders))
	}
	if cfg.AllowCredentials != false {
		t.Errorf("expected default allow credentials = false, got %t", cfg.AllowCredentials)
	}
	if cfg.MaxAge != 86400 {
		t.Errorf("expected default max age = 86400, got %d", cfg.MaxAge)
	}
	if cfg.OptionsPassthrough != false {
		t.Errorf("expected default options passthrough = false, got %t", cfg.OptionsPassthrough)
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
	}

	// Test default method values
	expectedMethods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions}
	if !reflect.DeepEqual(cfg.AllowedMethods, expectedMethods) {
		t.Errorf("expected default methods = %v, got %v", expectedMethods, cfg.AllowedMethods)
	}

	// Test default header values
	expectedHeaders := []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-Id"}
	if !reflect.DeepEqual(cfg.AllowedHeaders, expectedHeaders) {
		t.Errorf("expected default headers = %v, got %v", expectedHeaders, cfg.AllowedHeaders)
	}
}

func TestCORSOptions(t *testing.T) {
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
				cfg := DefaultCORSConfig
				WithCORSAllowedOrigins(tt.input)(&cfg)
				if len(cfg.AllowedOrigins) != len(tt.expected) {
					t.Errorf("expected %d allowed origins, got %d", len(tt.expected), len(cfg.AllowedOrigins))
				}
				if !reflect.DeepEqual(cfg.AllowedOrigins, tt.expected) {
					t.Errorf("expected origins = %v, got %v", tt.expected, cfg.AllowedOrigins)
				}
			})
		}
	})

	t.Run("allowed methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		cfg := DefaultCORSConfig
		WithCORSAllowedMethods(methods)(&cfg)
		if len(cfg.AllowedMethods) != 4 {
			t.Errorf("expected 4 allowed methods, got %d", len(cfg.AllowedMethods))
		}
		if !reflect.DeepEqual(cfg.AllowedMethods, methods) {
			t.Errorf("expected methods = %v, got %v", methods, cfg.AllowedMethods)
		}
	})

	t.Run("all HTTP methods", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions, http.MethodConnect, http.MethodTrace}
		cfg := DefaultCORSConfig
		WithCORSAllowedMethods(methods)(&cfg)
		if len(cfg.AllowedMethods) != 9 {
			t.Errorf("expected 9 methods, got %d", len(cfg.AllowedMethods))
		}
		if !reflect.DeepEqual(cfg.AllowedMethods, methods) {
			t.Errorf("expected methods = %v, got %v", methods, cfg.AllowedMethods)
		}
	})

	t.Run("allowed headers", func(t *testing.T) {
		headers := []string{"Content-Type", "Authorization", "X-API-Key", "X-Client-Version"}
		cfg := DefaultCORSConfig
		WithCORSAllowedHeaders(headers)(&cfg)
		if len(cfg.AllowedHeaders) != 4 {
			t.Errorf("expected 4 allowed headers, got %d", len(cfg.AllowedHeaders))
		}
		if !reflect.DeepEqual(cfg.AllowedHeaders, headers) {
			t.Errorf("expected headers = %v, got %v", headers, cfg.AllowedHeaders)
		}
	})

	t.Run("common headers", func(t *testing.T) {
		headers := []string{"Accept", "Accept-Language", "Authorization", "Cache-Control", "Content-Language", "Content-Type", "X-CSRF-Token", "X-Requested-With", "X-API-Key", "X-Client-Version"}
		cfg := DefaultCORSConfig
		WithCORSAllowedHeaders(headers)(&cfg)
		if len(cfg.AllowedHeaders) != 10 {
			t.Errorf("expected 10 common headers, got %d", len(cfg.AllowedHeaders))
		}
		if !reflect.DeepEqual(cfg.AllowedHeaders, headers) {
			t.Errorf("expected headers = %v, got %v", headers, cfg.AllowedHeaders)
		}
	})

	t.Run("exposed headers", func(t *testing.T) {
		headers := []string{"X-Total-Count", "X-Rate-Limit", "Link", "ETag"}
		cfg := DefaultCORSConfig
		WithCORSExposedHeaders(headers)(&cfg)
		if len(cfg.ExposedHeaders) != 4 {
			t.Errorf("expected 4 exposed headers, got %d", len(cfg.ExposedHeaders))
		}
		if !reflect.DeepEqual(cfg.ExposedHeaders, headers) {
			t.Errorf("expected exposed headers = %v, got %v", headers, cfg.ExposedHeaders)
		}
	})

	t.Run("boolean options", func(t *testing.T) {
		cfg := DefaultCORSConfig
		WithCORSAllowCredentials(true)(&cfg)
		if cfg.AllowCredentials != true {
			t.Errorf("expected allow credentials = true, got %t", cfg.AllowCredentials)
		}
		WithCORSOptionsPassthrough(true)(&cfg)
		if cfg.OptionsPassthrough != true {
			t.Errorf("expected options passthrough = true, got %t", cfg.OptionsPassthrough)
		}
		WithCORSAllowCredentials(false)(&cfg)
		WithCORSOptionsPassthrough(false)(&cfg)
		if cfg.AllowCredentials != false || cfg.OptionsPassthrough != false {
			t.Error("expected boolean values to be set back to false")
		}
	})

	t.Run("max age", func(t *testing.T) {
		cfg := DefaultCORSConfig
		WithCORSMaxAge(3600)(&cfg)
		if cfg.MaxAge != 3600 {
			t.Errorf("expected max age = 3600, got %d", cfg.MaxAge)
		}
	})

	t.Run("exempt paths", func(t *testing.T) {
		exemptPaths := []string{"/health", "/metrics", "/internal", "/debug"}
		cfg := DefaultCORSConfig
		WithCORSExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestCORSConfig_MultipleOptions(t *testing.T) {
	origins := []string{"https://example.com"}
	methods := []string{"GET", "POST"}
	headers := []string{"Content-Type"}
	exposedHeaders := []string{"X-Total-Count"}
	exemptPaths := []string{"/health"}
	cfg := DefaultCORSConfig
	WithCORSAllowedOrigins(origins)(&cfg)
	WithCORSAllowedMethods(methods)(&cfg)
	WithCORSAllowedHeaders(headers)(&cfg)
	WithCORSExposedHeaders(exposedHeaders)(&cfg)
	WithCORSAllowCredentials(true)(&cfg)
	WithCORSMaxAge(7200)(&cfg)
	WithCORSOptionsPassthrough(true)(&cfg)
	WithCORSExemptPaths(exemptPaths)(&cfg)

	if !reflect.DeepEqual(cfg.AllowedOrigins, origins) {
		t.Error("expected origins to be set correctly")
	}
	if !reflect.DeepEqual(cfg.AllowedMethods, methods) {
		t.Error("expected methods to be set correctly")
	}
	if !reflect.DeepEqual(cfg.AllowedHeaders, headers) {
		t.Error("expected headers to be set correctly")
	}
	if !reflect.DeepEqual(cfg.ExposedHeaders, exposedHeaders) {
		t.Error("expected exposed headers to be set correctly")
	}
	if cfg.AllowCredentials != true {
		t.Error("expected allow credentials to be true")
	}
	if cfg.MaxAge != 7200 {
		t.Error("expected max age to be 7200")
	}
	if cfg.OptionsPassthrough != true {
		t.Error("expected options passthrough to be true")
	}
	if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
		t.Error("expected exempt paths to be set correctly")
	}
}

func TestCORSConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := DefaultCORSConfig
		WithCORSAllowedOrigins([]string{})(&cfg)
		WithCORSAllowedMethods([]string{})(&cfg)
		WithCORSAllowedHeaders([]string{})(&cfg)
		WithCORSExposedHeaders([]string{})(&cfg)
		WithCORSExemptPaths([]string{})(&cfg)

		if cfg.AllowedOrigins == nil || len(cfg.AllowedOrigins) != 0 {
			t.Errorf("expected empty allowed origins slice, got %v", cfg.AllowedOrigins)
		}
		if cfg.AllowedMethods == nil || len(cfg.AllowedMethods) != 0 {
			t.Errorf("expected empty allowed methods slice, got %v", cfg.AllowedMethods)
		}
		if cfg.AllowedHeaders == nil || len(cfg.AllowedHeaders) != 0 {
			t.Errorf("expected empty allowed headers slice, got %v", cfg.AllowedHeaders)
		}
		if cfg.ExposedHeaders == nil || len(cfg.ExposedHeaders) != 0 {
			t.Errorf("expected empty exposed headers slice, got %v", cfg.ExposedHeaders)
		}
		if cfg.ExemptPaths == nil || len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %v", cfg.ExemptPaths)
		}
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := DefaultCORSConfig
		WithCORSAllowedOrigins(nil)(&cfg)
		WithCORSAllowedMethods(nil)(&cfg)
		WithCORSAllowedHeaders(nil)(&cfg)
		WithCORSExposedHeaders(nil)(&cfg)
		WithCORSExemptPaths(nil)(&cfg)

		if cfg.AllowedOrigins != nil {
			t.Error("expected allowed origins to remain nil when nil is passed")
		}
		if cfg.AllowedMethods != nil {
			t.Error("expected allowed methods to remain nil when nil is passed")
		}
		if cfg.AllowedHeaders != nil {
			t.Error("expected allowed headers to remain nil when nil is passed")
		}
		if cfg.ExposedHeaders != nil {
			t.Error("expected exposed headers to remain nil when nil is passed")
		}
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("max age boundary values", func(t *testing.T) {
		testCases := []int{0, 1, 3600, 86400, 604800, -1}
		for _, maxAge := range testCases {
			cfg := DefaultCORSConfig
			WithCORSMaxAge(maxAge)(&cfg)
			if cfg.MaxAge != maxAge {
				t.Errorf("WithCORSMaxAge(%d): expected max age = %d, got %d", maxAge, maxAge, cfg.MaxAge)
			}
		}
	})

	t.Run("case sensitive headers", func(t *testing.T) {
		headers := []string{"Content-Type", "content-type", "CONTENT-TYPE", "Content-type"}
		cfg := DefaultCORSConfig
		WithCORSAllowedHeaders(headers)(&cfg)
		if len(cfg.AllowedHeaders) != 4 {
			t.Errorf("expected 4 headers, got %d", len(cfg.AllowedHeaders))
		}
		for i, expectedHeader := range headers {
			if cfg.AllowedHeaders[i] != expectedHeader {
				t.Errorf("expected header[%d] = %s, got %s", i, expectedHeader, cfg.AllowedHeaders[i])
			}
		}
	})
}
