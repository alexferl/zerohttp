// Package cors provides Cross-Origin Resource Sharing middleware.
//
// Handles preflight requests and sets appropriate CORS headers
// for cross-origin browser requests.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/cors"
//
//	// Allow all origins (development only)
//	app.Use(cors.New(cors.DefaultConfig))
//
//	// Custom configuration
//	app.Use(cors.New(cors.Config{
//	    AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
//	    AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut},
//	    AllowedHeaders: []string{"Authorization", "Content-Type"},
//	    AllowCredentials: true,
//	    MaxAge: 3600,
//	}))
//
//	// Dynamic origin validation
//	app.Use(cors.New(cors.Config{
//	    AllowOriginFunc: func(origin string) bool {
//	        return strings.HasSuffix(origin, ".example.com")
//	    },
//	}))
package cors
