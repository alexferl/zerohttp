package middleware

import (
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

// pathMatches checks if a request path matches an exempt path.
// Supports:
//   - Exact matches
//   - Prefix matches (paths ending with /)
//   - Wildcard suffix matches (paths ending with *)
//
// For example:
//   - "/api/public/" matches "/api/public", "/api/public/users", "/api/public/status"
//   - "/api/live*" matches "/api/live", "/api/livez", "/api/health/live"
func pathMatches(requestPath, exemptPath string) bool {
	if exemptPath == requestPath {
		return true
	}

	// Support wildcard suffix matching for paths ending with *
	if base, hasSuffix := strings.CutSuffix(exemptPath, "*"); hasSuffix {
		return strings.HasPrefix(requestPath, base)
	}

	// Support prefix matching for paths ending with /
	if base, hasSuffix := strings.CutSuffix(exemptPath, "/"); hasSuffix {
		// Also match the path without the trailing slash (e.g., "/api/public" matches "/api/public/")
		if requestPath == base {
			return true
		}
		return strings.HasPrefix(requestPath, exemptPath)
	}

	return false
}
