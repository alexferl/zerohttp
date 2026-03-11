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

// Logger returns the structured logger instance used by the server.
// This logger is used for recording HTTP requests, errors, server lifecycle events,
// and can be used by application code for consistent logging.
//
// The returned logger implements the log.Logger interface and provides
// structured logging capabilities with fields and different log levels.
func (s *Server) Logger() log.Logger {
	return s.logger
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

func fmtHTTPAddr(addr string) string {
	return fmt.Sprintf("http://%s", addr)
}

func fmtHTTPSAddr(addr string) string {
	return fmt.Sprintf("https://%s", addr)
}
