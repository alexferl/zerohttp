package middleware

import (
	"net/http"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/log"
)

// RequestLogger creates a request logging middleware with the provided configuration.
func RequestLogger(logger log.Logger, cfg ...config.RequestLoggerConfig) func(http.Handler) http.Handler {
	c := config.DefaultRequestLoggerConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.Fields == nil {
		c.Fields = config.DefaultRequestLoggerConfig.Fields
	}

	fieldMap := make(map[config.LogField]bool)
	for _, field := range c.Fields {
		fieldMap[field] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if r.URL.Path == exemptPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()

			wrapped := rwutil.NewResponseWriter(w)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			LogRequest(logger, c, r, wrapped.StatusCode(), duration)
		})
	}
}

// LogRequest logs an HTTP request with consistent formatting.
func LogRequest(logger log.Logger, cfg config.RequestLoggerConfig, r *http.Request, statusCode int, duration time.Duration) {
	fieldMap := make(map[config.LogField]bool)
	for _, field := range cfg.Fields {
		fieldMap[field] = true
	}

	var logFields []log.Field

	if fieldMap[config.FieldMethod] {
		logFields = append(logFields, log.F("method", r.Method))
	}
	if fieldMap[config.FieldURI] {
		logFields = append(logFields, log.F("uri", r.RequestURI))
	}
	if fieldMap[config.FieldPath] {
		path := r.URL.Path
		if path == "" {
			path = "/"
		}
		logFields = append(logFields, log.F("path", path))
	}
	if fieldMap[config.FieldHost] {
		logFields = append(logFields, log.F("host", r.Host))
	}
	if fieldMap[config.FieldProtocol] {
		logFields = append(logFields, log.F("protocol", r.Proto))
	}
	if fieldMap[config.FieldReferer] {
		logFields = append(logFields, log.F("referer", r.Referer()))
	}
	if fieldMap[config.FieldUserAgent] {
		logFields = append(logFields, log.F("user_agent", r.UserAgent()))
	}
	if fieldMap[config.FieldStatus] {
		logFields = append(logFields, log.F("status", statusCode))
	}
	if fieldMap[config.FieldDurationNS] {
		logFields = append(logFields, log.F("duration_ns", duration.Nanoseconds()))
	}
	if fieldMap[config.FieldDurationHuman] {
		logFields = append(logFields, log.F("duration_human", duration.String()))
	}
	if fieldMap[config.FieldRemoteAddr] {
		logFields = append(logFields, log.F("remote_addr", r.RemoteAddr))
	}
	if fieldMap[config.FieldRequestID] {
		if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
			logFields = append(logFields, log.F("request_id", requestID))
		}
	}

	msg := "Request completed"

	if cfg.LogErrors {
		if statusCode >= http.StatusInternalServerError {
			logger.Error(msg, logFields...)
		} else if statusCode >= http.StatusBadRequest {
			logger.Warn(msg, logFields...)
		} else {
			logger.Info(msg, logFields...)
		}
	} else {
		logger.Info(msg, logFields...)
	}
}
