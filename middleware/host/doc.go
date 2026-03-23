// Package host provides host header validation middleware.
//
// Ensures requests have an acceptable Host header to prevent
// DNS rebinding and virtual host confusion attacks.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/host"
//
//	// Allow specific hosts
//	app.Use(host.New(host.Config{
//	    AllowedHosts: []string{"example.com", "www.example.com"},
//	}))
//
//	// Allow any host (not recommended for production)
//	app.Use(host.New(host.Config{
//	    AllowedHosts: []string{"*"},
//	}))
package host
