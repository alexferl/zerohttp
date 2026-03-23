// Package jwtauth provides JWT authentication middleware.
//
// The middleware provides pluggable JWT authentication. Users bring their own
// JWT library by implementing the Store interface.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/jwtauth"
//
//	app.Use(jwtauth.New(jwtauth.Config{
//	    TokenStore: myTokenStore,
//	    RequiredClaims: []string{"sub"},
//	}))
//
// # Built-in HS256
//
// For a zero-dependency option, use the built-in HS256 implementation:
//
//	app.Use(jwtauth.New(jwtauth.Config{
//	    TokenStore: jwtauth.NewHS256TokenStore(secret, opts),
//	}))
//
// # Accessing Claims
//
// Retrieve validated claims in handlers:
//
//	claims := jwtauth.GetClaims(r)
//	sub := claims.Subject()
//
// Security Note: The built-in HS256 uses HMAC-SHA256 symmetric signing.
// For asymmetric keys (RS256, ES256, EdDSA), use golang-jwt/jwt or lestrrat-go/jwx.
package jwtauth
