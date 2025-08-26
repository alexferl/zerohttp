package config

// LogField represents a field to log.
type LogField string

const (
	FieldMethod        LogField = "method"
	FieldURI           LogField = "uri"
	FieldPath          LogField = "path"
	FieldHost          LogField = "host"
	FieldProtocol      LogField = "protocol"
	FieldReferer       LogField = "referer"
	FieldUserAgent     LogField = "user_agent"
	FieldStatus        LogField = "status"
	FieldDurationNS    LogField = "duration_ns"
	FieldDurationHuman LogField = "duration_human"
	FieldRemoteAddr    LogField = "remote_addr"
	FieldClientIP      LogField = "client_ip"
	FieldRequestID     LogField = "request_id"
)

// RequestLoggerConfig allows customization of request logging.
type RequestLoggerConfig struct {
	// LogErrors determines if errors should be logged (defaults to true).
	LogErrors bool
	// Fields to include in logs (defaults to all fields).
	Fields []LogField
	// ExemptPaths contains paths to skip logging (e.g., health checks).
	ExemptPaths []string
}

// DefaultRequestLoggerConfig contains the default values for request logging configuration.
var DefaultRequestLoggerConfig = RequestLoggerConfig{
	LogErrors: true,
	Fields: []LogField{
		FieldMethod,
		FieldURI,
		FieldPath,
		FieldHost,
		FieldProtocol,
		FieldReferer,
		FieldUserAgent,
		FieldStatus,
		FieldDurationNS,
		FieldDurationHuman,
		FieldRemoteAddr,
		FieldClientIP,
		FieldRequestID,
	},
	ExemptPaths: []string{},
}

// RequestLoggerOption configures request logging middleware.
type RequestLoggerOption func(*RequestLoggerConfig)

// WithRequestLoggerLogErrors sets whether errors should be logged.
func WithRequestLoggerLogErrors(logErrors bool) RequestLoggerOption {
	return func(c *RequestLoggerConfig) {
		c.LogErrors = logErrors
	}
}

// WithRequestLoggerFields sets the fields to include in logs.
func WithRequestLoggerFields(fields []LogField) RequestLoggerOption {
	return func(c *RequestLoggerConfig) {
		c.Fields = fields
	}
}

// WithRequestLoggerExemptPaths sets the paths to skip logging.
func WithRequestLoggerExemptPaths(paths []string) RequestLoggerOption {
	return func(c *RequestLoggerConfig) {
		c.ExemptPaths = paths
	}
}

// requestLoggerConfigToOptions converts a RequestLoggerConfig struct to a slice of RequestLoggerOption functions.
func requestLoggerConfigToOptions(cfg RequestLoggerConfig) []RequestLoggerOption {
	return []RequestLoggerOption{
		WithRequestLoggerLogErrors(cfg.LogErrors),
		WithRequestLoggerFields(cfg.Fields),
		WithRequestLoggerExemptPaths(cfg.ExemptPaths),
	}
}
