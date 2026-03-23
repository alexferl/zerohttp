// Package timeout provides request timeout middleware.
//
// Cancels requests that exceed a specified duration, returning
// HTTP 504 Gateway Timeout.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/timeout"
//
//	// 30 second timeout (default)
//	app.Use(timeout.New())
//
//	// Custom timeout
//	app.Use(timeout.New(timeout.Config{
//	    Duration: 10 * time.Second,
//	}))
//
// # Per-Route Timeouts
//
//	// Different timeouts for different routes
//	api := app.Group(func(api zh.Router) {
//	    api.Use(timeout.New(timeout.Config{Duration: 5 * time.Second}))
//	    api.GET("/fast", fastHandler)
//	})
//
//	upload := app.Group(func(upload zh.Router) {
//	    upload.Use(timeout.New(timeout.Config{Duration: 5 * time.Minute}))
//	    upload.POST("/files", uploadHandler)
//	})
package timeout
