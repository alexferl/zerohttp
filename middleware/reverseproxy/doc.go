// Package reverseproxy provides reverse proxy middleware.
//
// Proxies requests to upstream servers with load balancing support
// and health checking.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/reverseproxy"
//
//	// Single upstream
//	app.Use(reverseproxy.New(reverseproxy.Config{
//	    Targets: []string{"http://localhost:8081"},
//	}))
//
//	// Load balancing
//	app.Use(reverseproxy.New(reverseproxy.Config{
//	    Targets:       []string{"http://app1:8080", "http://app2:8080"},
//	    Strategy:      reverseproxy.RoundRobin,
//	    HealthCheck:   "/health",
//	    HealthInterval: 10 * time.Second,
//	}))
//
// # Strategies
//
//   - RoundRobin: Distribute evenly across targets
//   - Random: Random target selection
//   - LeastConn: Target with fewest active connections
//   - IPHash: Consistent hashing by client IP
package reverseproxy
