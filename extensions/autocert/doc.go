// Package autocert provides automatic TLS certificate management via Let's Encrypt.
//
// This package defines the Manager interface that wraps golang.org/x/crypto/acme/autocert.Manager
// or any custom implementation for automatic certificate provisioning and renewal.
//
// # Usage
//
// Use with zerohttp's AutocertManager config option:
//
//	import (
//	    "golang.org/x/crypto/acme/autocert"
//	    zh "github.com/alexferl/zerohttp"
//	)
//
//	mgr := &autocert.Manager{
//	    Cache:      autocert.DirCache("/var/cache/certs"),
//	    Prompt:     autocert.AcceptTOS,
//	    HostPolicy: autocert.HostWhitelist("example.com"),
//	}
//
//	app := zh.New(zh.Config{
//	    AutocertManager: mgr,
//	})
//
//	log.Fatal(app.StartAutoTLS())
//
// The Manager interface is compatible with golang.org/x/crypto/acme/autocert.Manager,
// so you can use that implementation directly or provide your own.
package autocert
