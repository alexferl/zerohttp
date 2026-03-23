// Package zerohttp provides configuration structs for zerohttp servers and middleware.
//
// This package contains all configuration types used to customize zerohttp behavior,
// from server settings to individual middleware options.
//
// # Server Configuration
//
// The main [Config] struct holds all server and middleware configuration:
//
//	app := zh.New(zh.Config
//	    Addr: ":8080",
//	    Logger: myLogger,
//	})
//
// # Middleware Configuration
//
// Each middleware has its own configuration struct:
//
// # CORS
//
//	app.Use(cors.New(cors.Config{
//	    AllowedOrigins: []string{"https://example.com"},
//	    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
//	    AllowCredentials: true,
//	    MaxAge: 86400,
//	}))
//
// Or use [cors.DefaultConfig] as a starting point.
//
// # Basic Authentication
//
//	app.Use(basicauth.New(basicauth.Config{
//	    Credentials: map[string]string{
//	        "admin": "secret-password",
//	    },
//	    Realm: "Restricted Area",
//	    ExcludedPaths: []string{"/health"},
//	}))
//
// # JWT Authentication
//
//	cfg := jwtauth.Config{
//	    TokenStore:     myTokenStore,
//	    RequiredClaims: []string{"sub"},
//	    ExcludedPaths:    []string{"/login", "/register"},
//	}
//	app.Use(jwtauth.New(cfg))
//
// For a zero-dependency JWT solution, use the built-in HS256:
//
//	store := jwtauth.NewHS256Store(secret, jwtauth.HS256Config{
//	    Issuer: "my-app",
//	    AccessTokenTTL:  15 * time.Minute,
//	    RefreshTokenTTL: 7 * 24 * time.Hour,
//	})
//
// # Rate Limiting
//
//	app.Use(ratelimit.New(ratelimit.Config{
//	    Rate:      100,
//	    Window:    time.Minute,
//	    Algorithm: ratelimit.TokenBucket,
//	}))
//
// Algorithms: [ratelimit.TokenBucket] or [ratelimit.SlidingWindow].
//
// # Compression
//
//	app.Use(compress.New(compress.Config{
//	    Level:     6,
//	    Types:     []string{"text/html", "application/json"},
//	    MinLength: 1024,
//	}))
//
// # Security Headers
//
//	app.Use(securityheaders.New(securityheaders.Config{
//	    CSP:           "default-src 'self'; script-src 'self'",
//	    XFrameOptions: "DENY",
//	    HSTS: config.HSTSConfig{
//	        MaxAge: 31536000,
//	        Preload: true,
//	    },
//	}))
//
// # Request Logging
//
//	app.Use(requestlogger.New(logger, requestlogger.Config{
//	    Fields: []string{"method", "path", "status", "duration", "ip"},
//	}))
//
// # Circuit Breaker
//
//	app.Use(circuitbreaker.New(circuitbreaker.Config{
//	    FailureThreshold: 5,
//	    RecoveryTimeout:  30 * time.Second,
//	    SuccessThreshold: 3,
//	}))
//
// # Request Body Size Limit
//
//	app.Use(requestbodysize.New(requestbodysize.Config{
//	    MaxBytes: 1024 * 1024, // 1MB
//	}))
//
// # Request ID
//
//	app.Use(requestid.New(requestid.Config{
//	    Header: "X-Request-ID",
//	}))
//
// # Timeout
//
//	app.Use(timeout.New(timeout.Config{
//	    Duration: 30 * time.Second,
//	}))
//
// # CSRF
//
//	app.Use(csrf.New(csrf.Config{
//	    TokenLength: 32,
//	    CookieName:  "csrf_token",
//	    HeaderName:  "X-CSRF-Token",
//	}))
//
// # TLS Configuration
//
//	app := zh.New(zh.Config
//	    TLS: zh.TLSConfig{
//	        Addr:     ":8443",
//	        CertFile: "server.crt",
//	        KeyFile:  "server.key",
//	    },
//	})
//
// # Metrics Configuration
//
//	app := zh.New(zh.Config
//	    Metrics: metrics.Config{
//	        Enabled:  true,
//	        Endpoint: "/metrics",
//	        ExcludedPaths: []string{"/health", "/readyz"},
//	    },
//	})
//
// # Custom Validator
//
// Bring your own struct validator:
//
//	app := zh.New(zh.Config
//	    Validator: myCustomValidator, // implements Validator interface
//	})
//
// # Default Configurations
//
// Most middlewares provide a DefaultConfig variable with sensible defaults.
// These can be used as-is or as a base for customization:
//
//	// Use defaults
//	app.Use(cors.New(cors.DefaultConfig))
//
//	// Customize from defaults
//	cfg := cors.DefaultConfig
//	cfg.AllowedOrigins = []string{"https://example.com"}
//	app.Use(cors.New(cfg))
package zerohttp

