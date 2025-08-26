package config

import (
	"net"
	"net/http"

	"github.com/alexferl/zerohttp/log"
	"golang.org/x/crypto/acme/autocert"
)

// Config holds server and middleware configuration options for zerohttp.
type Config struct {
	// Addr is the address for the HTTP server to listen on.
	// Default: "localhost:8080"
	Addr string

	// TLSAddr is the address for the HTTPS server to listen on.
	// Default: "localhost:8443"
	TLSAddr string

	// DisableDefaultMiddlewares disables all built-in default middlewares when true.
	// Default: false (default middlewares are enabled)
	DisableDefaultMiddlewares bool

	// DefaultMiddlewares is a custom list of middlewares to use. If nil, uses the built-in default middleware list.
	// Default: nil (means use built-in defaults)
	DefaultMiddlewares []func(http.Handler) http.Handler

	// RecoverOptions contains options for configuring the panic recovery middleware.
	RecoverOptions []RecoverOption

	// RequestBodySizeOptions contains options for configuring the request body size limiting middleware.
	RequestBodySizeOptions []RequestBodySizeOption

	// RequestIDOptions contains options for configuring the request ID generation middleware.
	RequestIDOptions []RequestIDOption

	// RequestLoggerOptions contains options for configuring the HTTP request logging middleware.
	RequestLoggerOptions []RequestLoggerOption

	// SecurityHeadersOptions contains options for configuring the security headers middleware.
	SecurityHeadersOptions []SecurityHeadersOption

	// Recover holds the built configuration for the panic recovery middleware.
	Recover RecoverConfig

	// RequestBodySize holds the built configuration for the request body size limiting middleware.
	RequestBodySize RequestBodySizeConfig

	// RequestID holds the built configuration for the request ID generation middleware.
	RequestID RequestIDConfig

	// RequestLogger holds the built configuration for the HTTP request logging middleware.
	RequestLogger RequestLoggerConfig

	// SecurityHeaders holds the built configuration for the security headers middleware.
	SecurityHeaders SecurityHeadersConfig

	// Logger is the logger instance used by the server and middlewares.
	// Default: nil (a default logger will be created if nil)
	Logger log.Logger

	// Server is the HTTP server instance for plain (non-TLS) traffic.
	// Default: preconfigured server listening on "localhost:8080"
	Server *http.Server

	// Listener allows specifying a custom net.Listener for HTTP traffic (optional).
	// Default: nil (system default listener will be created)
	Listener net.Listener

	// TLSServer is the HTTPS server instance for encrypted traffic.
	// Default: preconfigured server listening on "localhost:8443"
	TLSServer *http.Server

	// TLSListener allows specifying a custom net.Listener for HTTPS traffic (optional).
	// Default: nil (system default listener will be created)
	TLSListener net.Listener

	// CertFile is the file path to the TLS certificate (PEM) when serving HTTPS.
	// Default: "" (no certificate loaded unless specified)
	CertFile string

	// KeyFile is the file path to the TLS private key (PEM) when serving HTTPS.
	// Default: "" (no key loaded unless specified)
	KeyFile string

	// AutocertManager is an optional autocert.Manager for Let's Encrypt certificate management (AutoTLS).
	// Default: nil (AutoTLS not enabled unless set)
	AutocertManager *autocert.Manager
}

// DefaultConfig contains all default values used by Config.
// Update this file if you want to change system-wide defaults.
var DefaultConfig = Config{
	Addr:                      "localhost:8080",
	TLSAddr:                   "localhost:8443",
	DisableDefaultMiddlewares: false,
	DefaultMiddlewares:        nil, // means use DefaultMiddlewares
	RecoverOptions:            recoverConfigToOptions(DefaultRecoverConfig),
	RequestBodySizeOptions:    requestBodySizeConfigToOptions(DefaultRequestBodySizeConfig),
	RequestIDOptions:          requestIDConfigToOptions(DefaultRequestIDConfig),
	RequestLoggerOptions:      requestLoggerConfigToOptions(DefaultRequestLoggerConfig),
	SecurityHeadersOptions:    securityHeadersConfigToOptions(DefaultSecurityHeadersConfig),
	Logger:                    nil, // means use DefaultLogger
	Server:                    nil,
	TLSServer:                 nil,
}

