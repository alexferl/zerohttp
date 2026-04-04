package mediatype

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultConfig(t *testing.T) {
	zhtest.AssertEqual(t, 0, len(DefaultConfig.AllowedTypes))
	zhtest.AssertEqual(t, "", DefaultConfig.DefaultType)
	zhtest.AssertEqual(t, false, DefaultConfig.ValidateContentType)
	zhtest.AssertEqual(t, 0, len(DefaultConfig.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(DefaultConfig.IncludedPaths))
}

func TestConfigMerge(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected Config
	}{
		{
			name: "custom allowed types",
			config: Config{
				AllowedTypes: []string{"application/vnd.api+json"},
			},
			expected: Config{
				AllowedTypes:        []string{"application/vnd.api+json"},
				DefaultType:         "",
				ValidateContentType: false,
				ExcludedPaths:       []string{},
				IncludedPaths:       []string{},
			},
		},
		{
			name: "with default type",
			config: Config{
				AllowedTypes: []string{"application/vnd.api+json"},
				DefaultType:  "application/vnd.api+json",
			},
			expected: Config{
				AllowedTypes:        []string{"application/vnd.api+json"},
				DefaultType:         "application/vnd.api+json",
				ValidateContentType: false,
				ExcludedPaths:       []string{},
				IncludedPaths:       []string{},
			},
		},
		{
			name: "with content type validation",
			config: Config{
				AllowedTypes:        []string{"application/json"},
				ValidateContentType: true,
			},
			expected: Config{
				AllowedTypes:        []string{"application/json"},
				DefaultType:         "",
				ValidateContentType: true,
				ExcludedPaths:       []string{},
				IncludedPaths:       []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := New(tt.config)
			zhtest.AssertNotNil(t, middleware)
		})
	}
}
