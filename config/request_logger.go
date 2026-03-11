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
	FieldRequestBody   LogField = "request_body"
	FieldResponseBody  LogField = "response_body"
)

// RequestLoggerConfig allows customization of request logging.
type RequestLoggerConfig struct {
	// Enabled determines if request logging is enabled at all.
	// When false, no request logging occurs (fastest option).
	// Use a pointer to distinguish between "not set" and "explicitly set to false".
	// Default: true
	Enabled *bool
	// LogErrors determines if errors should be logged (defaults to true).
	LogErrors bool
	// Fields to include in logs (defaults to all fields).
	Fields []LogField
	// ExemptPaths contains paths to skip logging (e.g., health checks).
	ExemptPaths []string
	// AllowedPaths contains paths where body logging is explicitly allowed.
	// If set, body logging (LogRequestBody/LogResponseBody) will only occur
	// for paths matching these patterns. Supports exact matches and prefixes (ending with /).
	// If empty, body logging applies to all paths (subject to ExemptPaths).
	AllowedPaths []string
	// LogRequestBody enables logging of request bodies (defaults to false).
	// This is opt-in due to performance and security considerations.
	LogRequestBody bool
	// LogResponseBody enables logging of response bodies (defaults to false).
	// This is opt-in due to performance and security considerations.
	LogResponseBody bool
	// MaxBodySize is the maximum number of bytes to log for request/response bodies.
	// If 0, defaults to 1KB. Use -1 for unlimited (not recommended).
	MaxBodySize int
	// SensitiveFields contains field names (case-insensitive) whose values should be
	// masked in request/response body logs (e.g., "password", "token", "secret").
	// Defaults to common sensitive field names if nil.
	SensitiveFields []string
}

// DefaultSensitiveFields contains common sensitive field names that should be masked.
// These are case-insensitive matches.
var DefaultSensitiveFields = []string{
	"password",
	"passwd",
	"pwd",
	"secret",
	"token",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"id_token",
	"authorization",
	"auth",
	"credential",
	"credentials",
	"private_key",
	"privatekey",
	"ssh_key",
	"sshkey",
	"credit_card",
	"creditcard",
	"cc_number",
	"cvv",
	"ssn",
	"dob",
}

// DefaultRequestLoggerConfig contains the default values for request logging configuration.
var DefaultRequestLoggerConfig = RequestLoggerConfig{
	Enabled:   Bool(true),
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
	ExemptPaths:     []string{},
	AllowedPaths:    []string{},
	MaxBodySize:     1024, // 1KB default
	SensitiveFields: DefaultSensitiveFields,
}
