// Package circuitbreaker provides circuit breaker middleware for fault tolerance.
//
// The circuit breaker prevents cascade failures by temporarily rejecting requests
// when a service is experiencing high error rates.
//
// # States
//
//   - Closed: Normal operation, requests pass through
//   - Open: Requests are rejected immediately (service failing)
//   - Half-Open: Testing if service has recovered
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/circuitbreaker"
//
//	app.Use(circuitbreaker.New(circuitbreaker.Config{
//	    Threshold:   5,                // Open after 5 failures
//	    ResetTimeout: 30 * time.Second, // Try again after 30s
//	}))
//
// # Per-Endpoint Circuits
//
//	app.Use(circuitbreaker.New(circuitbreaker.Config{
//	    KeyExtractor: func(r *http.Request) string {
//	        return r.URL.Path // Separate circuit per endpoint
//	    },
//	}))
package circuitbreaker
