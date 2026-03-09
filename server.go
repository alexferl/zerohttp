package zerohttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/middleware"
)

// Server represents a zerohttp server instance that wraps Go's standard HTTP server
// with additional functionality including middleware support, TLS configuration,
// automatic certificate management, and structured logging.
//
// The Server embeds a Router interface, providing direct access to HTTP routing
// methods (GET, POST, PUT, DELETE, etc.) and middleware management.
type Server struct {
	// Router provides HTTP routing functionality including method-specific
	// route registration, middleware support, and request handling.
	Router

	// mu protects concurrent access to server fields during startup,
	// shutdown, and configuration operations.
	mu sync.RWMutex

	// baseCtx is the root context for all requests.
	// It is cancelled when Shutdown is called to signal request cancellation.
	baseCtx context.Context

	// cancelBaseCtx cancels the base context.
	cancelBaseCtx context.CancelFunc

	// server is the HTTP server instance for handling plain HTTP traffic.
	// If nil, HTTP server will not be started.
	server *http.Server

	// listener is the network listener for HTTP traffic. If nil, a default
	// listener will be created using the server's configured address.
	listener net.Listener

	// tlsServer is the HTTPS server instance for handling encrypted traffic.
	// If nil, HTTPS server will not be started.
	tlsServer *http.Server

	// tlsListener is the network listener for HTTPS traffic. If nil, a default
	// TLS listener will be created using the tlsServer's configured address.
	tlsListener net.Listener

	// certFile is the file path to the TLS certificate in PEM format.
	// Used when serving HTTPS traffic with certificate files.
	certFile string

	// keyFile is the file path to the TLS private key in PEM format.
	// Used when serving HTTPS traffic with certificate files.
	keyFile string

	// logger is the structured logger used by the server and its middleware
	// for recording HTTP requests, errors, and server lifecycle events.
	logger log.Logger

	// preShutdownHooks execute sequentially before server shutdown begins.
	preShutdownHooks []config.ShutdownHookConfig

	// shutdownHooks execute concurrently with server shutdown.
	shutdownHooks []config.ShutdownHookConfig

	// postShutdownHooks execute sequentially after all servers are shut down.
	postShutdownHooks []config.ShutdownHookConfig

	// validator is an optional struct validator for validating request data.
	// Users can inject their own implementation (e.g., go-playground/validator/v10).
	// If nil, the default built-in validator will be used.
	validator config.Validator

	// metricsRegistry holds the metrics registry for collecting and exposing metrics.
	// If nil, metrics collection is disabled.
	metricsRegistry metrics.Registry

	// metricsServer is a dedicated HTTP server for serving metrics.
	// When Metrics.ServerAddr is set, metrics are served on this separate server
	// bound to the specified address (typically localhost for security).
	metricsServer *http.Server

	// metricsListener is the network listener for the metrics server.
	metricsListener net.Listener

	// metricsServerAddr is the configured address for the metrics server.
	metricsServerAddr string

	// autocertManager handles automatic certificate provisioning and renewal
	// using Let's Encrypt ACME protocol. If set, enables automatic TLS.
	// Users must provide their own implementation (e.g., golang.org/x/crypto/acme/autocert.Manager).
	autocertManager config.AutocertManager

	// http3Server is an optional HTTP/3 server for handling HTTP/3 traffic over QUIC.
	// Users can inject their own implementation (e.g., quic-go/http3) to enable HTTP/3.
	// If nil, HTTP/3 server will not be started.
	http3Server config.HTTP3Server

	// sseProvider is an optional SSE provider for handling Server-Sent Events connections.
	// Users can inject their own implementation or use the built-in stdlib provider.
	// If nil, SSE is not available but users can still handle SSE manually in their handlers.
	sseProvider config.SSEProvider

	// webSocketUpgrader is an optional WebSocket upgrader for handling WebSocket connections.
	// Users provide their own implementation using their preferred WebSocket library.
	// If nil, WebSocket is not available but users can still handle upgrades manually.
	webSocketUpgrader config.WebSocketUpgrader

	// webTransportServer is an optional WebTransport server for handling WebTransport sessions.
	// Users can inject their own implementation (e.g., quic-go/webtransport-go) to enable WebTransport.
	// If nil, WebTransport support will not be enabled.
	// The server will be started automatically when ListenAndServeTLS or Start is called.
	webTransportServer config.WebTransportServer
}

// mergeRecoverConfig merges user config with defaults
func mergeRecoverConfig(defaultCfg, userCfg config.RecoverConfig) config.RecoverConfig {
	if userCfg.StackSize != 0 {
		defaultCfg.StackSize = userCfg.StackSize
	}
	if userCfg.EnableStackTrace {
		defaultCfg.EnableStackTrace = userCfg.EnableStackTrace
	}
	return defaultCfg
}

// mergeRequestBodySizeConfig merges user config with defaults
func mergeRequestBodySizeConfig(defaultCfg, userCfg config.RequestBodySizeConfig) config.RequestBodySizeConfig {
	if userCfg.MaxBytes != 0 {
		defaultCfg.MaxBytes = userCfg.MaxBytes
	}
	if len(userCfg.ExemptPaths) > 0 {
		defaultCfg.ExemptPaths = userCfg.ExemptPaths
	}
	return defaultCfg
}

// mergeRequestIDConfig merges user config with defaults
func mergeRequestIDConfig(defaultCfg, userCfg config.RequestIDConfig) config.RequestIDConfig {
	if userCfg.Header != "" {
		defaultCfg.Header = userCfg.Header
	}
	if userCfg.Generator != nil {
		defaultCfg.Generator = userCfg.Generator
	}
	if userCfg.ContextKey != "" {
		defaultCfg.ContextKey = userCfg.ContextKey
	}
	return defaultCfg
}

