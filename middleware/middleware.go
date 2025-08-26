package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

func DefaultMiddlewares(cfg config.Config, logger log.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		RequestID(cfg.RequestIDOptions...),
		Recover(logger, cfg.RecoverOptions...),
		RequestBodySize(cfg.RequestBodySizeOptions...),
		SecurityHeaders(cfg.SecurityHeadersOptions...),
		RequestLogger(logger, cfg.RequestLoggerOptions...),
	}
}

// pathMatches checks if a request path matches an exempt path
// Supports exact matches and prefix matches (paths ending with /)
func pathMatches(requestPath, exemptPath string) bool {
	if exemptPath == requestPath {
		return true
	}

	// Support prefix matching for paths ending with /
	if strings.HasSuffix(exemptPath, "/") {
		return strings.HasPrefix(requestPath, exemptPath)
	}

	return false
}
