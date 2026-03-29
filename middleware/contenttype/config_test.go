package contenttype

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestContentTypeConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 3, len(cfg.ContentTypes))
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	expectedContentTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}
	zhtest.AssertDeepEqual(t, expectedContentTypes, cfg.ContentTypes)
	zhtest.AssertEqual(t, "application/json", cfg.ContentTypes[0])
	zhtest.AssertEqual(t, "application/x-www-form-urlencoded", cfg.ContentTypes[1])
	zhtest.AssertEqual(t, "multipart/form-data", cfg.ContentTypes[2])
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
				zhtest.AssertEqual(t, len(tt.expected), len(cfg.ContentTypes))
				zhtest.AssertDeepEqual(t, tt.expected, cfg.ContentTypes)
			})
		}
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/api/upload", "/health", "/webhook", "/files"}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("included paths assignment", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
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

	zhtest.AssertEqual(t, 2, len(cfg.ContentTypes))
	zhtest.AssertDeepEqual(t, contentTypes, cfg.ContentTypes)
	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
	zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
}

func TestContentTypeConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := Config{
			ContentTypes:  []string{},
			ExcludedPaths: []string{},
			IncludedPaths: []string{},
		}

		zhtest.AssertNotNil(t, cfg.ContentTypes)
		zhtest.AssertEqual(t, 0, len(cfg.ContentTypes))
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := Config{
			ContentTypes:  nil,
			ExcludedPaths: nil,
			IncludedPaths: nil,
		}

		zhtest.AssertNil(t, cfg.ContentTypes)
		zhtest.AssertNil(t, cfg.ExcludedPaths)
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("case sensitivity", func(t *testing.T) {
		contentTypes := []string{"application/json", "Application/JSON", "APPLICATION/JSON", "application/Json"}
		cfg := Config{
			ContentTypes: contentTypes,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ContentTypes))
		for i, expectedType := range contentTypes {
			zhtest.AssertEqual(t, expectedType, cfg.ContentTypes[i])
		}
	})

	t.Run("duplicate content types", func(t *testing.T) {
		contentTypes := []string{"application/json", "text/plain", "application/json", "text/html", "text/plain"}
		cfg := Config{
			ContentTypes: contentTypes,
		}
		zhtest.AssertEqual(t, 5, len(cfg.ContentTypes))
		for i, expectedType := range contentTypes {
			zhtest.AssertEqual(t, expectedType, cfg.ContentTypes[i])
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
		zhtest.AssertEqual(t, 4, len(cfg.ContentTypes))
		for i, expectedType := range contentTypes {
			zhtest.AssertEqual(t, expectedType, cfg.ContentTypes[i])
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		contentTypes := []string{"", "application/json", ""}
		excludedPaths := []string{"", "/health", ""}
		cfg := Config{
			ContentTypes:  contentTypes,
			ExcludedPaths: excludedPaths,
		}

		zhtest.AssertEqual(t, 3, len(cfg.ContentTypes))
		zhtest.AssertEqual(t, 3, len(cfg.ExcludedPaths))

		for i, expectedType := range contentTypes {
			zhtest.AssertEqual(t, expectedType, cfg.ContentTypes[i])
		}
		for i, expectedPath := range excludedPaths {
			zhtest.AssertEqual(t, expectedPath, cfg.ExcludedPaths[i])
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
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("special character paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api-v1/upload",
			"/static_files",
			"/health-check",
			"/files.json",
			"/admin/files (test)",
			"/path with spaces",
			"/path/with/unicode-\xc3\xb1",
			"/files/test@example.com",
		}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})
}