// mergeRequestLoggerConfig merges user config with defaults
func mergeRequestLoggerConfig(defaultCfg, userCfg config.RequestLoggerConfig) config.RequestLoggerConfig {
	if userCfg.LogErrors {
		defaultCfg.LogErrors = userCfg.LogErrors
	}
	if len(userCfg.Fields) > 0 {
		defaultCfg.Fields = userCfg.Fields
	}
	if len(userCfg.ExemptPaths) > 0 {
		defaultCfg.ExemptPaths = userCfg.ExemptPaths
	}
	return defaultCfg
}

// mergeSecurityHeadersConfig merges user config with defaults
func mergeSecurityHeadersConfig(defaultCfg, userCfg config.SecurityHeadersConfig) config.SecurityHeadersConfig {
	if userCfg.ContentSecurityPolicy != "" {
		defaultCfg.ContentSecurityPolicy = userCfg.ContentSecurityPolicy
	}
	defaultCfg.ContentSecurityPolicyReportOnly = userCfg.ContentSecurityPolicyReportOnly
	if userCfg.CrossOriginEmbedderPolicy != "" {
		defaultCfg.CrossOriginEmbedderPolicy = userCfg.CrossOriginEmbedderPolicy
	}
	if userCfg.CrossOriginOpenerPolicy != "" {
		defaultCfg.CrossOriginOpenerPolicy = userCfg.CrossOriginOpenerPolicy
	}
	if userCfg.CrossOriginResourcePolicy != "" {
		defaultCfg.CrossOriginResourcePolicy = userCfg.CrossOriginResourcePolicy
	}
	if userCfg.PermissionsPolicy != "" {
		defaultCfg.PermissionsPolicy = userCfg.PermissionsPolicy
	}
	if userCfg.ReferrerPolicy != "" {
		defaultCfg.ReferrerPolicy = userCfg.ReferrerPolicy
	}
	if userCfg.Server != "" {
		defaultCfg.Server = userCfg.Server
	}
	if userCfg.StrictTransportSecurity.MaxAge != 0 {
		defaultCfg.StrictTransportSecurity = userCfg.StrictTransportSecurity
	}
	if userCfg.XContentTypeOptions != "" {
		defaultCfg.XContentTypeOptions = userCfg.XContentTypeOptions
	}
	if userCfg.XFrameOptions != "" {
		defaultCfg.XFrameOptions = userCfg.XFrameOptions
	}
	if len(userCfg.ExemptPaths) > 0 {
		defaultCfg.ExemptPaths = userCfg.ExemptPaths
	}
	return defaultCfg
}

// mergeMetricsConfig merges user config with defaults
func mergeMetricsConfig(defaultCfg, userCfg config.MetricsConfig) config.MetricsConfig {
	defaultCfg.Enabled = userCfg.Enabled
	if userCfg.Endpoint != "" {
		defaultCfg.Endpoint = userCfg.Endpoint
	}
	// ServerAddr can be explicitly set to empty string to disable separate metrics server
	defaultCfg.ServerAddr = userCfg.ServerAddr
	if len(userCfg.DurationBuckets) > 0 {
		defaultCfg.DurationBuckets = userCfg.DurationBuckets
	}
	if len(userCfg.SizeBuckets) > 0 {
		defaultCfg.SizeBuckets = userCfg.SizeBuckets
	}
	if len(userCfg.ExcludePaths) > 0 {
		defaultCfg.ExcludePaths = userCfg.ExcludePaths
	}
	if userCfg.PathLabelFunc != nil {
		defaultCfg.PathLabelFunc = userCfg.PathLabelFunc
	}
	if userCfg.CustomLabels != nil {
		defaultCfg.CustomLabels = userCfg.CustomLabels
	}
	return defaultCfg
}

