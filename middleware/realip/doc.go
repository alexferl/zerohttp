// Package realip provides client IP extraction middleware.
//
// Extracts the real client IP from X-Forwarded-For, X-Real-Ip headers,
// or other proxy headers. Configurable trusted proxies.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/realip"
//
//	// Use defaults (trusts X-Forwarded-For, X-Real-Ip)
//	app.Use(realip.New())
//
//	// Custom trusted proxies
//	app.Use(realip.New(realip.Config{
//	    TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
//	}))
//
// # Custom Headers
//
//	app.Use(realip.New(realip.Config{
//	    Headers: []string{"X-Forwarded-For", "CF-Connecting-IP"},
//	}))
package realip
