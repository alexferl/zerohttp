package sse

import "net/http"

// Provider creates SSE connections from HTTP requests.
// Implement this to provide custom SSE implementations.
type Provider interface {
	// New creates a new SSE connection from the request/response.
	// Returns error if headers were already sent or SSE is not supported.
	New(w http.ResponseWriter, r *http.Request) (Connection, error)
}

// Ensure DefaultProvider implements Provider
var _ Provider = (*DefaultProvider)(nil)

// DefaultProvider implements Provider using the stdlib
type DefaultProvider struct{}

// NewDefaultProvider creates a new stdlib-based SSE provider.
func NewDefaultProvider() *DefaultProvider {
	return &DefaultProvider{}
}

// New creates a new SSE connection using the stdlib implementation.
func (p *DefaultProvider) New(w http.ResponseWriter, r *http.Request) (Connection, error) {
	return New(w, r)
}