// New creates and configures a new Server instance with the provided config.
// It initializes the server with default configurations that can be overridden
// using the provided config. The server includes HTTP and HTTPS support,
// middleware integration, and structured logging.
//
// Example usage:
//
//	// Use defaults
//	server := zerohttp.New()
//
//	// With custom config
//	server := zerohttp.New(config.Config{
//	    Addr: ":8080",
//	    Logger: myLogger,
//	})
func New(cfg ...config.Config) *Server {
	c := config.DefaultConfig
	if len(cfg) > 0 {
		userCfg := cfg[0]
		if userCfg.Addr != "" {
			c.Addr = userCfg.Addr
		}
		if userCfg.TLSAddr != "" {
			c.TLSAddr = userCfg.TLSAddr
		}
		if userCfg.Server != nil {
			c.Server = userCfg.Server
		}
		if userCfg.Listener != nil {
			c.Listener = userCfg.Listener
		}
		if userCfg.TLSServer != nil {
			c.TLSServer = userCfg.TLSServer
		}
		if userCfg.TLSListener != nil {
			c.TLSListener = userCfg.TLSListener
		}
		if userCfg.CertFile != "" {
			c.CertFile = userCfg.CertFile
		}
		if userCfg.KeyFile != "" {
			c.KeyFile = userCfg.KeyFile
		}
		if userCfg.Logger != nil {
			c.Logger = userCfg.Logger
		}
		if len(userCfg.PreShutdownHooks) > 0 {
			c.PreShutdownHooks = userCfg.PreShutdownHooks
		}
		if len(userCfg.ShutdownHooks) > 0 {
			c.ShutdownHooks = userCfg.ShutdownHooks
		}
		if len(userCfg.PostShutdownHooks) > 0 {
			c.PostShutdownHooks = userCfg.PostShutdownHooks
		}
		c.DisableDefaultMiddlewares = userCfg.DisableDefaultMiddlewares
		if len(userCfg.DefaultMiddlewares) > 0 {
			c.DefaultMiddlewares = userCfg.DefaultMiddlewares
		}
		c.Recover = mergeRecoverConfig(c.Recover, userCfg.Recover)
		c.RequestBodySize = mergeRequestBodySizeConfig(c.RequestBodySize, userCfg.RequestBodySize)
		c.RequestID = mergeRequestIDConfig(c.RequestID, userCfg.RequestID)
		c.RequestLogger = mergeRequestLoggerConfig(c.RequestLogger, userCfg.RequestLogger)
		c.SecurityHeaders = mergeSecurityHeadersConfig(c.SecurityHeaders, userCfg.SecurityHeaders)
		if userCfg.AutocertManager != nil {
			c.AutocertManager = userCfg.AutocertManager
		}
		if userCfg.HTTP3Server != nil {
			c.HTTP3Server = userCfg.HTTP3Server
		}
		if userCfg.SSEProvider != nil {
			c.SSEProvider = userCfg.SSEProvider
		}
		if userCfg.WebSocketUpgrader != nil {
			c.WebSocketUpgrader = userCfg.WebSocketUpgrader
		}
		if userCfg.WebTransportServer != nil {
			c.WebTransportServer = userCfg.WebTransportServer
		}
		if userCfg.Validator != nil {
			c.Validator = userCfg.Validator
		}
		c.Metrics = mergeMetricsConfig(c.Metrics, userCfg.Metrics)
	}

	router := NewRouter()

	logger := c.Logger
	if logger == nil {
		logger = log.NewDefaultLogger()
	}

	router.SetLogger(logger)
	router.SetConfig(c)

	server := c.Server
	if server == nil {
		server = &http.Server{
			Addr:           c.Addr,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1 MB
		}
	}

	tlsServer := c.TLSServer
	if tlsServer == nil {
		tlsServer = &http.Server{
			Addr:           c.TLSAddr,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1 MB
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				NextProtos: []string{"h2", "http/1.1"},
			},
		}
	}

	// Initialize metrics registry if enabled
	var registry metrics.Registry
	if c.Metrics.Enabled {
		registry = metrics.NewRegistry()
	}

	// Create base context for request cancellation during shutdown
	baseCtx, cancelBaseCtx := context.WithCancel(context.Background())

	s := &Server{
		Router:             router,
		server:             server,
		listener:           c.Listener,
		tlsServer:          tlsServer,
		tlsListener:        c.TLSListener,
		certFile:           c.CertFile,
		keyFile:            c.KeyFile,
		autocertManager:    c.AutocertManager,
		http3Server:        c.HTTP3Server,
		webTransportServer: c.WebTransportServer,
		webSocketUpgrader:  c.WebSocketUpgrader,
		sseProvider:        c.SSEProvider,
		metricsRegistry:    registry,
		metricsServerAddr:  c.Metrics.ServerAddr,
		validator:          c.Validator,
		logger:             logger,
		preShutdownHooks:   c.PreShutdownHooks,
		shutdownHooks:      c.ShutdownHooks,
		postShutdownHooks:  c.PostShutdownHooks,
		baseCtx:            baseCtx,
		cancelBaseCtx:      cancelBaseCtx,
	}

	// Configure separate metrics server if ServerAddr is set
	if c.Metrics.Enabled && registry != nil && c.Metrics.ServerAddr != "" {
		s.metricsServer = &http.Server{
			Addr:         c.Metrics.ServerAddr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
			Handler:      metrics.Handler(registry),
		}
	}

	var middlewares []func(http.Handler) http.Handler

	// Add metrics middleware first so it will be innermost after reverse,
	// running inside Recover and able to capture status codes written by other middleware
	if c.Metrics.Enabled && registry != nil {
		middlewares = append(middlewares, metrics.NewMiddleware(registry, c.Metrics))
	}

	if c.DisableDefaultMiddlewares {
		middlewares = append(middlewares, c.DefaultMiddlewares...)
	} else if c.DefaultMiddlewares == nil {
		middlewares = append(middlewares, middleware.DefaultMiddlewares(c, s.logger)...)
	} else {
		defaults := middleware.DefaultMiddlewares(c, s.logger)
		middlewares = append(middlewares, defaults...)
		middlewares = append(middlewares, c.DefaultMiddlewares...)
	}

	if len(middlewares) > 0 {
		s.Use(middlewares...)
	}

	if s.server != nil {
		s.server.Handler = router
		s.server.BaseContext = func(net.Listener) context.Context {
			return s.baseCtx
		}
	}

	if s.tlsServer != nil {
		s.tlsServer.Handler = router
		s.tlsServer.BaseContext = func(net.Listener) context.Context {
			return s.baseCtx
		}
	}

	// Register metrics endpoint on main router only if no separate metrics server
	if c.Metrics.Enabled && registry != nil && c.Metrics.ServerAddr == "" {
		s.logger.Warn("Metrics endpoint registered on main server (set Metrics.ServerAddr to isolate)", log.F("endpoint", c.Metrics.Endpoint))
		s.GET(c.Metrics.Endpoint, metrics.Handler(registry))
	}

	// Finalize router to register catch-all handler with middleware
	if r, ok := router.(interface{ finalize() }); ok {
		r.finalize()
	}

	return s
}

// ListenAndServe starts the HTTP server and begins accepting connections.
// It creates a listener if one is not already configured and serves HTTP
// traffic on the configured address. If the server is not configured,
// this method logs a debug message and returns nil without error.
//
// This method blocks until the server encounters an error or is shut down.
// Returns any error encountered while starting or running the server.
func (s *Server) ListenAndServe() error {
	s.mu.Lock()

	if s.server == nil {
		s.mu.Unlock()
		s.logger.Debug("HTTP server not configured, skipping")
		return nil
	}

	var err error
	if s.listener == nil {
		s.logger.Debug("Creating HTTP listener", log.F("addr", s.server.Addr))
		s.listener, err = net.Listen("tcp", s.server.Addr)
		if err != nil {
			s.mu.Unlock()
			return err
		}
	}

	s.mu.Unlock()

	s.logger.Info("Starting HTTP server", log.F("addr", fmtHTTPAddr(s.listener.Addr().String())))

	return s.server.Serve(s.listener)
}

