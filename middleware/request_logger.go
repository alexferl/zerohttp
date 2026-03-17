package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/log"
)

// RequestLogger creates a request logging middleware with the provided configuration.
func RequestLogger(logger log.Logger, cfg ...config.RequestLoggerConfig) func(http.Handler) http.Handler {
	c := config.DefaultRequestLoggerConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	if len(c.ExemptPaths) > 0 && len(c.AllowedPaths) > 0 {
		logger.Panic("RequestLogger: cannot set both ExemptPaths and AllowedPaths")
	}

	fieldMap := make(map[config.LogField]bool)
	for _, field := range c.Fields {
		fieldMap[field] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()

			bodyLoggingAllowed := isBodyLoggingAllowed(r.URL.Path, c.AllowedPaths)

			var requestBody string
			if c.LogRequestBody && bodyLoggingAllowed {
				requestBody = captureRequestBody(r, c.MaxBodySize)
			}

			var wrapped *bodyCapturingResponseWriter
			if c.LogResponseBody && bodyLoggingAllowed {
				wrapped = newBodyCapturingResponseWriter(w, c.MaxBodySize)
				next.ServeHTTP(wrapped, r)
			} else {
				wrapped = &bodyCapturingResponseWriter{
					ResponseWriter: rwutil.NewResponseWriter(w),
				}
				next.ServeHTTP(wrapped, r)
			}

			duration := time.Since(start)

			var responseBody string
			if c.LogResponseBody && bodyLoggingAllowed {
				responseBody = maskSensitiveData(wrapped.bodyString(), c.SensitiveFields)
			}
			if c.LogRequestBody && bodyLoggingAllowed && requestBody != "" {
				requestBody = maskSensitiveData(requestBody, c.SensitiveFields)
			}

			LogRequest(logger, c, fieldMap, r, wrapped.StatusCode(), duration, requestBody, responseBody)
		})
	}
}