// Build applies all configured options to populate the middleware configuration structs.
func (c *Config) Build() {
	c.Recover = DefaultRecoverConfig
	for _, opt := range c.RecoverOptions {
		opt(&c.Recover)
	}

	c.RequestBodySize = DefaultRequestBodySizeConfig
	for _, opt := range c.RequestBodySizeOptions {
		opt(&c.RequestBodySize)
	}

	c.RequestID = DefaultRequestIDConfig
	for _, opt := range c.RequestIDOptions {
		opt(&c.RequestID)
	}

	c.RequestLogger = DefaultRequestLoggerConfig
	for _, opt := range c.RequestLoggerOptions {
		opt(&c.RequestLogger)
	}

	c.SecurityHeaders = DefaultSecurityHeadersConfig
	for _, opt := range c.SecurityHeadersOptions {
		opt(&c.SecurityHeaders)
	}
}

// Option is a function that sets a field in Config.
type Option func(*Config)

// WithAddr sets the HTTP server address.
func WithAddr(addr string) Option {
	return func(c *Config) {
		c.Addr = addr
	}
}

// WithTLSAddr sets the HTTPS server address.
func WithTLSAddr(addr string) Option {
	return func(c *Config) {
		c.TLSAddr = addr
	}
}

// WithDisableDefaultMiddlewares disables all built-in default middlewares.
func WithDisableDefaultMiddlewares() Option {
	return func(c *Config) {
		c.DisableDefaultMiddlewares = true
	}
}

// WithDefaultMiddlewares sets custom default middlewares.
func WithDefaultMiddlewares(mw []func(http.Handler) http.Handler) Option {
	return func(c *Config) {
		c.DefaultMiddlewares = mw
	}
}

// WithRecoverOptions configures the panic recovery middleware.
func WithRecoverOptions(opts ...RecoverOption) Option {
	return func(c *Config) {
		c.RecoverOptions = append([]RecoverOption{}, opts...)
	}
}

// WithRequestBodySizeOptions configures the request body size limiting middleware.
func WithRequestBodySizeOptions(opts ...RequestBodySizeOption) Option {
	return func(c *Config) {
		c.RequestBodySizeOptions = append([]RequestBodySizeOption{}, opts...)
	}
}

// WithRequestIDOptions configures the request ID generation middleware.
func WithRequestIDOptions(opts ...RequestIDOption) Option {
	return func(c *Config) {
		c.RequestIDOptions = append([]RequestIDOption{}, opts...)
	}
}

// WithRequestLoggerOptions configures the HTTP request logging middleware.
func WithRequestLoggerOptions(opts ...RequestLoggerOption) Option {
	return func(c *Config) {
		c.RequestLoggerOptions = append([]RequestLoggerOption{}, opts...)
	}
}

// WithSecurityHeadersOptions configures the security headers middleware.
func WithSecurityHeadersOptions(opts ...SecurityHeadersOption) Option {
	return func(c *Config) {
		c.SecurityHeadersOptions = append([]SecurityHeadersOption{}, opts...)
	}
}

// WithLogger sets a custom logger instance.
func WithLogger(logger log.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithServer sets a custom HTTP server instance.
func WithServer(server *http.Server) Option {
	return func(c *Config) {
		c.Server = server
	}
}

// WithListener sets a custom network listener for HTTP traffic.
func WithListener(listener net.Listener) Option {
	return func(c *Config) {
		c.Listener = listener
	}
}

// WithTLSServer sets a custom HTTPS server instance.
func WithTLSServer(server *http.Server) Option {
	return func(c *Config) {
		c.TLSServer = server
	}
}

// WithTLSListener sets a custom network listener for HTTPS traffic.
func WithTLSListener(listener net.Listener) Option {
	return func(c *Config) {
		c.TLSListener = listener
	}
}

// WithCertFile sets the file path to the TLS certificate.
func WithCertFile(path string) Option {
	return func(c *Config) {
		c.CertFile = path
	}
}

// WithKeyFile sets the file path to the TLS private key.
func WithKeyFile(path string) Option {
	return func(c *Config) {
		c.KeyFile = path
	}
}

// WithAutocertManager sets an autocert.Manager for automatic TLS certificate management.
func WithAutocertManager(mgr *autocert.Manager) Option {
	return func(c *Config) {
		c.AutocertManager = mgr
	}
}