// ListenAndServeTLS starts the HTTPS server with the specified certificate files.
// It creates a TLS listener if one is not already configured and serves HTTPS
// traffic using the provided certificate and key files. If the TLS server is
// not configured, this method logs a debug message and returns nil without error.
//
// Parameters:
//   - certFile: Path to the TLS certificate file in PEM format
//   - keyFile: Path to the TLS private key file in PEM format
//
// This method blocks until the server encounters an error or is shut down.
// Returns any error encountered while starting or running the TLS server.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	s.mu.Lock()

	if s.tlsServer == nil {
		s.mu.Unlock()
		s.logger.Debug("TLS server not configured, skipping")
		return nil
	}

	s.logger.Debug("TLS server is configured, proceeding")

	// Load certificates if provided
	if certFile != "" && keyFile != "" {
		s.logger.Debug("Loading TLS certificates", log.F("cert", certFile), log.F("key", keyFile))
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			s.mu.Unlock()
			s.logger.Error("Failed to load TLS certificates", log.E(err))
			return fmt.Errorf("failed to load certificates: %w", err)
		}
		if s.tlsServer.TLSConfig == nil {
			s.tlsServer.TLSConfig = &tls.Config{}
		}
		s.tlsServer.TLSConfig.Certificates = []tls.Certificate{cert}
	}

	var err error
	if s.tlsListener == nil {
		s.logger.Debug("Creating TLS listener", log.F("addr", s.tlsServer.Addr))
		s.tlsListener, err = tls.Listen("tcp", s.tlsServer.Addr, s.tlsServer.TLSConfig)
		if err != nil {
			s.logger.Error("Failed to create TLS listener", log.E(err))
			s.mu.Unlock()
			return err
		}
		s.logger.Debug("TLS listener created successfully")
	}

	s.mu.Unlock()

	s.logger.Info("Starting HTTPS server",
		log.F("addr", fmtHTTPSAddr(s.tlsListener.Addr().String())),
		log.F("cert_file", certFile),
		log.F("key_file", keyFile))

	// Start HTTP/3 server in background if configured
	if s.http3Server != nil {
		go func() {
			s.logger.Info("Starting HTTP/3 server",
				log.F("cert_file", certFile),
				log.F("key_file", keyFile))
			if err := s.http3Server.ListenAndServeTLS(certFile, keyFile); err != nil {
				s.logger.Error("HTTP/3 server error", log.E(err))
			}
		}()
	}

	// Start WebTransport server in background if configured
	if s.webTransportServer != nil {
		go func() {
			s.logger.Info("Starting WebTransport server",
				log.F("cert_file", certFile),
				log.F("key_file", keyFile))
			if err := s.webTransportServer.ListenAndServeTLS(certFile, keyFile); err != nil {
				s.logger.Error("WebTransport server error", log.E(err))
			}
		}()
	}

	// Use Serve (not ServeTLS) since we already have a tls.Listener
	return s.tlsServer.Serve(s.tlsListener)
}

