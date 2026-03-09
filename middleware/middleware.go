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
// Supports exact matches and prefix matches (paths ending with /).
// For example, "/api/public/" matches "/api/public", "/api/public/users", and "/api/public/status".
func pathMatches(requestPath, exemptPath string) bool {
	if exemptPath == requestPath {
		return true
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
