// Package mwutil provides shared utilities for middleware implementations.
package mwutil

import (
	"fmt"
	"strings"
)

// PathMatches checks if a request path matches an excluded path.
// Supports:
//   - Exact matches
//   - Prefix matches (paths ending with /)
//   - Wildcard suffix matches (paths ending with *)
//
// For example:
//   - "/api/public/" matches "/api/public", "/api/public/users", "/api/public/status"
//   - "/api/live*" matches "/api/live", "/api/livez", "/api/health/live"
func PathMatches(requestPath, excludedPath string) bool {
	if excludedPath == requestPath {
		return true
	}

	// Support wildcard suffix matching for paths ending with *
	if base, hasSuffix := strings.CutSuffix(excludedPath, "*"); hasSuffix {
		return strings.HasPrefix(requestPath, base)
	}

	// Support prefix matching for paths ending with /
	if base, hasSuffix := strings.CutSuffix(excludedPath, "/"); hasSuffix {
		// Also match the path without the trailing slash (e.g., "/api/public" matches "/api/public/")
		if requestPath == base {
			return true
		}
		return strings.HasPrefix(requestPath, excludedPath)
	}

	return false
}

// ShouldProcessMiddleware checks if a path should be processed by middleware based on
// IncludedPaths and ExcludedPaths configuration. Returns true if the middleware should run.
//
// Rules:
//   - If IncludedPaths is set, the path must match one of the allowed patterns
//   - If ExcludedPaths is set, the path must NOT match any of the excluded patterns
//   - If both are empty, the middleware runs for all paths
func ShouldProcessMiddleware(path string, includedPaths, excludedPaths []string) bool {
	// If IncludedPaths is set, only process matching paths
	if len(includedPaths) > 0 {
		for _, allowedPath := range includedPaths {
			if PathMatches(path, allowedPath) {
				return true
			}
		}
		return false
	}

	// Check excluded paths
	for _, excludedPath := range excludedPaths {
		if PathMatches(path, excludedPath) {
			return false
		}
	}

	return true
}

// ValidatePathConfig checks that ExcludedPaths and IncludedPaths are not both set,
// which would be a configuration error. The middleware name is used for the panic message.
func ValidatePathConfig(excludedPaths, includedPaths []string, middlewareName string) {
	if len(excludedPaths) > 0 && len(includedPaths) > 0 {
		panic(fmt.Sprintf("%s: cannot set both ExcludedPaths and IncludedPaths", middlewareName))
	}
}