// Start begins serving both HTTP and HTTPS traffic concurrently.
// It starts the HTTP server (if configured) and the HTTPS server (if configured
// with certificates or TLS config). The method returns when the first server
// encounters an error.
//
// For HTTPS, the server will start if:
//   - TLS server is configured AND
//   - Either certificates are loaded in TLS config OR certificate files are specified
//
// This method is non-blocking for individual servers but blocks until one fails.
// Returns the first error encountered by any server during startup or operation.
func (s *Server) Start() error {
	s.logger.Info("Starting server...")
	errCh := make(chan error, 4)

	handler := s.Router

	// Start metrics server if configured
	if s.metricsServer != nil {
		go func() {
			s.logger.Info("Starting metrics server...", log.F("addr", fmtHTTPAddr(s.metricsServer.Addr)))
			if err := s.startMetricsServer(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("metrics server error: %w", err)
			}
		}()
	}

	// Start HTTP server
	if s.server != nil {
		s.server.Handler = handler
		go func() {
			s.logger.Info("Starting HTTP server...", log.F("addr", fmtHTTPAddr(s.server.Addr)))
			if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()
	}

	// Determine if we should start HTTPS server
	shouldStartTLS := s.tlsServer != nil &&
		((s.tlsServer.TLSConfig != nil &&
			(len(s.tlsServer.TLSConfig.Certificates) > 0 || s.tlsServer.TLSConfig.GetCertificate != nil)) ||
			(s.certFile != "" && s.keyFile != ""))

	// Start HTTPS server
	if shouldStartTLS {
		if s.tlsServer.TLSConfig == nil {
			s.tlsServer.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
		s.tlsServer.Handler = handler

		// Load cert/key if paths are specified
		if s.certFile != "" && s.keyFile != "" &&
			(len(s.tlsServer.TLSConfig.Certificates) == 0 && s.tlsServer.TLSConfig.GetCertificate == nil) {
			cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
			if err != nil {
				s.logger.Error("Failed to load TLS certificate", log.E(err))
				errCh <- fmt.Errorf("failed to load certificate files: %w", err)
				return <-errCh
			}
			s.tlsServer.TLSConfig.Certificates = []tls.Certificate{cert}
		}

		go func() {
			s.logger.Info("Starting HTTPS server...",
				log.F("addr", fmtHTTPSAddr(s.tlsServer.Addr)),
				log.F("cert_file", s.certFile),
				log.F("key_file", s.keyFile))
			if err := s.tlsServer.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("HTTPS server error: %w", err)
			}
		}()
	}

	// Start HTTP/3 server if configured and we have TLS
	if s.http3Server != nil && shouldStartTLS {
		go func() {
			s.logger.Info("Starting HTTP/3 server...",
				log.F("cert_file", s.certFile),
				log.F("key_file", s.keyFile))
			if err := s.http3Server.ListenAndServeTLS(s.certFile, s.keyFile); err != nil {
				s.logger.Error("HTTP/3 server error", log.E(err))
			}
		}()
	}

	// Start WebTransport server if configured and we have TLS
	if s.webTransportServer != nil && shouldStartTLS {
		go func() {
			s.logger.Info("Starting WebTransport server...",
				log.F("cert_file", s.certFile),
				log.F("key_file", s.keyFile))
			if err := s.webTransportServer.ListenAndServeTLS(s.certFile, s.keyFile); err != nil {
				s.logger.Error("WebTransport server error", log.E(err))
			}
		}()
	}

	// Wait for any server to return error
	return <-errCh
}

// StartTLS is a convenience method that starts only the HTTPS server with
// the specified certificate files. If the TLS server is not configured,
// this method returns nil without error.
//
// Parameters:
//   - certFile: Path to the TLS certificate file in PEM format
//   - keyFile: Path to the TLS private key file in PEM format
//
// This is equivalent to calling ListenAndServeTLS directly.
// Returns any error encountered while starting or running the TLS server.
func (s *Server) StartTLS(certFile, keyFile string) error {
	if s.tlsServer == nil {
		return fmt.Errorf("TLS server not configured")
	}

	return s.ListenAndServeTLS(certFile, keyFile)
}

// StartAutoTLS starts the server with automatic TLS certificate management using Let's Encrypt.
// It starts both HTTP (for ACME challenges) and HTTPS servers.
// The HTTP server redirects to HTTPS and handles ACME challenges.
//
// Users must configure the AutocertManager with their desired host policy before calling
// this method. For example, using golang.org/x/crypto/acme/autocert:
//
//	mgr := &autocert.Manager{
//	    Cache:      autocert.DirCache("/var/cache/certs"),
//	    Prompt:     autocert.AcceptTOS,
//	    HostPolicy: autocert.HostWhitelist("example.com"),
//	}
//	srv := zerohttp.New(config.WithAutocertManager(mgr))
//	srv.StartAutoTLS()
//
// The HTTP server handles:
//   - ACME challenge requests from Let's Encrypt
//   - Redirects all other HTTP traffic to HTTPS
//
// Returns an error if the autocert manager is not configured or if any server fails to start.
func (s *Server) StartAutoTLS() error {
	if s.autocertManager == nil {
		return fmt.Errorf("autocert manager not configured")
	}

	s.logger.Info("Starting server with AutoTLS...")

	errCh := make(chan error, 4)
	httpReady := make(chan struct{})

	if s.server == nil {
		close(httpReady)
	}

	// Start HTTP server for ACME challenges and redirects
	if s.server != nil {
		go func() {
			// Create a new server for HTTP with autocert handler
			httpServer := &http.Server{
				Addr:    s.server.Addr,
				Handler: s.autocertManager.HTTPHandler(s.createHTTPSRedirectHandler()),
			}

			ln, err := net.Listen("tcp", httpServer.Addr)
			if err != nil {
				s.logger.Error("Failed to bind HTTP listener", log.E(err))
				errCh <- err
				return
			}

			s.logger.Info("Starting HTTP server for ACME challenges and redirects",
				log.F("addr", fmtHTTPAddr(httpServer.Addr)))
			close(httpReady)
			errCh <- httpServer.Serve(ln)
		}()
	}

	certReady := make(chan struct{})
	var certOnce sync.Once
	signalCertReady := func() {
		certOnce.Do(func() {
			s.logger.Info("AutoTLS certificate is ready")
			close(certReady)
		})
	}

	// Start HTTPS server with autocert
	if s.tlsServer != nil {
		go func() {
			// Configure TLS with autocert
			if s.tlsServer.TLSConfig == nil {
				s.tlsServer.TLSConfig = &tls.Config{}
			}
			s.tlsServer.TLSConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert, err := s.autocertManager.GetCertificate(hello)
				if err == nil {
					// Signal that cert is ready (first successful retrieval)
					signalCertReady()
				}
				return cert, err
			}

			s.logger.Info("Starting HTTPS server with AutoTLS",
				log.F("addr", fmtHTTPSAddr(s.tlsServer.Addr)))
			errCh <- s.tlsServer.ListenAndServeTLS("", "")
		}()
	}

	// Warm-up goroutine: proactively fetch certificate for HTTP/3/WebTransport
	if s.http3Server != nil || s.webTransportServer != nil {
		go func() {
			<-httpReady
			hostnames := s.autocertManager.Hostnames()
			if len(hostnames) == 0 {
				s.logger.Error("AutocertManager returned no hostnames, cannot warm up certificate")
				return
			}

			hello := &tls.ClientHelloInfo{ServerName: hostnames[0]}

			// Attempt immediately before starting the ticker loop
			// so a cached cert on restart doesn't incur a 2-second delay
			cert, err := s.autocertManager.GetCertificate(hello)
			if err == nil && cert != nil {
				signalCertReady()
				return
			}
			s.logger.Debug("Certificate not yet ready on first attempt, starting poll loop...", log.E(err))

			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			timeout := time.After(5 * time.Minute)

			for {
				select {
				case <-ticker.C:
					cert, err := s.autocertManager.GetCertificate(hello)
					if err != nil {
						s.logger.Debug("Certificate not yet ready, retrying...", log.E(err))
						continue
					}
					if cert != nil {
						signalCertReady()
						return
					}
				case <-timeout:
					s.logger.Error("Timed out waiting for AutoTLS certificate")
					return
				}
			}
		}()
	}

	// Start HTTP/3 server with autocert if supported (after cert is ready)
	if s.http3Server != nil {
		if h3Autocert, ok := s.http3Server.(config.HTTP3ServerWithAutocert); ok {
			go func() {
				s.logger.Info("Waiting for certificate before starting HTTP/3...")
				<-certReady
				s.logger.Info("Starting HTTP/3 server with AutoTLS")
				errCh <- h3Autocert.ListenAndServeTLSWithAutocert(s.autocertManager)
			}()
		}
	}

	// Start WebTransport server with autocert if supported (after cert is ready)
	if s.webTransportServer != nil {
		if wtAutocert, ok := s.webTransportServer.(config.WebTransportServerWithAutocert); ok {
			go func() {
				s.logger.Info("Waiting for certificate before starting WebTransport...")
				<-certReady
				s.logger.Info("Starting WebTransport server with AutoTLS")
				errCh <- wtAutocert.ListenAndServeTLSWithAutocert(s.autocertManager)
			}()
		}
	}

	return <-errCh
}

