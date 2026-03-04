package config

import (
	"context"
	"crypto/tls"
	"fmt"
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

	// Logger is the logger instance used by the server and middlewares.
	// Default: nil (a default logger will be created if nil)
	Logger log.Logger

	// PreShutdownHooks are hooks that execute sequentially before server shutdown begins.
	// These run before any servers start shutting down.
	// Default: nil
	PreShutdownHooks []ShutdownHookConfig

	// ShutdownHooks are hooks that execute concurrently with server shutdown.
	// These run alongside the HTTP/HTTPS/HTTP3 server shutdown.
	// Default: nil
	ShutdownHooks []ShutdownHookConfig

	// PostShutdownHooks are hooks that execute sequentially after all servers are shut down.
	// These run after all servers have completed shutdown.
	// Default: nil
	PostShutdownHooks []ShutdownHookConfig

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

	// WebSocketUpgrader is an optional handler for WebSocket upgrades.
	// Users can set their own upgrader (e.g., wrapping gorilla/websocket).
	// If nil, WebSocket is not available but users can still handle
	// upgrades manually in their handlers.
	WebSocketUpgrader WebSocketUpgrader

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

// ============================================================================
// Server Address Options
// ============================================================================

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

// ============================================================================
// Server Instance Options
// ============================================================================

// WithServer sets a custom HTTP server instance.
func WithServer(server *http.Server) Option {
	return func(c *Config) {
		c.Server = server
	}
}

// WithTLSServer sets a custom HTTPS server instance.
func WithTLSServer(server *http.Server) Option {
	return func(c *Config) {
		c.TLSServer = server
	}
}

// WithListener sets a custom network listener for HTTP traffic.
func WithListener(listener net.Listener) Option {
	return func(c *Config) {
		c.Listener = listener
	}
}

// WithTLSListener sets a custom network listener for HTTPS traffic.
func WithTLSListener(listener net.Listener) Option {
	return func(c *Config) {
		c.TLSListener = listener
	}
}

// ============================================================================
// Logging Options
// ============================================================================

// WithLogger sets a custom logger instance.
func WithLogger(logger log.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// ============================================================================
// Shutdown Hook Options
// ============================================================================

// ShutdownHook is a function called during server shutdown.
// The context passed to the hook will be cancelled when the shutdown
// timeout is reached or if the parent context is cancelled.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
type ShutdownHook func(ctx context.Context) error

// ShutdownHookConfig configures a shutdown hook.
type ShutdownHookConfig struct {
	Name string
	Hook ShutdownHook
}

// WithPreShutdownHook registers a hook to run before server shutdown begins.
// Pre-shutdown hooks execute sequentially in registration order.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	zerohttp.New(zerohttp.WithPreShutdownHook("health", func(ctx context.Context) error {
//	    health.SetUnhealthy()
//	    return nil
//	}))
func WithPreShutdownHook(name string, hook ShutdownHook) Option {
	return func(c *Config) {
		c.PreShutdownHooks = append(c.PreShutdownHooks, ShutdownHookConfig{Name: name, Hook: hook})
	}
}

// WithShutdownHook registers a hook to run concurrently with server shutdown.
// Shutdown hooks execute concurrently alongside server shutdown.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	zerohttp.New(zerohttp.WithShutdownHook("close-db", func(ctx context.Context) error {
//	    return db.Close()
//	}))
func WithShutdownHook(name string, hook ShutdownHook) Option {
	return func(c *Config) {
		c.ShutdownHooks = append(c.ShutdownHooks, ShutdownHookConfig{Name: name, Hook: hook})
	}
}

// WithPostShutdownHook registers a hook to run after servers are shut down.
// Post-shutdown hooks execute sequentially in registration order.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	zerohttp.New(zerohttp.WithPostShutdownHook("cleanup", func(ctx context.Context) error {
//	    return os.RemoveAll("/tmp/app-*")
//	}))
func WithPostShutdownHook(name string, hook ShutdownHook) Option {
	return func(c *Config) {
		c.PostShutdownHooks = append(c.PostShutdownHooks, ShutdownHookConfig{Name: name, Hook: hook})
	}
}

// ============================================================================
// Middleware Options
// ============================================================================

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

// ============================================================================
// TLS Certificate Options
// ============================================================================

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

	// Hostnames returns the list of hostnames configured for this manager.
	// This is used to proactively fetch certificates before starting HTTP/3.
	// Must return at least one hostname.
	Hostnames() []string
}

// WithAutocertManager sets an autocert manager for automatic TLS certificate management.
// The manager must implement the AutocertManager interface.
func WithAutocertManager(mgr AutocertManager) Option {
	return func(c *Config) {
		c.AutocertManager = mgr
	}
}

// ============================================================================
// HTTP/3 Options
// ============================================================================

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

// WithHTTP3Server sets a custom HTTP/3 server instance.
func WithHTTP3Server(server HTTP3Server) Option {
	return func(c *Config) {
		c.HTTP3Server = server
	}
}

// ============================================================================
// WebSocket Options
// ============================================================================

// WebSocketConn represents a WebSocket connection.
// This is a minimal interface that can be implemented by wrapping
// any WebSocket library (e.g., gorilla/websocket, nhooyr/websocket).
type WebSocketConn interface {
	// ReadMessage reads a message from the connection.
	// Returns message type (text=1, binary=2), payload, and error.
	ReadMessage() (int, []byte, error)

	// WriteMessage writes a message to the connection.
	// messageType is 1 for text, 2 for binary.
	WriteMessage(messageType int, data []byte) error

	// Close closes the connection gracefully.
	Close() error

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr
}

// WebSocketUpgrader handles upgrading HTTP connections to WebSocket.
// Users provide their own implementation using their preferred WebSocket library.
type WebSocketUpgrader interface {
	// Upgrade upgrades the HTTP connection to WebSocket.
	// The implementation is responsible for the RFC 6455 handshake
	// and returning a WebSocketConn.
	Upgrade(w http.ResponseWriter, r *http.Request) (WebSocketConn, error)
}

// CloseCode represents a WebSocket close code as defined in RFC 6455.
type CloseCode int

// WebSocket close code constants.
const (
	CloseNormalClosure           CloseCode = 1000
	CloseGoingAway               CloseCode = 1001
	CloseProtocolError           CloseCode = 1002
	CloseUnsupportedData         CloseCode = 1003
	CloseNoStatusReceived        CloseCode = 1005
	CloseAbnormalClosure         CloseCode = 1006
	CloseInvalidFramePayloadData CloseCode = 1007
	ClosePolicyViolation         CloseCode = 1008
	CloseMessageTooBig           CloseCode = 1009
	CloseMandatoryExtension      CloseCode = 1010
	CloseInternalServerErr       CloseCode = 1011
	CloseServiceRestart          CloseCode = 1012
	CloseTryAgainLater           CloseCode = 1013
	CloseBadGateway              CloseCode = 1014
	CloseTLSHandshake            CloseCode = 1015
)

// CloseError represents a WebSocket close error.
type CloseError struct {
	Code   int
	Reason string
}

// Error implements the error interface.
func (e *CloseError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("websocket: close %d %s", e.Code, e.Reason)
	}
	return fmt.Sprintf("websocket: close %d", e.Code)
}

