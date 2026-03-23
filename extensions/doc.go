// Package extensions provides optional interfaces for extending zerohttp functionality.
//
// These extensions allow you to plug in third-party implementations for features
// that require external dependencies, maintaining zerohttp's core zero-dependency
// philosophy while enabling advanced use cases.
//
// # Available Extensions
//
//   - [github.com/alexferl/zerohttp/extensions/autocert] - Automatic TLS certificate
//     management (Let's Encrypt) via golang.org/x/crypto/acme/autocert
//
//   - [github.com/alexferl/zerohttp/extensions/http3] - HTTP/3 support via
//     github.com/quic-go/quic-go/http3
//
//   - [github.com/alexferl/zerohttp/extensions/webtransport] - WebTransport support
//     via github.com/quic-go/webtransport-go
//
//   - [github.com/alexferl/zerohttp/extensions/websocket] - WebSocket support
//     via github.com/gorilla/websocket
//
// # Usage
//
// Extensions are configured via Config options:
//
//	import (
//	    zh "github.com/alexferl/zerohttp"
//	    "github.com/alexferl/zerohttp/extensions/autocert"
//	    "github.com/alexferl/zerohttp/extensions/http3"
//	)
//
//	// HTTP/3 with autocert
//	mgr := &autocert.Manager{...}
//	app := zh.New(zh.Config{
//	    Extensions: zh.ExtensionsConfig{
//	        AutocertManager: mgr,
//	        HTTP3Server:     &http3.Server{...},
//	    },
//	})
//
// Each extension package defines interfaces that third-party libraries can
// implement to integrate with zerohttp.
package extensions