// ListenerAddr returns the network address that the HTTP server is listening on.
// If a listener is configured, it returns the listener's actual address.
// If no listener is configured but a server is configured, it returns the server's configured address.
// If neither is configured, it returns an empty string.
//
// This method is thread-safe and can be called concurrently.
// The returned address includes both host and port (e.g., "127.0.0.1:8080").
func (s *Server) ListenerAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}

	if s.server != nil {
		return s.server.Addr
	}

	return ""
}

// ListenerTLSAddr returns the network address that the HTTPS server is listening on.
// If a TLS listener is configured, it returns the listener's actual address.
// If no TLS listener is configured but a TLS server is configured, it returns the server's configured address.
// If neither is configured, it returns an empty string.
//
// This method is thread-safe and can be called concurrently.
// The returned address includes both host and port (e.g., "127.0.0.1:8443").
func (s *Server) ListenerTLSAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tlsListener != nil {
		return s.tlsListener.Addr().String()
	}

	if s.tlsServer != nil {
		return s.tlsServer.Addr
	}

	return ""
}

// Logger returns the structured logger instance used by the server.
// This logger is used for recording HTTP requests, errors, server lifecycle events,
// and can be used by application code for consistent logging.
//
// The returned logger implements the log.Logger interface and provides
// structured logging capabilities with fields and different log levels.
func (s *Server) Logger() log.Logger {
	return s.logger
}

// RegisterPreShutdownHook registers a hook to run before server shutdown begins.
// Pre-shutdown hooks execute sequentially in registration order.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	app.RegisterPreShutdownHook("health", func(ctx context.Context) error {
//	    health.SetUnhealthy()
//	    return nil
//	})
func (s *Server) RegisterPreShutdownHook(name string, hook config.ShutdownHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preShutdownHooks = append(s.preShutdownHooks, config.ShutdownHookConfig{Name: name, Hook: hook})
}

// RegisterShutdownHook registers a hook to run concurrently with server shutdown.
// Shutdown hooks execute concurrently alongside server shutdown.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	app.RegisterShutdownHook("close-db", func(ctx context.Context) error {
//	    return db.Close()
//	})
func (s *Server) RegisterShutdownHook(name string, hook config.ShutdownHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownHooks = append(s.shutdownHooks, config.ShutdownHookConfig{Name: name, Hook: hook})
}

// RegisterPostShutdownHook registers a hook to run after servers are shut down.
// Post-shutdown hooks execute sequentially in registration order.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	app.RegisterPostShutdownHook("cleanup", func(ctx context.Context) error {
//	    return os.RemoveAll("/tmp/app-*")
//	})
func (s *Server) RegisterPostShutdownHook(name string, hook config.ShutdownHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.postShutdownHooks = append(s.postShutdownHooks, config.ShutdownHookConfig{Name: name, Hook: hook})
}

// SetValidator sets the struct validator instance. This can be used to inject
// a custom validation implementation (e.g., go-playground/validator/v10) after
// creating the server. If nil, the default built-in validator will be used.
//
// Example:
//
//	import "github.com/go-playground/validator/v10"
//
//	app := zerohttp.New()
//	app.SetValidator(&myValidator{v: validator.New()})
//
// Parameters:
//   - validator: A validator instance implementing the config.Validator interface
func (s *Server) SetValidator(validator config.Validator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.validator = validator
}

// Validator returns the configured struct validator (if any).
// Returns nil if no custom validator has been configured - in this case,
// the default built-in validator (zh.V) should be used.
func (s *Server) Validator() config.Validator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.validator
}

// Metrics returns the metrics registry for collecting custom metrics.
// Returns nil if metrics are not enabled.
//
// Use this to create custom metrics in your handlers or middleware:
//
//	requests := app.Metrics().Counter("my_requests_total", "status")
//	requests.WithLabelValues("200").Inc()
func (s *Server) Metrics() metrics.Registry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metricsRegistry
}

// ListenAndServeHTTP3 starts the HTTP/3 server with the specified certificate files.
// HTTP/3 requires TLS and uses the provided certificate and key files for encryption.
// If the HTTP/3 server is not configured, this method logs a debug message and returns nil without error.
//
// Parameters:
//   - certFile: Path to the TLS certificate file in PEM format
//   - keyFile: Path to the TLS private key file in PEM format
//
// This method blocks until the server encounters an error or is shut down.
// Returns any error encountered while starting or running the HTTP/3 server.
func (s *Server) ListenAndServeHTTP3(certFile, keyFile string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.http3Server == nil {
		s.logger.Debug("HTTP/3 server not configured, skipping")
		return nil
	}

	s.logger.Info("Starting HTTP/3 server",
		log.F("cert_file", certFile),
		log.F("key_file", keyFile))

	return s.http3Server.ListenAndServeTLS(certFile, keyFile)
}

// StartHTTP3 starts only the HTTP/3 server with the specified certificate files.
// This is a convenience method for starting just HTTP/3 without HTTP or HTTPS.
// If the HTTP/3 server is not configured, this method returns nil without error.
//
// Parameters:
//   - certFile: Path to the TLS certificate file in PEM format
//   - keyFile: Path to the TLS private key file in PEM format
//
// This is equivalent to calling ListenAndServeHTTP3 directly.
// Returns any error encountered while starting or running the HTTP/3 server.
func (s *Server) StartHTTP3(certFile, keyFile string) error {
	return s.ListenAndServeHTTP3(certFile, keyFile)
}

// SetHTTP3Server sets the HTTP/3 server instance. This can be used to inject
// an HTTP/3 implementation (e.g., quic-go/http3) after creating the server.
//
// The HTTP/3 server will be started automatically when ListenAndServeTLS or StartTLS
// is called. You don't need to call ListenAndServeTLS on the HTTP/3 server yourself.
//
// Parameters:
//   - server: An HTTP/3 server instance implementing the config.HTTP3Server interface
func (s *Server) SetHTTP3Server(server config.HTTP3Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.http3Server = server
}

