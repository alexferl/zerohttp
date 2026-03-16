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
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/middleware"
)

// Default server timeout constants
const (
	DefaultReadTimeout       = 10 * time.Second
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultWriteTimeout      = 15 * time.Second
	DefaultIdleTimeout       = 60 * time.Second
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

	// preStartupHooks execute sequentially before any startup hooks.
	preStartupHooks []config.StartupHookConfig

	// startupHooks execute sequentially before the server starts accepting connections.
	// If any startup hook returns an error, the server will not start.
	startupHooks []config.StartupHookConfig

	// postStartupHooks execute sequentially after the server has started.
	postStartupHooks []config.StartupHookConfig

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
// It initializes the server with sensible defaults that can be overridden
// using the provided config.
//
// The server includes:
//   - HTTP and HTTPS support
//   - Middleware integration
//   - Structured logging
//   - Automatic metrics collection (enabled by default)
//   - Request binding and validation
//
// Example - Basic usage with defaults:
//
//	app := zh.New()
//	app.GET("/", handler)
//	log.Fatal(app.Start())
//
// Example - With custom configuration:
//
//	app := zh.New(config.Config{
//	    Addr:         ":8080",
//	    ReadTimeout:  10 * time.Second,
//	    WriteTimeout: 15 * time.Second,
//	    Logger:       myLogger,
//	    Metrics: config.MetricsConfig{
//	        Enabled: false, // Disable metrics
//	    },
//	})
//
// Example - With pluggable validator:
//
//	app := zh.New(config.Config{
//	    Validator: myCustomValidator,
//	})
func New(cfg ...config.Config) *Server {
	c := mergeConfig(cfg...)
	router := NewRouter()
	logger := createLogger(c)

	router.SetLogger(logger)
	router.SetConfig(c)

	server := createHTTPServer(c)
	tlsServer := createTLSServer(c)
	registry := createMetricsRegistry(c)
	metricsServer := createMetricsServer(c, registry)

	baseCtx, cancelBaseCtx := context.WithCancel(context.Background())

	s := &Server{
		Router:             router,
		server:             server,
		listener:           c.Listener,
		tlsServer:          tlsServer,
		tlsListener:        c.TLS.Listener,
		certFile:           c.TLS.CertFile,
		keyFile:            c.TLS.KeyFile,
		autocertManager:    c.Extensions.AutocertManager,
		http3Server:        c.Extensions.HTTP3Server,
		webTransportServer: c.Extensions.WebTransportServer,
		webSocketUpgrader:  c.Extensions.WebSocketUpgrader,
		sseProvider:        c.Extensions.SSEProvider,
		metricsRegistry:    registry,
		metricsServer:      metricsServer,
		metricsServerAddr:  c.Metrics.ServerAddr,
		validator:          c.Validator,
		logger:             logger,
		preStartupHooks:    c.Lifecycle.PreStartupHooks,
		startupHooks:       c.Lifecycle.StartupHooks,
		postStartupHooks:   c.Lifecycle.PostStartupHooks,
		preShutdownHooks:   c.Lifecycle.PreShutdownHooks,
		shutdownHooks:      c.Lifecycle.ShutdownHooks,
		postShutdownHooks:  c.Lifecycle.PostShutdownHooks,
		baseCtx:            baseCtx,
		cancelBaseCtx:      cancelBaseCtx,
	}

	setupMiddleware(s, c, registry)
	setupServerHandlers(s, router)
	registerMetricsEndpoint(s, c, registry)

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

// Start begins serving HTTP, HTTPS, and metrics traffic concurrently.
// It starts all configured servers (HTTP, HTTPS, metrics, HTTP/3, WebTransport)
// in separate goroutines and blocks until all servers exit.
//
// For HTTPS, the server will start if:
//   - TLS server is configured AND
//   - Either certificates are loaded in TLS config OR certificate files are specified
//
// Start blocks until all servers exit. If any server encounters an unexpected
// error (i.e. not ErrServerClosed), that error is returned immediately.
// Returns nil when all servers shut down cleanly (e.g. via Shutdown()).
func (s *Server) Start() error {
	s.logger.Info("Starting server...")

	// Run pre-startup hooks first
	if err := s.runPreStartupHooks(s.baseCtx); err != nil {
		s.logger.Error("Pre-startup hook failed, server not starting", log.E(err))
		return err
	}

	handler := s.Router

	// Determine if we should start HTTPS server
	shouldStartTLS := s.tlsServer != nil &&
		((s.tlsServer.TLSConfig != nil &&
			(len(s.tlsServer.TLSConfig.Certificates) > 0 || s.tlsServer.TLSConfig.GetCertificate != nil)) ||
			(s.certFile != "" && s.keyFile != ""))

	// Validate and load certificates before starting any goroutines.
	// This avoids orphaning the HTTP/metrics server goroutines on cert failure.
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
				return fmt.Errorf("failed to load certificate files: %w", err)
			}
			s.tlsServer.TLSConfig.Certificates = []tls.Certificate{cert}
		}
	}

	var wg sync.WaitGroup

	// Calculate actual number of servers that will start for error channel capacity
	serverCount := 0
	if s.metricsServer != nil {
		serverCount++
	}
	if s.server != nil {
		serverCount++
	}
	if shouldStartTLS {
		serverCount++ // HTTPS server
	}
	if s.http3Server != nil && shouldStartTLS {
		serverCount++
	}
	if s.webTransportServer != nil && shouldStartTLS {
		serverCount++
	}

	errCh := make(chan error, serverCount)

	// Start metrics server if configured
	if s.metricsServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Starting metrics server...", log.F("addr", fmtHTTPAddr(s.metricsServer.Addr)))
			if err := s.startMetricsServer(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("metrics server error: %w", err)
			}
		}()
	}

	// Start HTTP server
	if s.server != nil {
		s.server.Handler = handler
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Starting HTTP server...", log.F("addr", fmtHTTPAddr(s.server.Addr)))
			if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()
	}

	// Start HTTPS server
	if shouldStartTLS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("Starting HTTPS server...",
				log.F("addr", fmtHTTPSAddr(s.tlsServer.Addr)),
				log.F("cert_file", s.certFile),
				log.F("key_file", s.keyFile))
			// Pass empty strings - certs are already loaded in TLSConfig.Certificates
			if err := s.tlsServer.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
			// Pass empty strings - certs are already loaded in TLSConfig.Certificates
			if err := s.http3Server.ListenAndServeTLS("", ""); err != nil {
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
			// Pass empty strings - certs are already loaded in TLSConfig.Certificates
			if err := s.webTransportServer.ListenAndServeTLS("", ""); err != nil {
				s.logger.Error("WebTransport server error", log.E(err))
			}
		}()
	}

	// Guard against hanging if no servers were started
	started := 0
	if s.metricsServer != nil {
		started++
	}
	if s.server != nil {
		started++
	}
	if shouldStartTLS {
		started++
	}
	if started == 0 {
		s.logger.Warn("No servers configured, Start() returning immediately")
		return nil
	}

	// Run startup hooks concurrently with servers
	startupHookErrCh := make(chan error, 1)
	go func() {
		if err := s.runStartupHooks(s.baseCtx); err != nil {
			s.logger.Error("Startup hook failed, initiating shutdown", log.E(err))
			startupHookErrCh <- err
			// Trigger shutdown to stop the servers
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.Shutdown(shutdownCtx)
			return
		}
		close(startupHookErrCh)

		// Run post-startup hooks after startup hooks complete successfully
		if err := s.runPostStartupHooks(s.baseCtx); err != nil {
			s.logger.Error("Post-startup hook failed", log.E(err))
			// Non-fatal - continue running
		}
	}()

	// Close errCh when all goroutines complete, then range to collect any errors
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Check for server errors or startup hook errors
	for {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
			// errCh closed without error, check startup hook
			if hookErr := <-startupHookErrCh; hookErr != nil {
				return hookErr
			}
			return nil
		case hookErr := <-startupHookErrCh:
			// Startup hook failed, wait for servers to shut down
			if hookErr != nil {
				// Drain errCh
				go func() {
					for range errCh {
					}
				}()
				return hookErr
			}
		}
	}
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
	errCh := make(chan error, 5) // 5 potential goroutines: server, tlsServer, webTransport, http3, metrics

	if s.server != nil {
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

	if s.tlsServer != nil {
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

	// Shutdown WebTransport and HTTP/3 concurrently
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
	if s.metricsServer != nil {
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

	// Collect errors from servers and return the first one
	var firstErr error
	for err := range errCh {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Wait for shutdown hooks to complete
	hookWg.Wait()
	close(hookErrCh)

	// Drain hook errors (log them but don't fail shutdown)
	for err := range hookErrCh {
		if err != nil {
			s.logger.Error("Shutdown hook error", log.E(err))
		}
	}

	// Execute post-shutdown hooks sequentially
	if err := s.runPostShutdownHooks(ctx); err != nil {
		s.logger.Error("Post-shutdown hook error", log.E(err))
	}

	s.logger.Info("Server shutdown complete")
	return firstErr
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

	// Close HTTP server directly - works for both ListenAndServe() and Start()
	if s.server != nil {
		s.logger.Debug("Closing HTTP server")
		if err := s.server.Close(); err != nil {
			s.logger.Error("Error closing HTTP server", log.F("error", err))
			lastErr = err
		}
	}

	// Close HTTPS server directly - works for both ListenAndServe() and Start()
	if s.tlsServer != nil {
		s.logger.Debug("Closing HTTPS server")
		if err := s.tlsServer.Close(); err != nil {
			s.logger.Error("Error closing HTTPS server", log.F("error", err))
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

	// Close metrics server directly - works for both ListenAndServe() and Start()
	if s.metricsServer != nil {
		s.logger.Debug("Closing metrics server")
		if err := s.metricsServer.Close(); err != nil {
			s.logger.Error("Error closing metrics server", log.F("error", err))
			lastErr = err
		}
	}

	if lastErr == nil {
		s.logger.Debug("All listeners closed successfully")
	}

	return lastErr
}

// mergeConfig merges user config with defaults.
func mergeConfig(cfg ...config.Config) config.Config {
	c := config.DefaultConfig
	if len(cfg) > 0 {
		userCfg := cfg[0]
		zconfig.Merge(&c, userCfg)
		// Handle fields that must always be copied (even if zero value)
		// ServerAddr can be set to empty string to disable separate metrics server
		c.Metrics.ServerAddr = userCfg.Metrics.ServerAddr
	}
	return c
}

// createLogger creates a logger instance from config or returns default.
func createLogger(c config.Config) log.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return log.NewDefaultLogger()
}

// createHTTPServer creates the HTTP server from config.
func createHTTPServer(c config.Config) *http.Server {
	if c.Server != nil {
		return c.Server
	}
	return &http.Server{
		Addr:              c.Addr,
		ReadTimeout:       DefaultReadTimeout,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}
}

// createTLSServer creates the TLS server from config if TLS is configured.
func createTLSServer(c config.Config) *http.Server {
	if c.TLS.Server != nil {
		return c.TLS.Server
	}
	if !needsTLSServer(c) {
		return nil
	}
	return &http.Server{
		Addr:              c.TLS.Addr,
		ReadTimeout:       DefaultReadTimeout,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		MaxHeaderBytes:    1 << 20, // 1 MB
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"},
		},
	}
}

// needsTLSServer returns true if the config requires a TLS server to be created.
func needsTLSServer(c config.Config) bool {
	return c.TLS.CertFile != "" ||
		c.TLS.KeyFile != "" ||
		c.Extensions.AutocertManager != nil ||
		c.TLS.Listener != nil ||
		c.Extensions.HTTP3Server != nil
}

// createMetricsRegistry creates metrics registry if enabled.
func createMetricsRegistry(c config.Config) metrics.Registry {
	if c.Metrics.Enabled {
		return metrics.NewRegistry()
	}
	return nil
}

// createMetricsServer creates a separate metrics server if ServerAddr is set.
func createMetricsServer(c config.Config, registry metrics.Registry) *http.Server {
	if !c.Metrics.Enabled || registry == nil || c.Metrics.ServerAddr == "" {
		return nil
	}
	return &http.Server{
		Addr:              c.Metrics.ServerAddr,
		ReadTimeout:       DefaultReadTimeout,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		Handler:           metrics.Handler(registry),
	}
}

// setupMiddleware configures the middleware chain on the server.
func setupMiddleware(s *Server, c config.Config, registry metrics.Registry) {
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
}

// setupServerHandlers sets the router and base context on server instances.
func setupServerHandlers(s *Server, router Router) {
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
}

// registerMetricsEndpoint registers the metrics endpoint on the main router if needed.
func registerMetricsEndpoint(s *Server, c config.Config, registry metrics.Registry) {
	if c.Metrics.Enabled && registry != nil && c.Metrics.ServerAddr == "" {
		s.logger.Warn("Metrics endpoint registered on main server (set Metrics.ServerAddr to isolate)", log.F("endpoint", c.Metrics.Endpoint))
		s.GET(c.Metrics.Endpoint, metrics.Handler(registry))
	}
}

func fmtHTTPAddr(addr string) string {
	return fmt.Sprintf("http://%s", addr)
}

func fmtHTTPSAddr(addr string) string {
	return fmt.Sprintf("https://%s", addr)
}