import (
	"context"
	"net"
	"net/http"
	"reflect"

	"github.com/alexferl/zerohttp/extensions/autocert"
	"github.com/alexferl/zerohttp/extensions/http3"
	"github.com/alexferl/zerohttp/extensions/websocket"
	"github.com/alexferl/zerohttp/extensions/webtransport"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/middleware/recover"
	"github.com/alexferl/zerohttp/middleware/requestbodysize"
	"github.com/alexferl/zerohttp/middleware/requestid"
	"github.com/alexferl/zerohttp/middleware/requestlogger"
	"github.com/alexferl/zerohttp/middleware/securityheaders"
	"github.com/alexferl/zerohttp/middleware/tracer"
	"github.com/alexferl/zerohttp/sse"
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
	Recover recover.Config

	// RequestBodySize holds the configuration for the request body size limiting middleware.
	RequestBodySize requestbodysize.Config

	// RequestID holds the configuration for the request ID generation middleware.
	RequestID requestid.Config

	// RequestLogger holds the configuration for the HTTP request logging middleware.
	RequestLogger requestlogger.Config

	// SecurityHeaders holds the configuration for the security headers middleware.
	SecurityHeaders securityheaders.Config

	// Metrics holds the configuration for the metrics middleware.
	Metrics metrics.Config

	// Tracer holds the configuration for the tracing middleware.
	// Default: DefaultTracerConfig
	Tracer tracer.Config

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
	// PreStartupHooks are hooks that execute sequentially before any startup hooks.
	// These run before the server begins initialization.
	// Default: nil
	PreStartupHooks []StartupHookConfig

	// StartupHooks are hooks that execute sequentially before the server starts
	// accepting connections. If any startup hook returns an error, the server
	// will not start.
	// Default: nil
	StartupHooks []StartupHookConfig

	// PostStartupHooks are hooks that execute sequentially after the server has
	// started accepting connections.
	// Default: nil
	PostStartupHooks []StartupHookConfig

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
	AutocertManager autocert.Manager

	// HTTP3Server is an optional HTTP/3 server instance for handling HTTP/3 traffic over QUIC.
	// Users can inject their own HTTP/3 implementation (e.g., quic-go/http3).
	// The server must implement the HTTP3Server interface.
	// Default: nil (HTTP/3 not enabled unless set)
	HTTP3Server http3.Server

	// SSEProvider is an optional handler for Server-Sent Events connections.
	// Users can set their own provider (e.g., wrapping a custom SSE library).
	// If nil, SSE is not available but users can still handle SSE manually in their handlers.
	// Default: nil
	SSEProvider sse.Provider

	// WebSocketUpgrader is an optional handler for WebSocket upgrades.
	// Users can set their own upgrader (e.g., wrapping gorilla/websocket).
	// If nil, WebSocket is not available but users can still handle
	// upgrades manually in their handlers.
	WebSocketUpgrader websocket.Upgrader

	// WebTransportServer is an optional WebTransport server for handling WebTransport sessions.
	// Users can inject their own implementation (e.g., quic-go/webtransport-go).
	// The server must implement the WebTransportServer interface.
	// If nil, WebTransport support will not be enabled.
	// The server will be started automatically when ListenAndServeTLS or Start is called.
	// Default: nil
	WebTransportServer webtransport.Server
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
	Recover:                   recover.DefaultConfig,
	RequestBodySize:           requestbodysize.DefaultConfig,
	RequestID:                 requestid.DefaultConfig,
	RequestLogger:             requestlogger.DefaultConfig,
	SecurityHeaders:           securityheaders.DefaultConfig,
	Metrics:                   metrics.DefaultConfig,
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
//	app := zerohttp.New(zh.ConfigValidator: &myValidator{v: validator.New()}})
type Validator interface {
	// Struct validates a struct using `validate` struct tags.
	// It returns an error containing all validation failures, or nil if valid.
	Struct(dst any) error

	// Register adds a custom validation function with the given name.
	// The name can be used in struct tags like `validate:"customName"`.
	Register(name string, fn func(reflect.Value, string) error)
}