// SetSSEProvider sets the SSE provider instance. This can be used to inject
// an SSE implementation after creating the server.
//
// Users can implement their own SSE provider or use the built-in stdlib provider:
//
//	app := zerohttp.New()
//	app.SetSSEProvider(zh.NewDefaultProvider())
//
//	app.GET("/events", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    provider := app.SSEProvider()
//	    sse, err := provider.NewSSE(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer sse.Close()
//	    // ... stream events ...
//	}))
//
// Parameters:
//   - provider: An SSE provider instance implementing the config.SSEProvider interface
func (s *Server) SetSSEProvider(provider config.SSEProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sseProvider = provider
}

// SSEProvider returns the configured SSE provider (if any).
// Returns nil if no SSE provider has been configured.
func (s *Server) SSEProvider() config.SSEProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sseProvider
}

// SetWebSocketUpgrader sets the WebSocket upgrader instance. This can be used to inject
// a WebSocket implementation (e.g., gorilla/websocket, nhooyr/websocket) after creating the server.
//
// The WebSocket upgrader provides the Upgrade method for handling WebSocket connections.
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
//	app := zerohttp.New()
//	app.SetWebSocketUpgrader(&myUpgrader{upgrader})
//
//	app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    ws, err := app.WebSocketUpgrader().Upgrade(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer ws.Close()
//	    // ... handle connection ...
//	}))
//
// Parameters:
//   - upgrader: A WebSocket upgrader instance implementing the config.WebSocketUpgrader interface
func (s *Server) SetWebSocketUpgrader(upgrader config.WebSocketUpgrader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webSocketUpgrader = upgrader
}

// WebSocketUpgrader returns the configured WebSocket upgrader (if any).
// Returns nil if no WebSocket upgrader has been configured.
func (s *Server) WebSocketUpgrader() config.WebSocketUpgrader {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.webSocketUpgrader
}

// SetWebTransportServer sets the WebTransport server instance. This can be used to inject
// a WebTransport implementation (e.g., quic-go/webtransport-go) after creating the server.
//
// The WebTransport server will be started automatically when ListenAndServeTLS or Start
// is called. You don't need to call ListenAndServeTLS on the WebTransport server yourself.
//
// Parameters:
//   - server: A WebTransport server instance implementing the config.WebTransportServer interface
func (s *Server) SetWebTransportServer(server config.WebTransportServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webTransportServer = server
}

// Shutdown gracefully shuts down both HTTP and HTTPS servers without interrupting
// any active connections. It waits for active connections to finish or for the
// provided context to be cancelled.
//
// Parameters:
//   - ctx: Context that controls the shutdown timeout. If the context is cancelled
//     before shutdown completes, the servers will be forcefully closed.
//
// The shutdown process runs concurrently for both servers. If any server
// encounters an error during shutdown, that error is returned.
// Shutdown hooks are executed during the shutdown process:
//   - Pre-shutdown hooks run sequentially before server shutdown begins
//   - Shutdown hooks run concurrently with server shutdown
//   - Post-shutdown hooks run sequentially after all servers are shut down
//
// Returns the first error encountered during shutdown, or nil if successful.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	// Cancel the base context to signal all requests to close
	// This happens before pre-shutdown hooks so requests can start terminating
	if s.cancelBaseCtx != nil {
		s.cancelBaseCtx()
	}

	// Execute pre-shutdown hooks sequentially
	if err := s.runPreShutdownHooks(ctx); err != nil {
		s.logger.Error("Pre-shutdown hook error", log.E(err))
		// Return context errors as they indicate shutdown was cancelled
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
	}

	// Start shutdown hooks concurrently and wait for them
	hookWg, hookErrCh := s.startShutdownHooks(ctx)

	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	if s.server != nil && s.listener != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Shutting down HTTP server")
			if err := s.server.Shutdown(ctx); err != nil {
				s.logger.Error("Error shutting down HTTP server", log.F("error", err))
				errCh <- err
			} else {
				s.logger.Info("HTTP server shutdown complete")
			}
		}()
	}

	if s.tlsServer != nil && s.tlsListener != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Shutting down HTTPS server")
			if err := s.tlsServer.Shutdown(ctx); err != nil {
				s.logger.Error("Error shutting down HTTPS server", log.F("error", err))
				errCh <- err
			} else {
				s.logger.Info("HTTPS server shutdown complete")
			}
		}()
	}

	// Shutdown WebTransport first since it depends on HTTP/3
	if s.webTransportServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Closing WebTransport server")
			if err := s.webTransportServer.Close(); err != nil {
				s.logger.Error("Error closing WebTransport server", log.F("error", err))
				errCh <- err
			} else {
				s.logger.Info("WebTransport server closed")
			}
		}()
	}

	if s.http3Server != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Shutting down HTTP/3 server")
			if err := s.http3Server.Shutdown(ctx); err != nil {
				s.logger.Error("Error shutting down HTTP/3 server", log.F("error", err))
				errCh <- err
			} else {
				s.logger.Info("HTTP/3 server shutdown complete")
			}
		}()
	}

	// Shutdown metrics server
	if s.metricsServer != nil && s.metricsListener != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Shutting down metrics server")
			if err := s.metricsServer.Shutdown(ctx); err != nil {
				s.logger.Error("Error shutting down metrics server", log.F("error", err))
				errCh <- err
			} else {
				s.logger.Info("Metrics server shutdown complete")
			}
		}()
	}

	wg.Wait()
	close(errCh)

	// Wait for shutdown hooks to complete
	hookWg.Wait()
	close(hookErrCh)

	// Execute post-shutdown hooks sequentially
	if err := s.runPostShutdownHooks(ctx); err != nil {
		s.logger.Error("Post-shutdown hook error", log.E(err))
	}

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	s.logger.Info("Server shutdown complete")
	return nil
}

