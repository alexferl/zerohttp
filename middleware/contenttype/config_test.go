package contenttype

import (
	"reflect"
	"testing"
)

func TestContentTypeConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	if len(cfg.ContentTypes) != 3 {
		t.Errorf("expected 3 default content types, got %d", len(cfg.ContentTypes))
	}
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("expected default excluded paths to be empty, got %d paths", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 0 {
		t.Errorf("expected default included paths to be empty, got %d paths", len(cfg.IncludedPaths))
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

func TestContentTypeConfig_StructAssignment(t *testing.T) {
	t.Run("content types assignment", func(t *testing.T) {
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
				cfg := Config{
					ContentTypes: tt.input,
				}
				if len(cfg.ContentTypes) != len(tt.expected) {
					t.Errorf("expected %d content types, got %d", len(tt.expected), len(cfg.ContentTypes))
				}
				if !reflect.DeepEqual(cfg.ContentTypes, tt.expected) {
					t.Errorf("expected content types = %v, got %v", tt.expected, cfg.ContentTypes)
				}
			})
		}
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/api/upload", "/health", "/webhook", "/files"}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != 4 {
			t.Errorf("expected 4 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("included paths assignment", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
			t.Errorf("expected included paths = %v, got %v", includedPaths, cfg.IncludedPaths)
		}
	})
}

func TestContentTypeConfig_MultipleFields(t *testing.T) {
	contentTypes := []string{"application/json", "text/xml"}
	excludedPaths := []string{"/upload", "/download"}
	includedPaths := []string{"/api/public"}
	cfg := Config{
		ContentTypes:  contentTypes,
		ExcludedPaths: excludedPaths,
		IncludedPaths: includedPaths,
	}

	if len(cfg.ContentTypes) != 2 {
		t.Errorf("expected 2 content types, got %d", len(cfg.ContentTypes))
	}
	if !reflect.DeepEqual(cfg.ContentTypes, contentTypes) {
		t.Error("expected content types to be set correctly")
	}
	if len(cfg.ExcludedPaths) != 2 {
		t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
	}
	if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
		t.Error("expected excluded paths to be set correctly")
	}
	if len(cfg.IncludedPaths) != 1 {
		t.Errorf("expected 1 allowed path, got %d", len(cfg.IncludedPaths))
	}
	if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
		t.Error("expected included paths to be set correctly")
	}
}

func TestContentTypeConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := Config{
			ContentTypes:  []string{},
			ExcludedPaths: []string{},
			IncludedPaths: []string{},
		}

		if cfg.ContentTypes == nil || len(cfg.ContentTypes) != 0 {
			t.Errorf("expected empty content types slice, got %v", cfg.ContentTypes)
		}
		if cfg.ExcludedPaths == nil || len(cfg.ExcludedPaths) != 0 {
			t.Errorf("expected empty excluded paths slice, got %v", cfg.ExcludedPaths)
		}
		if cfg.IncludedPaths == nil || len(cfg.IncludedPaths) != 0 {
			t.Errorf("expected empty included paths slice, got %v", cfg.IncludedPaths)
		}
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := Config{
			ContentTypes:  nil,
			ExcludedPaths: nil,
			IncludedPaths: nil,
		}

		if cfg.ContentTypes != nil {
			t.Error("expected content types to remain nil when nil is passed")
		}
		if cfg.ExcludedPaths != nil {
			t.Error("expected excluded paths to remain nil when nil is passed")
		}
		if cfg.IncludedPaths != nil {
			t.Error("expected included paths to remain nil when nil is passed")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		contentTypes := []string{"application/json", "Application/JSON", "APPLICATION/JSON", "application/Json"}
		cfg := Config{
			ContentTypes: contentTypes,
		}
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
		cfg := Config{
			ContentTypes: contentTypes,
		}
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
		cfg := Config{
			ContentTypes: contentTypes,
		}
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
		excludedPaths := []string{"", "/health", ""}
		cfg := Config{
			ContentTypes:  contentTypes,
			ExcludedPaths: excludedPaths,
		}

		if len(cfg.ContentTypes) != 3 {
			t.Errorf("expected 3 content types, got %d", len(cfg.ContentTypes))
		}
		if len(cfg.ExcludedPaths) != 3 {
			t.Errorf("expected 3 excluded paths, got %d", len(cfg.ExcludedPaths))
		}

		for i, expectedType := range contentTypes {
			if cfg.ContentTypes[i] != expectedType {
				t.Errorf("expected content type[%d] = %q, got %q", i, expectedType, cfg.ContentTypes[i])
			}
		}
		for i, expectedPath := range excludedPaths {
			if cfg.ExcludedPaths[i] != expectedPath {
				t.Errorf("expected excluded path[%d] = %q, got %q", i, expectedPath, cfg.ExcludedPaths[i])
			}
		}
	})
}

func TestContentTypeConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api/v1/upload/*",
			"/static/*",
			"/health",
			"/metrics",
			"*.json",
			"*.xml",
			"/admin/files/*",
			"/webhook/*",
		}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api-v1/upload",
			"/static_files",
			"/health-check",
			"/files.json",
			"/admin/files (test)",
			"/path with spaces",
			"/path/with/unicode-ñ",
			"/files/test@example.com",
		}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})
}
