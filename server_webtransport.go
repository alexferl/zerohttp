// Package zerohttp provides WebTransport server support. See [Server.SetWebTransportServer].
package zerohttp

import "github.com/alexferl/zerohttp/extensions/webtransport"

// SetWebTransportServer sets the WebTransport server instance. This can be used to inject
// a WebTransport implementation (e.g., quic-go/webtransport-go) after creating the server.
//
// The WebTransport server will be started automatically when ListenAndServeTLS or Start
// is called. You don't need to call ListenAndServeTLS on the WebTransport server yourself.
//
// Parameters:
//   - server: A WebTransport server instance implementing the config.WebTransportServer interface
func (s *Server) SetWebTransportServer(server webtransport.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webTransportServer = server
}
