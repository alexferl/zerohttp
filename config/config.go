package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/alexferl/zerohttp/log"
)

// Config holds server and middleware configuration options for zerohttp.
type Config struct {
	// Addr is the address for the HTTP server to listen on.
	// Default: "localhost:8080"
	Addr string

	// Server is the HTTP server instance for plain (non-TLS) traffic.
	// Default: preconfigured server listening on "localhost:8080"
	Server *http.Server

	// Listener allows specifying a custom net.Listener for HTTP traffic (optional).
	// Default: nil (system default listener will be created)
	Listener net.Listener

	// TLS holds the configuration for the HTTPS server.
	TLS TLSConfig

	// Lifecycle holds the server startup and shutdown hook configuration.
	Lifecycle LifecycleConfig

	// Logger is the logger instance used by the server and middlewares.
	// Default: nil (a default logger will be created if nil)
	Logger log.Logger

	// DisableDefaultMiddlewares disables all built-in default middlewares when true.
	// Default: false (default middlewares are enabled)
	DisableDefaultMiddlewares bool

	// DefaultMiddlewares is a custom list of middlewares to use. If nil, uses the built-in default middleware list.
	// Default: nil (means use built-in defaults)
	DefaultMiddlewares []func(http.Handler) http.Handler

	// Recover holds the configuration for the panic recovery middleware.
	Recover RecoverConfig

	// RequestBodySize holds the configuration for the request body size limiting middleware.
	RequestBodySize RequestBodySizeConfig

	// RequestID holds the configuration for the request ID generation middleware.
	RequestID RequestIDConfig

	// RequestLogger holds the configuration for the HTTP request logging middleware.
	RequestLogger RequestLoggerConfig

	// SecurityHeaders holds the configuration for the security headers middleware.
	SecurityHeaders SecurityHeadersConfig

	// Metrics holds the configuration for the metrics middleware.
	Metrics MetricsConfig

	// Tracer holds the configuration for the tracing middleware.
	// Default: DefaultTracerConfig
	Tracer TracerConfig

	// Validator is an optional struct validator for validating request data.
	// Users can inject their own implementation (e.g., github.com/go-playground/validator/v10).
	// The validator must implement the Validator interface.
	// If nil, the default built-in validator will be used.
	// Default: nil
	Validator Validator

	// Extensions holds optional protocol and feature extensions.
	Extensions ExtensionsConfig
}

type TLSConfig struct {
	// Addr is the address for the HTTPS server to listen on.
	// Default: "localhost:8443"
	Addr string
	// Server is the HTTPS server instance for encrypted traffic.
	// Default: preconfigured server listening on "localhost:8443"
	Server *http.Server

	// Listener allows specifying a custom net.Listener for HTTPS traffic (optional).
	// Default: nil (system default listener will be created)
	Listener net.Listener

	// CertFile is the file path to the TLS certificate (PEM) when serving HTTPS.
	// Default: "" (no certificate loaded unless specified)
	CertFile string

	// KeyFile is the file path to the TLS private key (PEM) when serving HTTPS.
	// Default: "" (no key loaded unless specified)
	KeyFile string
}

type LifecycleConfig struct {
	// StartupHooks are hooks that execute sequentially before the server starts
	// accepting connections. If any startup hook returns an error, the server
	// will not start.
	// Default: nil
	StartupHooks []StartupHookConfig

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
}

type ExtensionsConfig struct {
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

	// SSEProvider is an optional handler for Server-Sent Events connections.
	// Users can set their own provider (e.g., wrapping a custom SSE library).
	// If nil, SSE is not available but users can still handle SSE manually in their handlers.
	// Default: nil
	SSEProvider SSEProvider

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
	Addr: "localhost:8080",
	TLS: TLSConfig{
		Addr:   "localhost:8443",
		Server: nil,
	},
	DisableDefaultMiddlewares: false,
	DefaultMiddlewares:        nil, // means use DefaultMiddlewares
	Recover:                   DefaultRecoverConfig,
	RequestBodySize:           DefaultRequestBodySizeConfig,
	RequestID:                 DefaultRequestIDConfig,
	RequestLogger:             DefaultRequestLoggerConfig,
	SecurityHeaders:           DefaultSecurityHeadersConfig,
	Metrics:                   DefaultMetricsConfig,
	Logger:                    nil, // means use DefaultLogger
	Server:                    nil,
}

// ============================================================================
// Startup Hook Types
// ============================================================================

// StartupHook is a function called before the server starts accepting connections.
// The context passed to the hook has a deadline based on any configured timeout.
//
// Startup hooks execute sequentially in registration order.
// If any startup hook returns an error, the server will not start.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, startup will hang.
//
// Example:
//
//	app.RegisterStartupHook("migrations", func(ctx context.Context) error {
//	    return goose.Up(db.DB, "migrations")
//	})
type StartupHook func(ctx context.Context) error

// StartupHookConfig configures a startup hook.
type StartupHookConfig struct {
	Name string
	Hook StartupHook
}

// ============================================================================
// Shutdown Hook Types
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

// ============================================================================
// Validator Interface
// ============================================================================

// Validator is the interface for struct validation.
// Users can implement this interface to provide their own validation logic
// (e.g., wrapping github.com/go-playground/validator/v10).
//
// Example with go-playground/validator:
//
//	import "github.com/go-playground/validator/v10"
//
//	type myValidator struct {
//	    v *validator.Validate
//	}
//
//	func (m *myValidator) Struct(dst any) error {
//	    return m.v.Struct(dst)
//	}
//
//	func (m *myValidator) Register(name string, fn func(reflect.Value, string) error) {
//	    // Custom registration or no-op
//	}
//
//	app := zerohttp.New(config.Config{Validator: &myValidator{v: validator.New()}})
type Validator interface {
	// Struct validates a struct using `validate` struct tags.
	// It returns an error containing all validation failures, or nil if valid.
	Struct(dst any) error

	// Register adds a custom validation function with the given name.
	// The name can be used in struct tags like `validate:"customName"`.
	Register(name string, fn func(reflect.Value, string) error)
}

// ============================================================================
// Autocert Manager Interface
// ============================================================================

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
//	app := zerohttp.New(config.Config{AutocertManager: mgr})
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

// ============================================================================
// HTTP/3 Interfaces
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

// ============================================================================
// SSE Interfaces
// ============================================================================

// SSEConnection represents an active Server-Sent Events connection.
// Users can implement this interface with their own SSE library, or use
// the built-in EventStream implementation.
type SSEConnection interface {
	// Send writes an event to the client.
	// Returns error if the connection is closed or write fails.
	Send(event SSEEvent) error

	// SendComment sends a comment (heartbeat/keepalive).
	// Comments are ignored by the client but keep connections alive through proxies.
	SendComment(comment string) error

	// Close signals the SSE connection is done.
	// No further events should be sent after Close.
	Close() error

	// SetRetry sets the default reconnection time for this connection.
	// Affects subsequent events without explicit Retry value.
	SetRetry(d time.Duration) error
}

// SSEProvider creates SSE connections from HTTP requests.
// Implement this to provide custom SSE implementations.
type SSEProvider interface {
	// NewSSE creates a new SSE connection from the request/response.
	// Returns error if headers were already sent or SSE is not supported.
	NewSSE(w http.ResponseWriter, r *http.Request) (SSEConnection, error)
}

// SSEEvent represents a single SSE event
type SSEEvent struct {
	ID    string
	Name  string
	Data  []byte
	Retry time.Duration
}

// ============================================================================
// WebSocket Interfaces
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

// ============================================================================
// WebTransport Interfaces
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
