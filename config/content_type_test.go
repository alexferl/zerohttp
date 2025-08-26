package config

import (
	"reflect"
	"testing"
)

func TestContentTypeConfig_DefaultValues(t *testing.T) {
	cfg := DefaultContentTypeConfig
	if len(cfg.ContentTypes) != 3 {
		t.Errorf("expected 3 default content types, got %d", len(cfg.ContentTypes))
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
	}
	expectedContentTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}
	if !reflect.DeepEqual(cfg.ContentTypes, expectedContentTypes) {
		t.Errorf("expected default content types = %v, got %v", expectedContentTypes, cfg.ContentTypes)
	}
	if cfg.ContentTypes[0] != "application/json" {
		t.Errorf("expected first content type = 'application/json', got %s", cfg.ContentTypes[0])
	}
	if cfg.ContentTypes[1] != "application/x-www-form-urlencoded" {
		t.Errorf("expected second content type = 'application/x-www-form-urlencoded', got %s", cfg.ContentTypes[1])
	}
	if cfg.ContentTypes[2] != "multipart/form-data" {
		t.Errorf("expected third content type = 'multipart/form-data', got %s", cfg.ContentTypes[2])
	}
}

func TestContentTypeOptions(t *testing.T) {
	t.Run("content types option", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []string
			expected []string
		}{
			{"basic types", []string{"application/json", "application/xml", "text/plain", "text/html"}, []string{"application/json", "application/xml", "text/plain", "text/html"}},
			{"JSON only", []string{"application/json"}, []string{"application/json"}},
			{"common content types", []string{"application/json", "application/xml", "text/xml", "text/plain", "text/html", "text/css", "application/javascript", "text/javascript", "application/x-www-form-urlencoded", "multipart/form-data", "application/octet-stream"}, []string{"application/json", "application/xml", "text/xml", "text/plain", "text/html", "text/css", "application/javascript", "text/javascript", "application/x-www-form-urlencoded", "multipart/form-data", "application/octet-stream"}},
			{"binary types", []string{"application/octet-stream", "image/jpeg", "image/png", "image/gif", "video/mp4", "audio/mpeg", "application/pdf", "application/zip"}, []string{"application/octet-stream", "image/jpeg", "image/png", "image/gif", "video/mp4", "audio/mpeg", "application/pdf", "application/zip"}},
			{"API types", []string{"application/json", "application/xml", "application/hal+json", "application/vnd.api+json", "application/problem+json", "application/merge-patch+json", "application/json-patch+json"}, []string{"application/json", "application/xml", "application/hal+json", "application/vnd.api+json", "application/problem+json", "application/merge-patch+json", "application/json-patch+json"}},
			{"wildcards", []string{"text/*", "application/*", "image/*", "*/*"}, []string{"text/*", "application/*", "image/*", "*/*"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := DefaultContentTypeConfig
				WithContentTypeContentTypes(tt.input)(&cfg)
				if len(cfg.ContentTypes) != len(tt.expected) {
					t.Errorf("expected %d content types, got %d", len(tt.expected), len(cfg.ContentTypes))
				}
				if !reflect.DeepEqual(cfg.ContentTypes, tt.expected) {
					t.Errorf("expected content types = %v, got %v", tt.expected, cfg.ContentTypes)
				}
			})
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/api/upload", "/health", "/webhook", "/files"}
		cfg := DefaultContentTypeConfig
		WithContentTypeExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestContentTypeConfig_MultipleOptions(t *testing.T) {
	contentTypes := []string{"application/json", "text/xml"}
	exemptPaths := []string{"/upload", "/download"}
	cfg := DefaultContentTypeConfig
	WithContentTypeContentTypes(contentTypes)(&cfg)
	WithContentTypeExemptPaths(exemptPaths)(&cfg)

	if len(cfg.ContentTypes) != 2 {
		t.Errorf("expected 2 content types, got %d", len(cfg.ContentTypes))
	}
	if !reflect.DeepEqual(cfg.ContentTypes, contentTypes) {
		t.Error("expected content types to be set correctly")
	}
	if len(cfg.ExemptPaths) != 2 {
		t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
	}
	if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
		t.Error("expected exempt paths to be set correctly")
	}
}

func TestContentTypeConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := DefaultContentTypeConfig
		WithContentTypeContentTypes([]string{})(&cfg)
		WithContentTypeExemptPaths([]string{})(&cfg)

		if cfg.ContentTypes == nil || len(cfg.ContentTypes) != 0 {
			t.Errorf("expected empty content types slice, got %v", cfg.ContentTypes)
		}
		if cfg.ExemptPaths == nil || len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %v", cfg.ExemptPaths)
		}
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := DefaultContentTypeConfig
		WithContentTypeContentTypes(nil)(&cfg)
		WithContentTypeExemptPaths(nil)(&cfg)

		if cfg.ContentTypes != nil {
			t.Error("expected content types to remain nil when nil is passed")
		}
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		contentTypes := []string{"application/json", "Application/JSON", "APPLICATION/JSON", "application/Json"}
		cfg := DefaultContentTypeConfig
		WithContentTypeContentTypes(contentTypes)(&cfg)
		if len(cfg.ContentTypes) != 4 {
			t.Errorf("expected 4 content types, got %d", len(cfg.ContentTypes))
		}
		for i, expectedType := range contentTypes {
			if cfg.ContentTypes[i] != expectedType {
				t.Errorf("expected content type[%d] = %s, got %s", i, expectedType, cfg.ContentTypes[i])
			}
		}
	})

	t.Run("duplicate content types", func(t *testing.T) {
		contentTypes := []string{"application/json", "text/plain", "application/json", "text/html", "text/plain"}
		cfg := DefaultContentTypeConfig
		WithContentTypeContentTypes(contentTypes)(&cfg)
		if len(cfg.ContentTypes) != 5 {
			t.Errorf("expected 5 content types (including duplicates), got %d", len(cfg.ContentTypes))
		}
		for i, expectedType := range contentTypes {
			if cfg.ContentTypes[i] != expectedType {
				t.Errorf("expected content type[%d] = %s, got %s", i, expectedType, cfg.ContentTypes[i])
			}
		}
	})

	t.Run("content type parameters", func(t *testing.T) {
		contentTypes := []string{
			"application/json; charset=utf-8",
			"text/html; charset=iso-8859-1",
			"multipart/form-data; boundary=something",
			"application/json",
		}
		cfg := DefaultContentTypeConfig
		WithContentTypeContentTypes(contentTypes)(&cfg)
		if len(cfg.ContentTypes) != 4 {
			t.Errorf("expected 4 content types, got %d", len(cfg.ContentTypes))
		}
		for i, expectedType := range contentTypes {
			if cfg.ContentTypes[i] != expectedType {
				t.Errorf("expected content type[%d] = %s, got %s", i, expectedType, cfg.ContentTypes[i])
			}
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		contentTypes := []string{"", "application/json", ""}
		exemptPaths := []string{"", "/health", ""}
		cfg := DefaultContentTypeConfig
		WithContentTypeContentTypes(contentTypes)(&cfg)
		WithContentTypeExemptPaths(exemptPaths)(&cfg)

		if len(cfg.ContentTypes) != 3 {
			t.Errorf("expected 3 content types, got %d", len(cfg.ContentTypes))
		}
		if len(cfg.ExemptPaths) != 3 {
			t.Errorf("expected 3 exempt paths, got %d", len(cfg.ExemptPaths))
		}

		for i, expectedType := range contentTypes {
			if cfg.ContentTypes[i] != expectedType {
				t.Errorf("expected content type[%d] = %q, got %q", i, expectedType, cfg.ContentTypes[i])
			}
		}
		for i, expectedPath := range exemptPaths {
			if cfg.ExemptPaths[i] != expectedPath {
				t.Errorf("expected exempt path[%d] = %q, got %q", i, expectedPath, cfg.ExemptPaths[i])
			}
		}
	})
}

func TestContentTypeConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		exemptPaths := []string{
			"/api/v1/upload/*",
			"/static/*",
			"/health",
			"/metrics",
			"*.json",
			"*.xml",
			"/admin/files/*",
			"/webhook/*",
		}
		cfg := DefaultContentTypeConfig
		WithContentTypeExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		exemptPaths := []string{
			"/api-v1/upload",
			"/static_files",
			"/health-check",
			"/files.json",
			"/admin/files (test)",
			"/path with spaces",
			"/path/with/unicode-ñ",
			"/files/test@example.com",
		}
		cfg := DefaultContentTypeConfig
		WithContentTypeExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}
