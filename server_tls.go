package zerohttp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

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
