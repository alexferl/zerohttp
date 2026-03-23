package webtransport

import "github.com/alexferl/zerohttp/extensions/autocert"

// Server is the interface that WebTransport servers must implement
// to be used with zerohttp. Users can inject their own WebTransport implementation
// (e.g., github.com/quic-go/webtransport-go).
type Server interface {
	// ListenAndServeTLS starts the WebTransport server with the provided certificate and key.
	// Certificate files are in PEM format.
	ListenAndServeTLS(certFile, keyFile string) error

	// Close immediately closes the WebTransport server.
	Close() error
}

// ServerWithAutocert is an optional interface for WebTransport servers that support
// automatic certificate management via autocert.Manager. If a WebTransport server
// implements this interface, it will be used by StartAutoTLS to configure
// WebTransport with Let's Encrypt certificates.
type ServerWithAutocert interface {
	Server

	// ListenAndServeTLSWithAutocert starts the WebTransport server with automatic
	// certificate management using the provided autocert manager.
	// The manager's GetCertificate function is used to obtain TLS certificates.
	ListenAndServeTLSWithAutocert(manager autocert.Manager) error
}
