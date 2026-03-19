package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

func DefaultMiddlewares(cfg config.Config, logger log.Logger) []func(http.Handler) http.Handler {
	// Sync RequestID header configuration with Recover config
	recoverConfig := cfg.Recover
	recoverConfig.RequestIDHeader = cfg.RequestID.Header

	return []func(http.Handler) http.Handler{
		RequestID(cfg.RequestID),
		Recover(logger, recoverConfig),
		RequestBodySize(cfg.RequestBodySize),
		SecurityHeaders(cfg.SecurityHeaders),
		RequestLogger(logger, cfg.RequestLogger),
	}
}

// pathMatches checks if a request path matches an excluded path.
// Supports:
//   - Exact matches
//   - Prefix matches (paths ending with /)
//   - Wildcard suffix matches (paths ending with *)
//
// For example:
//   - "/api/public/" matches "/api/public", "/api/public/users", "/api/public/status"
//   - "/api/live*" matches "/api/live", "/api/livez", "/api/health/live"
func pathMatches(requestPath, excludedPath string) bool {
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

// shouldProcessMiddleware checks if a path should be processed by middleware based on
// IncludedPaths and ExcludedPaths configuration. Returns true if the middleware should run.
//
// Rules:
//   - If IncludedPaths is set, the path must match one of the allowed patterns
//   - If ExcludedPaths is set, the path must NOT match any of the excluded patterns
//   - If both are empty, the middleware runs for all paths
func shouldProcessMiddleware(path string, includedPaths, excludedPaths []string) bool {
	// If IncludedPaths is set, only process matching paths
	if len(includedPaths) > 0 {
		for _, allowedPath := range includedPaths {
			if pathMatches(path, allowedPath) {
				return true
			}
		}
		return false
	}

	// Check excluded paths
	for _, excludedPath := range excludedPaths {
		if pathMatches(path, excludedPath) {
			return false
		}
	}

	return true
}

// validatePathConfig checks that ExcludedPaths and IncludedPaths are not both set,
// which would be a configuration error. The middleware name is used for the panic message.
func validatePathConfig(excludedPaths, includedPaths []string, middlewareName string) {
	if len(excludedPaths) > 0 && len(includedPaths) > 0 {
		panic(fmt.Sprintf("%s: cannot set both ExcludedPaths and IncludedPaths", middlewareName))
	}
}
