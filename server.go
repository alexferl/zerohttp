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
	"github.com/alexferl/zerohttp/middleware"
	"golang.org/x/crypto/acme/autocert"
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

	// autocertManager handles automatic certificate provisioning and renewal
	// using Let's Encrypt ACME protocol. If set, enables automatic TLS.
	autocertManager *autocert.Manager

	// logger is the structured logger used by the server and its middleware
	// for recording HTTP requests, errors, and server lifecycle events.
	logger log.Logger

	// mu protects concurrent access to server fields during startup,
	// shutdown, and configuration operations.
	mu sync.RWMutex
}

// New creates and configures a new Server instance with the provided options.
// It initializes the server with default configurations that can be overridden
// using the provided options. The server includes HTTP and HTTPS support,
// middleware integration, and structured logging.
//
// Example usage:
//
//	server := zerohttp.New(
//	    config.WithAddr(":8080"),
//	    config.WithLogger(myLogger),
//	)
func New(opts ...config.Option) *Server {
	cfg := config.DefaultConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.Build()

	router := NewRouter()

	logger := cfg.Logger
	if logger == nil {
		logger = log.NewDefaultLogger()
	}

	router.SetLogger(logger)
	router.SetConfig(cfg)

	server := cfg.Server
	if server == nil {
		server = &http.Server{
			Addr:           cfg.Addr,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1 MB
		}
	}

	tlsServer := cfg.TLSServer
	if tlsServer == nil {
		tlsServer = &http.Server{
			Addr:           cfg.TLSAddr,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1 MB
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
	}

	s := &Server{
		Router:          router,
		server:          server,
		listener:        cfg.Listener,
		tlsServer:       tlsServer,
		tlsListener:     cfg.TLSListener,
		certFile:        cfg.CertFile,
		keyFile:         cfg.KeyFile,
		autocertManager: cfg.AutocertManager,
		logger:          logger,
	}

	if s.server != nil {
		s.server.Handler = router
	}

	if s.tlsServer != nil {
		s.tlsServer.Handler = router
	}

	var middlewares []func(http.Handler) http.Handler

	if cfg.DisableDefaultMiddlewares {
		middlewares = cfg.DefaultMiddlewares
	} else if cfg.DefaultMiddlewares == nil {
		middlewares = middleware.DefaultMiddlewares(cfg, s.logger)
	} else {
		defaults := middleware.DefaultMiddlewares(cfg, s.logger)
		middlewares = append(defaults, cfg.DefaultMiddlewares...)
	}

	if len(middlewares) > 0 {
		s.Use(middlewares...)
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

	s.logger.Info("Starting HTTP server", log.F("addr", s.listener.Addr().String()))
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
	s.logger.Debug("ListenAndServeTLS called", log.F("certFile", certFile), log.F("keyFile", keyFile))

	s.mu.Lock()

	if s.tlsServer == nil {
		s.mu.Unlock()
		s.logger.Debug("TLS server not configured, skipping")
		return nil
	}

	s.logger.Debug("TLS server is configured, proceeding")

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
		log.F("addr", s.tlsListener.Addr().String()),
		log.F("cert_file", certFile),
		log.F("key_file", keyFile))

	s.logger.Debug("About to call ServeTLS")
	return s.tlsServer.ServeTLS(s.tlsListener, certFile, keyFile)
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
	errCh := make(chan error, 2)

	handler := s.Router

	// Start HTTP server
	if s.server != nil {
		s.server.Handler = handler
		go func() {
			s.logger.Debug("Starting HTTP server...", log.F("addr", fmt.Sprintf("http://%s", s.server.Addr)))
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
			s.logger.Debug("Starting HTTPS server...", log.F("addr", fmt.Sprintf("https://%s", s.tlsServer.Addr)), log.F("cert_file", s.certFile), log.F("key_file", s.keyFile))
			if err := s.tlsServer.ListenAndServeTLS(s.certFile, s.keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("HTTPS server error: %w", err)
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
		return nil
	}

	return s.ListenAndServeTLS(certFile, keyFile)
}

// StartAutoTLS starts the server with automatic TLS certificate management using Let's Encrypt.
// It starts both HTTP (for ACME challenges) and HTTPS servers.
// The HTTP server redirects to HTTPS and handles ACME challenges.
//
// Parameters:
//   - hosts: Optional list of hostnames for certificate generation. If not provided,
//     the autocert manager's existing host policy will be used.
//
// The HTTP server handles:
//   - ACME challenge requests from Let's Encrypt
//   - Redirects all other HTTP traffic to HTTPS
//
// Returns an error if the autocert manager is not configured or if any server fails to start.
func (s *Server) StartAutoTLS(hosts ...string) error {
	if s.autocertManager == nil {
		return fmt.Errorf("autocert manager not configured")
	}

	// Configure hosts if provided
	if len(hosts) > 0 {
		s.autocertManager.HostPolicy = autocert.HostWhitelist(hosts...)
	}

	s.logger.Info("Starting server with AutoTLS...", log.F("hosts", hosts))

	errCh := make(chan error, 2)

	// Start HTTP server for ACME challenges and redirects
	if s.server != nil {
		go func() {
			// Create a new server for HTTP with autocert handler
			httpServer := &http.Server{
				Addr:    s.server.Addr,
				Handler: s.autocertManager.HTTPHandler(s.createHTTPSRedirectHandler()),
			}

			s.logger.Info("Starting HTTP server for ACME challenges and redirects",
				log.F("addr", httpServer.Addr))
			errCh <- httpServer.ListenAndServe()
		}()
	}

	// Start HTTPS server with autocert
	if s.tlsServer != nil {
		go func() {
			// Configure TLS with autocert
			if s.tlsServer.TLSConfig == nil {
				s.tlsServer.TLSConfig = &tls.Config{}
			}
			s.tlsServer.TLSConfig.GetCertificate = s.autocertManager.GetCertificate

			s.logger.Info("Starting HTTPS server with AutoTLS",
				log.F("addr", s.tlsServer.Addr))
			errCh <- s.tlsServer.ListenAndServeTLS("", "")
		}()
	}

	return <-errCh
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
// Returns the first error encountered during shutdown, or nil if successful.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	if s.server != nil && s.listener != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Debug("Shutting down HTTP server")
			if err := s.server.Shutdown(ctx); err != nil {
				s.logger.Error("Error shutting down HTTP server", log.F("error", err))
				errCh <- err
			} else {
				s.logger.Debug("HTTP server shutdown complete")
			}
		}()
	}

	if s.tlsServer != nil && s.tlsListener != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Debug("Shutting down HTTPS server")
			if err := s.tlsServer.Shutdown(ctx); err != nil {
				s.logger.Error("Error shutting down HTTPS server", log.F("error", err))
				errCh <- err
			} else {
				s.logger.Debug("HTTPS server shutdown complete")
			}
		}()
	}

	wg.Wait()
	close(errCh)

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

	if lastErr == nil {
		s.logger.Debug("All listeners closed successfully")
	}

	return lastErr
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

// NewAutocertManager creates a new autocert manager with the given cache directory and hosts.
// The manager handles automatic certificate provisioning and renewal using Let's Encrypt's ACME protocol.
//
// Parameters:
//   - cacheDir: Directory path where certificates and ACME account information will be cached.
//     This directory should be persistent across server restarts.
//   - hosts: List of hostnames for which certificates should be automatically obtained.
//     Only requests for these hosts will be served with auto-generated certificates.
//
// The returned manager is configured to:
//   - Accept Let's Encrypt Terms of Service automatically
//   - Use directory-based caching for certificates
//   - Restrict certificate generation to the specified hosts
//
// Example usage:
//
//	manager := NewAutocertManager("/var/cache/certs", "example.com", "www.example.com")
//	server := zerohttp.New(config.WithAutocertManager(manager))
func NewAutocertManager(cacheDir string, hosts ...string) *autocert.Manager {
	manager := &autocert.Manager{
		Cache:      autocert.DirCache(cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hosts...),
	}
	return manager
}
