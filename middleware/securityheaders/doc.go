// Package securityheaders provides security header middleware.
//
// Sets recommended security headers by default including:
//   - Content-Security-Policy
//   - X-Content-Type-Options
//   - X-Frame-Options
//   - Strict-Transport-Security (HSTS)
//   - Referrer-Policy
//   - Permissions-Policy
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/securityheaders"
//
//	// Use defaults (recommended security headers)
//	app.Use(securityheaders.New())
//
//	// Custom CSP
//	app.Use(securityheaders.New(securityheaders.Config{
//	    ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'",
//	    StrictTransportSecurity: securityheaders.StrictTransportSecurity{
//	        MaxAge: 31536000,
//	    },
//	}))
//
// # CSP Nonces
//
// Enable nonce generation for inline scripts:
//
//	app.Use(securityheaders.New(securityheaders.Config{
//	    ContentSecurityPolicy:             "script-src 'nonce-{{nonce}}'",
//	    ContentSecurityPolicyNonceEnabled: true,
//	}))
//
//	// Access nonce in handler:
//	nonce := r.Context().Value(securityheaders.CSPNonceContextKey)
package securityheaders
