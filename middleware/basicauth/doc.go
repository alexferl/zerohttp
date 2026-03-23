// Package basicauth provides HTTP Basic Authentication middleware.
//
// Basic auth verifies credentials against either a static credentials map
// or a custom validator function.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/basicauth"
//
//	// With static credentials
//	app.Use(basicauth.New(basicauth.Config{
//	    Realm: "Protected Area",
//	    Credentials: map[string]string{
//	        "admin": "secretpassword",
//	    },
//	}))
//
//	// With custom validator
//	app.Use(basicauth.New(basicauth.Config{
//	    Validator: func(user, pass string) bool {
//	        return validateAgainstDB(user, pass)
//	    },
//	}))
//
//	// Apply to specific routes only
//	app.Use(basicauth.New(basicauth.Config{
//	    Credentials: map[string]string{"admin": "secret"},
//	    IncludedPaths: []string{"/admin/*"},
//	}))
package basicauth
