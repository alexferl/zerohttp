package config

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/alexferl/zerohttp/log"
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

	// AutocertManager is an optional autocert manager for automatic certificate management (AutoTLS).
	// Users can inject their own implementation (e.g., golang.org/x/crypto/acme/autocert.Manager)
	// by implementing the AutocertManager interface.
	// Default: nil (AutoTLS not enabled unless set)
	AutocertManager AutocertManager

	// HTTP3Server is an optional HTTP/3 server instance for handling HTTP/3 traffic over QUIC.
	// Users can inject their own HTTP/3 implementation (e.g., quic-go/http3).
	// The server must implement the HTTP3Server interface.
	// Default: nil (HTTP/3 not enabled unless set)
	HTTP3Server HTTP3Server

	// WebTransportServer is an optional WebTransport server for handling WebTransport sessions.
	// Users can inject their own implementation (e.g., quic-go/webtransport-go).
	// The server must implement the WebTransportServer interface.
	// If nil, WebTransport support will not be enabled.
	// The server will be started automatically when ListenAndServeTLS or Start is called.
	// Default: nil
	WebTransportServer WebTransportServer
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

// AutocertManager is the interface for automatic TLS certificate management.
// Users can implement this interface or use golang.org/x/crypto/acme/autocert.Manager
// which satisfies this interface.
//
// Example with autocert:
//
//	import "golang.org/x/crypto/acme/autocert"
//
//	mgr := &autocert.Manager{
//	    Cache:      autocert.DirCache("/var/cache/certs"),
//	    Prompt:     autocert.AcceptTOS,
//	    HostPolicy: autocert.HostWhitelist("example.com"),
//	}
//	srv := zerohttp.New(config.WithAutocertManager(mgr))
type AutocertManager interface {
	// GetCertificate returns a TLS certificate for the given client hello.
	// This is called by the TLS server during the handshake.
	GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)

	// HTTPHandler wraps the given handler to handle ACME HTTP-01 challenges.
	// Non-challenge requests are passed through to the wrapped handler.
	HTTPHandler(fallback http.Handler) http.Handler
}

// WithAutocertManager sets an autocert manager for automatic TLS certificate management.
// The manager must implement the AutocertManager interface.
func WithAutocertManager(mgr AutocertManager) Option {
	return func(c *Config) {
		c.AutocertManager = mgr
	}
}

// HTTP3Server is the interface that HTTP/3 servers must implement to be used with zerohttp.
// Users can inject their own HTTP/3 implementation (e.g., github.com/quic-go/quic-go/http3).
//
// Example usage with quic-go:
//
//	import "github.com/quic-go/quic-go/http3"
//
//	app := zerohttp.New()
//	h3Server := &http3.Server{Addr: ":443", Handler: app}
//	app.SetHTTP3Server(h3Server)
//	app.StartHTTP3("cert.pem", "key.pem")
type HTTP3Server interface {
	// ListenAndServeTLS starts the HTTP/3 server with the provided certificate and key.
	// Certificate files are in PEM format.
	ListenAndServeTLS(certFile, keyFile string) error

	// Shutdown gracefully shuts down the HTTP/3 server.
	Shutdown(ctx context.Context) error

	// Close immediately closes the HTTP/3 server.
	Close() error
}

// HTTP3ServerWithAutocert is an optional interface for HTTP/3 servers that support
// automatic certificate management via autocert.Manager. If an HTTP/3 server
// implements this interface, it will be used by StartAutoTLS to configure
// HTTP/3 with Let's Encrypt certificates.
//
// quic-go's http3.Server implements this interface when configured with a TLSConfig
// containing the autocert GetCertificate function.
type HTTP3ServerWithAutocert interface {
	HTTP3Server

	// ListenAndServeTLSWithAutocert starts the HTTP/3 server with automatic
	// certificate management using the provided autocert manager.
	// The manager's GetCertificate function is used to obtain TLS certificates.
	ListenAndServeTLSWithAutocert(manager AutocertManager) error
}

// WebTransportServer is the interface that WebTransport servers must implement
// to be used with zerohttp. Users can inject their own WebTransport implementation
// (e.g., github.com/quic-go/webtransport-go).
//
// Example usage with webtransport-go:
//
//	import "github.com/quic-go/webtransport-go"
//
//	app := zerohttp.New()
//	wtServer := &webtransport.Server{
//	    H3: &http3.Server{Addr: ":443", Handler: app},
//	}
//	app.SetWebTransportServer(wtServer)
//	app.ListenAndServeTLS("cert.pem", "key.pem") // wtServer starts automatically
type WebTransportServer interface {
	// ListenAndServeTLS starts the WebTransport server with the provided certificate and key.
	// Certificate files are in PEM format.
	ListenAndServeTLS(certFile, keyFile string) error

	// Close immediately closes the WebTransport server.
	Close() error
}

// WithHTTP3Server sets a custom HTTP/3 server instance.
func WithHTTP3Server(server HTTP3Server) Option {
	return func(c *Config) {
		c.HTTP3Server = server
	}
}

// WithWebTransportServer sets a custom WebTransport server instance.
// WebTransport runs over HTTP/3 and provides low-latency, bidirectional communication.
//
// The WebTransport server will be started automatically when ListenAndServeTLS or Start
// is called on the zerohttp server. You don't need to call ListenAndServeTLS on the
// WebTransport server yourself.
//
// Example:
//
//	import "github.com/quic-go/webtransport-go"
//
//	app := zerohttp.New()
//	wtServer := &webtransport.Server{
//	    H3: &http3.Server{Addr: ":443", Handler: app},
//	}
//	app.SetWebTransportServer(wtServer)
//	app.ListenAndServeTLS("cert.pem", "key.pem") // wtServer starts automatically
func WithWebTransportServer(server WebTransportServer) Option {
	return func(c *Config) {
		c.WebTransportServer = server
	}
}
