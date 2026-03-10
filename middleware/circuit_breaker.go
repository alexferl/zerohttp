package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/metrics"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// circuit represents a single circuit breaker instance
type circuit struct {
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	mu              sync.RWMutex
	config          config.CircuitBreakerConfig
}

// circuitBreakerMiddleware manages multiple circuit breakers
type circuitBreakerMiddleware struct {
	circuits map[string]*circuit
	config   config.CircuitBreakerConfig
	mu       sync.RWMutex
}

// CircuitBreaker creates a circuit breaker middleware
func CircuitBreaker(cfg ...config.CircuitBreakerConfig) func(http.Handler) http.Handler {
	c := config.DefaultCircuitBreakerConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.FailureThreshold <= 0 {
		c.FailureThreshold = config.DefaultCircuitBreakerConfig.FailureThreshold
	}
	if c.RecoveryTimeout <= 0 {
		c.RecoveryTimeout = config.DefaultCircuitBreakerConfig.RecoveryTimeout
	}
	if c.SuccessThreshold <= 0 {
		c.SuccessThreshold = config.DefaultCircuitBreakerConfig.SuccessThreshold
	}
	if c.IsFailure == nil {
		c.IsFailure = config.DefaultCircuitBreakerConfig.IsFailure
	}
	if c.KeyExtractor == nil {
		c.KeyExtractor = config.DefaultCircuitBreakerConfig.KeyExtractor
	}
	if c.OpenStatusCode == 0 {
		c.OpenStatusCode = config.DefaultCircuitBreakerConfig.OpenStatusCode
	}
	if c.OpenMessage == "" {
		c.OpenMessage = config.DefaultCircuitBreakerConfig.OpenMessage
	}

	cbm := &circuitBreakerMiddleware{
		circuits: make(map[string]*circuit),
		config:   c,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			key := c.KeyExtractor(r)
			circ := cbm.getCircuit(key)

			reg.Gauge("circuit_breaker_state", "key").WithLabelValues(key).Set(float64(circ.getState()))

			if circ.isOpen() {
				reg.Counter("circuit_breaker_requests_total", "key", "result").WithLabelValues(key, "rejected").Inc()
				detail := problem.NewDetail(c.OpenStatusCode, c.OpenMessage)
				_ = detail.Render(w) // Best effort - client may have disconnected
				return
			}

			wrapped := rwutil.NewResponseWriter(w)

			next.ServeHTTP(wrapped, r)

			circ.recordResult(r, wrapped.StatusCode(), reg, key)
		})
	}
}

// getCircuit gets or creates a circuit breaker for the given key
func (cbm *circuitBreakerMiddleware) getCircuit(key string) *circuit {
	cbm.mu.RLock()
	c, exists := cbm.circuits[key]
	cbm.mu.RUnlock()

	if !exists {
		cbm.mu.Lock()
		// Double-check after acquiring write lock
		c, exists = cbm.circuits[key]
		if !exists {
			c = &circuit{
				state:  StateClosed,
				config: cbm.config,
			}
			cbm.circuits[key] = c
		}
		cbm.mu.Unlock()
	}

	return c
}

// getState returns the current state of the circuit
func (c *circuit) getState() CircuitState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// isOpen checks if the circuit is open or should transition to half-open
func (c *circuit) isOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state {
	case StateClosed:
		return false
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(c.lastFailureTime) >= c.config.RecoveryTimeout {
			c.state = StateHalfOpen
			c.successCount = 0
			return false
		}
		return true
	case StateHalfOpen:
		return false
	}

	return false
}

// recordResult records the result of a request and updates circuit state
func (c *circuit) recordResult(r *http.Request, statusCode int, reg metrics.Registry, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	isFailure := c.config.IsFailure(r, statusCode)

	switch c.state {
	case StateClosed:
		if isFailure {
			c.failureCount++
			reg.Counter("circuit_breaker_failures_total", "key").WithLabelValues(key).Inc()
			if c.failureCount >= c.config.FailureThreshold {
				c.state = StateOpen
				c.lastFailureTime = time.Now()
				reg.Counter("circuit_breaker_trips_total", "key").WithLabelValues(key).Inc()
			}
		} else {
			c.failureCount = 0
		}
		reg.Counter("circuit_breaker_requests_total", "key", "result").WithLabelValues(key, "allowed").Inc()

	case StateOpen:
		// In open state, requests are blocked

	case StateHalfOpen:
		if isFailure {
			c.state = StateOpen
			c.lastFailureTime = time.Now()
			c.failureCount++
			reg.Counter("circuit_breaker_failures_total", "key").WithLabelValues(key).Inc()
			reg.Counter("circuit_breaker_trips_total", "key").WithLabelValues(key).Inc()
			reg.Counter("circuit_breaker_requests_total", "key", "result").WithLabelValues(key, "rejected").Inc()
		} else {
			c.successCount++
			if c.successCount >= c.config.SuccessThreshold {
				c.state = StateClosed
				c.failureCount = 0
				c.successCount = 0
			}
			reg.Counter("circuit_breaker_requests_total", "key", "result").WithLabelValues(key, "allowed").Inc()
		}
	}
}

// GetState returns the current state of a circuit (for monitoring)
func (cbm *circuitBreakerMiddleware) GetState(key string) CircuitState {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	if c, exists := cbm.circuits[key]; exists {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.state
	}
	return StateClosed
}

// Reset manually resets a circuit breaker (for admin operations)
func (cbm *circuitBreakerMiddleware) Reset(key string) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	if c, exists := cbm.circuits[key]; exists {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.state = StateClosed
		c.failureCount = 0
		c.successCount = 0
	}
}
