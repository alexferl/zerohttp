package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
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

// responseWriter wraps http.ResponseWriter to capture status code
type circuitResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *circuitResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *circuitResponseWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

// CircuitBreaker creates a circuit breaker middleware
func CircuitBreaker(opts ...config.CircuitBreakerOption) func(http.Handler) http.Handler {
	cfg := config.DefaultCircuitBreakerConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = config.DefaultCircuitBreakerConfig.FailureThreshold
	}
	if cfg.RecoveryTimeout <= 0 {
		cfg.RecoveryTimeout = config.DefaultCircuitBreakerConfig.RecoveryTimeout
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = config.DefaultCircuitBreakerConfig.SuccessThreshold
	}
	if cfg.IsFailure == nil {
		cfg.IsFailure = config.DefaultCircuitBreakerConfig.IsFailure
	}
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = config.DefaultCircuitBreakerConfig.KeyExtractor
	}
	if cfg.OpenStatusCode == 0 {
		cfg.OpenStatusCode = config.DefaultCircuitBreakerConfig.OpenStatusCode
	}
	if cfg.OpenMessage == "" {
		cfg.OpenMessage = config.DefaultCircuitBreakerConfig.OpenMessage
	}

	cbm := &circuitBreakerMiddleware{
		circuits: make(map[string]*circuit),
		config:   cfg,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := cfg.KeyExtractor(r)
			c := cbm.getCircuit(key)

			if c.isOpen() {
				w.WriteHeader(cfg.OpenStatusCode)
				if _, err := w.Write([]byte(cfg.OpenMessage)); err != nil {
					panic(fmt.Errorf("circuit breaker message write failed: %w", err))
				}
				return
			}

			wrapped := &circuitResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			c.recordResult(r, wrapped.statusCode)
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
func (c *circuit) recordResult(r *http.Request, statusCode int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	isFailure := c.config.IsFailure(r, statusCode)

	switch c.state {
	case StateClosed:
		if isFailure {
			c.failureCount++
			if c.failureCount >= c.config.FailureThreshold {
				c.state = StateOpen
				c.lastFailureTime = time.Now()
			}
		} else {
			c.failureCount = 0 // Reset on success
		}

	case StateOpen:
		// In open state, we don't normally record results since requests are blocked
		// But if somehow a request gets through (shouldn't happen), just ignore it
		// The transition to half-open happens in isOpen() based on time

	case StateHalfOpen:
		if isFailure {
			c.state = StateOpen
			c.lastFailureTime = time.Now()
			c.failureCount++
		} else {
			c.successCount++
			if c.successCount >= c.config.SuccessThreshold {
				c.state = StateClosed
				c.failureCount = 0
				c.successCount = 0
			}
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
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	if c, exists := cbm.circuits[key]; exists {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.state = StateClosed
		c.failureCount = 0
		c.successCount = 0
	}
}