// MessageType represents the type of WebSocket message.
type MessageType int

// WebSocket message type constants as defined in RFC 6455.
const (
	TextMessage   MessageType = 1
	BinaryMessage MessageType = 2
	CloseMessage  MessageType = 8
	PingMessage   MessageType = 9
	PongMessage   MessageType = 10
)

// WithWebSocketUpgrader sets a WebSocket upgrader.
// Users bring their own WebSocket library and implement the WebSocketUpgrader interface,
// or use a thin wrapper around their preferred library.
//
// Example with gorilla/websocket:
//
//	import "github.com/gorilla/websocket"
//
//	upgrader := &websocket.Upgrader{
//	    CheckOrigin: func(r *http.Request) bool { return true },
//	}
//
//	app := zerohttp.New(
//	    zerohttp.WithWebSocketUpgrader(&myUpgrader{upgrader}),
//	)
//
//	app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    ws, err := app.WebSocketUpgrader().Upgrade(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer ws.Close()
//	    // ... handle connection ...
//	}))
func WithWebSocketUpgrader(upgrader WebSocketUpgrader) Option {
	return func(c *Config) {
		c.WebSocketUpgrader = upgrader
	}
}

// ============================================================================
// WebTransport Options
// ============================================================================

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

// WebTransportServerWithAutocert is an optional interface for WebTransport servers that support
// automatic certificate management via autocert.Manager. If a WebTransport server
// implements this interface, it will be used by StartAutoTLS to configure
// WebTransport with Let's Encrypt certificates.
type WebTransportServerWithAutocert interface {
	WebTransportServer

	// ListenAndServeTLSWithAutocert starts the WebTransport server with automatic
	// certificate management using the provided autocert manager.
	// The manager's GetCertificate function is used to obtain TLS certificates.
	ListenAndServeTLSWithAutocert(manager AutocertManager) error
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