// Close immediately closes all server listeners, terminating any active connections.
// Unlike Shutdown, this method does not wait for connections to finish gracefully.
// It closes both HTTP and HTTPS listeners concurrently.
//
// This method is thread-safe and can be called multiple times safely.
// Returns the last error encountered while closing listeners, or nil if successful.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Closing server listeners...")
	var lastErr error

	if s.listener != nil {
		s.logger.Debug("Closing HTTP listener")
		if err := s.listener.Close(); err != nil {
			s.logger.Error("Error closing HTTP listener", log.F("error", err))
			lastErr = err
		}
	}

	if s.tlsListener != nil {
		s.logger.Debug("Closing HTTPS listener")
		if err := s.tlsListener.Close(); err != nil {
			s.logger.Error("Error closing HTTPS listener", log.F("error", err))
			lastErr = err
		}
	}

	if s.http3Server != nil {
		s.logger.Debug("Closing HTTP/3 server")
		if err := s.http3Server.Close(); err != nil {
			s.logger.Error("Error closing HTTP/3 server", log.F("error", err))
			lastErr = err
		}
	}

	if s.webTransportServer != nil {
		s.logger.Debug("Closing WebTransport server")
		if err := s.webTransportServer.Close(); err != nil {
			s.logger.Error("Error closing WebTransport server", log.F("error", err))
			lastErr = err
		}
	}

	if s.metricsListener != nil {
		s.logger.Debug("Closing metrics listener")
		if err := s.metricsListener.Close(); err != nil {
			s.logger.Error("Error closing metrics listener", log.F("error", err))
			lastErr = err
		}
	}

	if lastErr == nil {
		s.logger.Debug("All listeners closed successfully")
	}

	return lastErr
}

// createHTTPSRedirectHandler creates an HTTP handler that redirects all requests
// to their HTTPS equivalent. This handler is used by the HTTP server when
// running in AutoTLS mode to ensure all traffic is encrypted.
//
// The redirect preserves the original request path and query parameters.
// Returns an http.Handler that performs permanent redirects (301) to HTTPS.
func (s *Server) createHTTPSRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build HTTPS URL by copying the URL and changing scheme
		target := *r.URL
		target.Scheme = "https"
		target.Host = r.Host

		httpsURL := target.String()
		s.logger.Debug("Redirecting HTTP to HTTPS",
			log.F("from", r.URL.String()),
			log.F("to", httpsURL))
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})
}

// runPreShutdownHooks executes pre-shutdown hooks sequentially in registration order.
func (s *Server) runPreShutdownHooks(ctx context.Context) error {
	s.mu.RLock()
	hooks := s.preShutdownHooks
	s.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	s.logger.Debug("Running pre-shutdown hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			s.logger.Warn("Pre-shutdown hook aborted due to context cancellation", log.F("hook", hook.Name))
			return ctx.Err()
		default:
		}

		s.logger.Debug("Running pre-shutdown hook", log.F("hook", hook.Name))
		if err := hook.Hook(ctx); err != nil {
			s.logger.Error("Pre-shutdown hook failed", log.F("hook", hook.Name), log.E(err))
			// Continue with other hooks despite error
		}
	}

	return nil
}

// startShutdownHooks starts shutdown hooks concurrently and returns a WaitGroup and error channel.
// The caller must wait on the returned WaitGroup and then close the error channel.
func (s *Server) startShutdownHooks(ctx context.Context) (*sync.WaitGroup, chan error) {
	s.mu.RLock()
	hooks := s.shutdownHooks
	s.mu.RUnlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(hooks))

	if len(hooks) == 0 {
		return &wg, errCh
	}

	s.logger.Debug("Starting shutdown hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		wg.Add(1)
		go func(h config.ShutdownHookConfig) {
			defer wg.Done()

			s.logger.Debug("Running shutdown hook", log.F("hook", h.Name))
			if err := h.Hook(ctx); err != nil {
				s.logger.Error("Shutdown hook failed", log.F("hook", h.Name), log.E(err))
				errCh <- err
			}
		}(hook)
	}

	return &wg, errCh
}

// runPostShutdownHooks executes post-shutdown hooks sequentially in registration order.
func (s *Server) runPostShutdownHooks(ctx context.Context) error {
	s.mu.RLock()
	hooks := s.postShutdownHooks
	s.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	s.logger.Debug("Running post-shutdown hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			s.logger.Warn("Post-shutdown hook aborted due to context cancellation", log.F("hook", hook.Name))
			return ctx.Err()
		default:
		}

		s.logger.Debug("Running post-shutdown hook", log.F("hook", hook.Name))
		if err := hook.Hook(ctx); err != nil {
			s.logger.Error("Post-shutdown hook failed", log.F("hook", hook.Name), log.E(err))
			// Continue with other hooks despite error
		}
	}

	return nil
}

// startMetricsServer starts the dedicated metrics server.
// It creates a listener if one doesn't exist and serves metrics.
func (s *Server) startMetricsServer() error {
	s.mu.Lock()

	var err error
	if s.metricsListener == nil {
		s.logger.Debug("Creating metrics listener", log.F("addr", s.metricsServer.Addr))
		s.metricsListener, err = net.Listen("tcp", s.metricsServer.Addr)
		if err != nil {
			s.mu.Unlock()
			return err
		}
	}

	s.mu.Unlock()

	return s.metricsServer.Serve(s.metricsListener)
}

// MetricsAddr returns the network address that the metrics server is listening on.
// If a listener is configured, it returns the listener's actual address.
// If no listener is configured but a metrics server is configured, it returns the server's configured address.
// If no metrics server is configured, it returns an empty string.
//
// This method is thread-safe and can be called concurrently.
func (s *Server) MetricsAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.metricsListener != nil {
		return s.metricsListener.Addr().String()
	}

	if s.metricsServer != nil {
		return s.metricsServer.Addr
	}

	return ""
}

func fmtHTTPAddr(addr string) string {
	return fmt.Sprintf("http://%s", addr)
}

func fmtHTTPSAddr(addr string) string {
	return fmt.Sprintf("https://%s", addr)
}