// LogRequest logs an HTTP request with consistent formatting.
// If fieldMap is nil, it will be computed from cfg.Fields.
func LogRequest(logger log.Logger, cfg config.RequestLoggerConfig, fieldMap map[config.LogField]bool, r *http.Request, statusCode int, duration time.Duration, requestBody, responseBody string) {
	if fieldMap == nil {
		fieldMap = make(map[config.LogField]bool)
		for _, field := range cfg.Fields {
			fieldMap[field] = true
		}
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
		if requestID := r.Header.Get(httpx.HeaderXRequestID); requestID != "" {
			logFields = append(logFields, log.F("request_id", requestID))
		}
	}
	if fieldMap[config.FieldRequestBody] && cfg.LogRequestBody && requestBody != "" {
		logFields = append(logFields, log.F("request_body", requestBody))
	}
	if fieldMap[config.FieldResponseBody] && cfg.LogResponseBody && responseBody != "" {
		logFields = append(logFields, log.F("response_body", responseBody))
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

// bodyCapturingResponseWriter wraps ResponseWriter to capture response body for logging.
type bodyCapturingResponseWriter struct {
	*rwutil.ResponseWriter
	body      *bytes.Buffer
	maxSize   int
	sizeLimit bool
	truncated bool
}

// newBodyCapturingResponseWriter creates a new response writer that captures body.
func newBodyCapturingResponseWriter(w http.ResponseWriter, maxSize int) *bodyCapturingResponseWriter {
	return &bodyCapturingResponseWriter{
		ResponseWriter: rwutil.NewResponseWriter(w),
		body:           &bytes.Buffer{},
		maxSize:        maxSize,
		sizeLimit:      maxSize > 0,
	}
}

// Write captures the response body and forwards to the underlying ResponseWriter.
func (rw *bodyCapturingResponseWriter) Write(data []byte) (int, error) {
	// Capture body up to max size (only if body buffer is initialized)
	if rw.body != nil {
		if !rw.sizeLimit || rw.body.Len() < rw.maxSize {
			rw.body.Write(data)
			// If we exceeded the limit, truncate and mark as truncated
			if rw.sizeLimit && rw.body.Len() > rw.maxSize {
				rw.body.Truncate(rw.maxSize)
				rw.truncated = true
			}
		} else {
			// Already exceeded limit, mark as truncated
			rw.truncated = true
		}
	}
	return rw.ResponseWriter.Write(data)
}

// body returns the captured body as a string.
func (rw *bodyCapturingResponseWriter) bodyString() string {
	if rw.body == nil {
		return ""
	}
	body := rw.body.String()
	if rw.truncated {
		return body + "..."
	}
	return body
}

// Flush implements http.Flusher to support streaming responses like SSE.
// It passes the flush through to the underlying ResponseWriter if it supports Flusher.
func (rw *bodyCapturingResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// captureRequestBody reads and restores the request body.
func captureRequestBody(r *http.Request, maxSize int) string {
	if r.Body == nil || r.Body == http.NoBody {
		return ""
	}

	// maxSize <= 0 means body logging is disabled
	if maxSize <= 0 {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, int64(maxSize)+1))

	// Always restore the body, even on error, so downstream handlers are unaffected
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err != nil {
		return ""
	}

	if len(body) > maxSize {
		return string(body[:maxSize]) + "..."
	}
	return string(body)
}

// maskSensitiveData masks sensitive fields in JSON data.
func maskSensitiveData(data string, sensitiveFields []string) string {
	if data == "" || len(sensitiveFields) == 0 {
		return data
	}

	// Try to parse as JSON object
	var jsonObj map[string]any
	if err := json.Unmarshal([]byte(data), &jsonObj); err != nil {
		// Try to parse as JSON array
		var jsonArr []map[string]any
		if err := json.Unmarshal([]byte(data), &jsonArr); err != nil {
			// Not valid JSON, return as-is
			return data
		}
		// Process array of objects
		for i, obj := range jsonArr {
			jsonArr[i] = maskObject(obj, sensitiveFields)
		}
		result, err := json.Marshal(jsonArr)
		if err != nil {
			return data
		}
		return string(result)
	}

	// Process single object
	maskedObj := maskObject(jsonObj, sensitiveFields)
	result, err := json.Marshal(maskedObj)
	if err != nil {
		return data
	}
	return string(result)
}

// maskObject masks sensitive fields in a single JSON object.
func maskObject(obj map[string]any, sensitiveFields []string) map[string]any {
	if obj == nil {
		return nil
	}

	result := make(map[string]any, len(obj))
	for key, value := range obj {
		if isSensitiveField(key, sensitiveFields) {
			result[key] = "[REDACTED]"
		} else {
			// Recursively mask nested objects
			switch v := value.(type) {
			case map[string]any:
				result[key] = maskObject(v, sensitiveFields)
			case []any:
				result[key] = maskArray(v, sensitiveFields)
			default:
				result[key] = value
			}
		}
	}
	return result
}

// maskArray masks sensitive fields in a JSON array.
func maskArray(arr []any, sensitiveFields []string) []any {
	if arr == nil {
		return nil
	}

	result := make([]any, len(arr))
	for i, value := range arr {
		switch v := value.(type) {
		case map[string]any:
			result[i] = maskObject(v, sensitiveFields)
		case []any:
			result[i] = maskArray(v, sensitiveFields)
		default:
			result[i] = value
		}
	}
	return result
}

// isSensitiveField checks if a field name matches any sensitive field (case-insensitive).
func isSensitiveField(field string, sensitiveFields []string) bool {
	fieldLower := strings.ToLower(field)
	for _, sensitive := range sensitiveFields {
		if strings.EqualFold(sensitive, fieldLower) || strings.EqualFold(sensitive, field) {
			return true
		}
	}
	return false
}

// isBodyLoggingAllowed checks if body logging is allowed for the given path.
// If allowedPaths is empty, body logging is allowed for all paths.
// If allowedPaths is set, body logging is only allowed for matching paths.
func isBodyLoggingAllowed(path string, allowedPaths []string) bool {
	if len(allowedPaths) == 0 {
		return true
	}
	for _, allowedPath := range allowedPaths {
		if pathMatches(path, allowedPath) {
			return true
		}
	}
	return false
}
