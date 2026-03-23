// Package zerohttp provides HTTP/3 server support. See [Server.ListenAndServeHTTP3].
package zerohttp

import (
	"github.com/alexferl/zerohttp/extensions/http3"
	"github.com/alexferl/zerohttp/log"
)

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
//   - server: An HTTP/3 server instance implementing the HTTP3Server interface
func (s *Server) SetHTTP3Server(server http3.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.http3Server = server
}
