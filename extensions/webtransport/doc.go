// Package webtransport provides WebTransport support for zerohttp.
//
// This package defines interfaces for WebTransport server implementations.
// Users can plug in their own WebTransport implementation
// (e.g., github.com/quic-go/webtransport-go).
//
// WebTransport is a protocol providing multiplexed, low-latency, bidirectional
// communication between clients and servers over HTTP/3.
//
// # Usage
//
// Use with zerohttp's WebTransport config option:
//
//	import (
//	    zh "github.com/alexferl/zerohttp"
//	    "github.com/alexferl/zerohttp/extensions/webtransport"
//	    "github.com/quic-go/quic-go/http3"
//	    "github.com/quic-go/webtransport-go"
//	)
//
//	app := zh.New()
//	wtServer := &webtransport.Server{
//	    H3: &http3.Server{Addr: ":443", Handler: app},
//	}
//	app.SetWebTransportServer(wtServer)
//
//	// WebTransport starts automatically with TLS
//	log.Fatal(app.StartTLS("cert.pem", "key.pem"))
//
// WebTransport servers that support autocert can implement ServerWithAutocert
// for automatic certificate management.
package webtransport
