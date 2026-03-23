// Package http3 provides HTTP/3 support for zerohttp.
//
// This package defines interfaces for HTTP/3 server implementations.
// Users can plug in their own HTTP/3 implementation (e.g., github.com/quic-go/quic-go/http3).
//
// # Usage
//
// Use with zerohttp's HTTP3Server config option:
//
//	import (
//	    zh "github.com/alexferl/zerohttp"
//	    "github.com/alexferl/zerohttp/extensions/http3"
//	    quichttp3 "github.com/quic-go/quic-go/http3"
//	)
//
//	app := zh.New()
//	h3Server := &quichttp3.Server{
//	    Addr:    ":443",
//	    Handler: app,
//	}
//	app.SetHTTP3Server(h3Server)
//
//	// Start with TLS - HTTP/3 starts automatically alongside HTTPS
//	log.Fatal(app.StartTLS("cert.pem", "key.pem"))
//
// HTTP/3 servers that support autocert can implement ServerWithAutocert
// for automatic certificate management.
package http3
