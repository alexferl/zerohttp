package http3

import (
	"context"

	"github.com/alexferl/zerohttp/extensions/autocert"
)

// Server is the interface that HTTP/3 servers must implement to be used with zerohttp.
// Users can inject their own HTTP/3 implementation (e.g., github.com/quic-go/quic-go/http3).
type Server interface {
	// ListenAndServeTLS starts the HTTP/3 server with the provided certificate and key.
	// Certificate files are in PEM format.
	ListenAndServeTLS(certFile, keyFile string) error

	// Shutdown gracefully shuts down the HTTP/3 server.
	Shutdown(ctx context.Context) error

	// Close immediately closes the HTTP/3 server.
	Close() error
}

// ServerWithAutocert is an optional interface for HTTP/3 servers that support
// automatic certificate management via autocert.Manager. If an HTTP/3 server
// implements this interface, it will be used by StartAutoTLS to configure
// HTTP/3 with Let's Encrypt certificates.
//
// quic-go's http3.Server implements this interface when configured with a TLSConfig
// containing the autocert GetCertificate function.
type ServerWithAutocert interface {
	Server

	// ListenAndServeTLSWithAutocert starts the HTTP/3 server with automatic
	// certificate management using the provided autocert manager.
	// The manager's GetCertificate function is used to obtain TLS certificates.
	ListenAndServeTLSWithAutocert(manager autocert.Manager) error
}
