// Package requestbodysize provides request body size limiting middleware.
//
// Prevents denial of service attacks by limiting the maximum request body size.
// Returns 413 Payload Too Large if the limit is exceeded.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/requestbodysize"
//
//	// Limit to 1MB (default)
//	app.Use(requestbodysize.New())
//
//	// Custom limit
//	app.Use(requestbodysize.New(requestbodysize.Config{
//	    MaxBytes: 5 * 1024 * 1024, // 5MB
//	}))
//
// # Skip Specific Paths
//
//	app.Use(requestbodysize.New(requestbodysize.Config{
//	    MaxBytes: 1 << 20,
//	    ExcludedPaths: []string{"/upload/*"},
//	}))
package requestbodysize
