package autocert

import (
	"crypto/tls"
	"net/http"
)

// Manager is the interface for automatic TLS certificate management.
// Users can implement this interface or use golang.org/x/crypto/acme/autocert.Manager.
type Manager interface {
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
