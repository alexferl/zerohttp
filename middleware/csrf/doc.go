// Package csrf provides Cross-Site Request Forgery protection middleware.
//
// CSRF middleware generates and validates tokens to prevent malicious
// websites from making requests on behalf of authenticated users.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/csrf"
//
//	// Use defaults
//	app.Use(csrf.New())
//
//	// Custom configuration
//	app.Use(csrf.New(csrf.Config{
//	    CookieName: "csrf",
//	    MaxAge:     24 * time.Hour,
//	    HMACKey:    []byte("your-secret-32-byte-key-here!!"),
//	}))
//
// # Token Lookup
//
// Tokens are extracted via TokenLookup (default: "header:X-Csrf-Token"):
//
//	// Extract from header
//	app.Use(csrf.New(csrf.Config{
//	    TokenLookup: "header:X-CSRF-Token",
//	}))
//
//	// Extract from form field
//	app.Use(csrf.New(csrf.Config{
//	    TokenLookup: "form:_csrf",
//	}))
//
//	// Extract from query parameter
//	app.Use(csrf.New(csrf.Config{
//	    TokenLookup: "query:csrf_token",
//	}))
//
// HMACKey is required and must be 32 bytes. Set it via CSRFConfig.
package csrf
