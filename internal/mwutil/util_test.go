package mwutil

import "testing"

func TestPathMatches(t *testing.T) {
	tests := []struct {
		requestPath  string
		excludedPath string
		expected     bool
	}{
		{"/health", "/health", true},
		{"/health", "/metrics", false},
		{"/api/public/users", "/api/public/", true},
		{"/api/public", "/api/public/", true},   // path without trailing slash matches
		{"/api/public/", "/api/public/", true},  // exact match with trailing slash
		{"/api/publicx", "/api/public/", false}, // different path, shouldn't match
		{"/", "/", true},
		{"", "", true},
		{"/api/v1/users", "/api/", true},
		{"/api/v1/users", "/api", false}, // no trailing slash = no prefix match
		{"/api", "/api/", true},          // path without trailing slash matches
		{"/different", "/api/", false},
	}

	for _, tt := range tests {
		t.Run(tt.requestPath+"_vs_"+tt.excludedPath, func(t *testing.T) {
			result := PathMatches(tt.requestPath, tt.excludedPath)
			if result != tt.expected {
				t.Errorf("pathMatches(%q, %q) = %v, expected %v",
					tt.requestPath, tt.excludedPath, result, tt.expected)
			}
		})
	}
}
