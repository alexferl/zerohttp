// Package compress provides response compression middleware.
//
// Supports gzip, brotli, and zstd encoding with automatic negotiation
// based on Accept-Encoding headers.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/compress"
//
//	// Use defaults (gzip, level 6)
//	app.Use(compress.New())
//
//	// Custom configuration
//	app.Use(compress.New(compress.Config{
//	    Algorithms: []compress.Algorithm{compress.Gzip, compress.Brotli},
//	    Level:      compress.BestSpeed,
//	}))
//
// # Content Types
//
// Only compresses specific content types by default (text, json, etc).
// Customize with:
//
//	app.Use(compress.New(compress.Config{
//	    Types: []string{"text/html", "application/json", "application/xml"},
//	}))
package compress
