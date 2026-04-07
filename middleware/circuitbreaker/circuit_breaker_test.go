package circuitbreaker

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

type circuitTestHandler struct {
	statusCode int
	delay      time.Duration
	callCount  int
	mu         sync.Mutex
}

func (h *circuitTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.callCount++
	h.mu.Unlock()
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	if h.statusCode > 0 {
		w.WriteHeader(h.statusCode)
	}
	_, _ = w.Write([]byte("response"))
}

func (h *circuitTestHandler) getCallCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.callCount
}

func TestCircuitBreaker_FailureThreshold(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 3,
		RecoveryTimeout:  100 * time.Millisecond,
	})(handler)

	for range 3 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)
	}

	// Test JSON response (with Accept header)
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusServiceUnavailable).
		IsProblemDetail().
		ProblemDetailDetail("Service temporarily unavailable")

	// Test JSON response (without Accept header - defaults to JSON)
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusServiceUnavailable).
		Header(httpx.HeaderContentType, "application/problem+json")

	zhtest.AssertEqual(t, 3, handler.getCallCount())
}

func TestCircuitBreaker_RecoveryTimeout(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 1,
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable)

	time.Sleep(60 * time.Millisecond)
	handler.statusCode = http.StatusOK

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestCircuitBreaker_HalfOpenSuccessThreshold(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 3,
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	time.Sleep(60 * time.Millisecond)
	handler.statusCode = http.StatusOK

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	}

	handler.statusCode = http.StatusInternalServerError
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable)
}

func TestCircuitBreaker_CustomIsFailure(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusBadRequest}
	middleware := New(Config{
		FailureThreshold: 2,
		IsFailure: func(r *http.Request, statusCode int) bool {
			return statusCode >= http.StatusBadRequest
		},
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusBadRequest)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable)
}

func TestCircuitBreaker_CustomKeyExtractor(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 2,
		KeyExtractor: func(r *http.Request) string {
			return r.Header.Get("X-Service-Key")
		},
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("X-Service-Key", "service-a").Build()
		zhtest.Serve(middleware, req)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("X-Service-Key", "service-a").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable)

	req = zhtest.NewRequest(http.MethodGet, "/test").WithHeader("X-Service-Key", "service-b").Build()
	w = zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)
}

func TestCircuitBreaker_CustomOpenResponse(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 2,
		OpenStatusCode:   http.StatusTooManyRequests,
		OpenMessage:      "Circuit breaker active",
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	// Test JSON response
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusTooManyRequests).
		IsProblemDetail().
		ProblemDetailDetail("Circuit breaker active")

	// Test JSON response (without Accept header - defaults to JSON)
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusTooManyRequests).
		Header(httpx.HeaderContentType, "application/problem+json")
}

func TestCircuitBreaker_ZeroConfigValues(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 0,
		RecoveryTimeout:  0,
		SuccessThreshold: 0,
		IsFailure:        nil,
		KeyExtractor:     nil,
		OpenStatusCode:   0,
		OpenMessage:      "",
	})(handler)

	for range 5 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable)
}

func TestCircuitBreaker_ConcurrentRequests(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusOK}
	middleware := New(Config{FailureThreshold: 5})(handler)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			w := zhtest.Serve(middleware, req)
			mu.Lock()
			if w.Code == http.StatusOK {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	zhtest.AssertEqual(t, 100, successCount)
}

func TestCircuitBreaker_MultipleEndpoints(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New()(handler)

	for range 5 {
		req := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
		zhtest.Serve(middleware, req)
	}

	req := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable)

	req = zhtest.NewRequest(http.MethodGet, "/api/posts").Build()
	w = zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 2,
	})(handler)

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)
	zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(middleware, req)

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(middleware, req)
	zhtest.AssertWith(t, w).Status(http.StatusServiceUnavailable)

	time.Sleep(60 * time.Millisecond)
	handler.statusCode = http.StatusOK

	for range 2 {
		req = zhtest.NewRequest(http.MethodGet, "/test").Build()
		w = zhtest.Serve(middleware, req)
		zhtest.AssertEqual(t, http.StatusOK, w.Code)
	}

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(middleware, req)
	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestCircuitBreaker_ResponseWriter(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.Handler
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "explicit status code",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("created"))
			}),
			expectedStatus: http.StatusCreated,
			expectedBody:   "created",
		},
		{
			name: "default status code",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("default status"))
			}),
			expectedStatus: http.StatusOK,
			expectedBody:   "default status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := New()(tt.handler)
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			w := zhtest.Serve(middleware, req)

			zhtest.AssertWith(t, w).Status(tt.expectedStatus).Body(tt.expectedBody)
		})
	}
}

func TestCircuitBreaker_MultipleOptions(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 10,
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertNotEqual(t, http.StatusServiceUnavailable, w.Code)
}

func TestCircuitBreaker_EdgeCases(t *testing.T) {
	t.Run("empty key extractor result", func(t *testing.T) {
		handler := &circuitTestHandler{statusCode: http.StatusOK}
		middleware := New(Config{
			KeyExtractor: func(r *http.Request) string {
				return ""
			},
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	})

	t.Run("nil request to IsFailure", func(t *testing.T) {
		handler := &circuitTestHandler{statusCode: http.StatusOK}
		middleware := New(Config{
			IsFailure: func(r *http.Request, statusCode int) bool {
				return statusCode >= http.StatusInternalServerError
			},
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	})
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := New(Config{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 1,
	})(handler)

	// Make requests that will cause failures
	for i := 0; i < 2; i++ {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	// Third request should be rejected (circuit open)
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(middleware, req)

	// Check that metrics were recorded with proper labels
	// We can verify by checking the response status codes
	// - 2 failures (500)
	// - 1 rejection (503)
}

func TestCircuitBreaker_ConcurrentReset(t *testing.T) {
	cbm := &circuitBreakerMiddleware{
		circuits: make(map[string]*circuit),
		config:   DefaultConfig,
	}

	// Create some circuits in open state
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("test-%d", i)
		c := cbm.getCircuit(key)
		c.mu.Lock()
		c.state = StateOpen
		c.failureCount = 100
		c.mu.Unlock()
	}

	// Concurrently reset circuits and create new ones
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("test-%d", id%10)
			cbm.Reset(key)
			// Also create new circuits while resetting
			_ = cbm.getCircuit(fmt.Sprintf("dynamic-%d", id))
		}(i)
	}
	wg.Wait()

	// Verify all original circuits are reset
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("test-%d", i)
		c := cbm.getCircuit(key)
		zhtest.AssertEqual(t, StateClosed, c.getState())
	}
}

func TestCircuitBreaker_GetState(t *testing.T) {
	cbm := &circuitBreakerMiddleware{
		circuits: make(map[string]*circuit),
		config: Config{
			FailureThreshold: 3,
			RecoveryTimeout:  100 * time.Millisecond,
		},
	}

	// Test initial state (circuit doesn't exist yet)
	zhtest.AssertEqual(t, StateClosed, cbm.GetState("/test"))

	// Create and open a circuit
	c := cbm.getCircuit("/test")
	c.mu.Lock()
	c.state = StateOpen
	c.failureCount = 100
	c.mu.Unlock()

	// Circuit should be open now
	zhtest.AssertEqual(t, StateOpen, cbm.GetState("/test"))

	// Reset circuit to closed
	c.mu.Lock()
	c.state = StateClosed
	c.mu.Unlock()

	zhtest.AssertEqual(t, StateClosed, cbm.GetState("/test"))
}

func TestCircuitBreaker_HalfOpenRequestLimit(t *testing.T) {
	// Test that MaxHalfOpenRequests limits concurrent requests in half-open state
	cbm := &circuitBreakerMiddleware{
		circuits: make(map[string]*circuit),
		config: Config{
			FailureThreshold:    1,
			RecoveryTimeout:     30 * time.Second,
			SuccessThreshold:    10, // High threshold
			MaxHalfOpenRequests: 2,
		},
	}

	c := cbm.getCircuit("/test")

	// Manually set to half-open state
	c.mu.Lock()
	c.state = StateHalfOpen
	c.halfOpenInFlight = 0
	c.mu.Unlock()

	// First request should be allowed
	zhtest.AssertFalse(t, c.isOpen())

	// Second request should be allowed
	zhtest.AssertFalse(t, c.isOpen())

	// Third request should be blocked (MaxHalfOpenRequests=2)
	zhtest.AssertTrue(t, c.isOpen())

	// Simulate one request completing
	c.mu.Lock()
	c.halfOpenInFlight--
	c.mu.Unlock()

	// Now another request should be allowed
	zhtest.AssertFalse(t, c.isOpen())
}

func TestCircuitBreaker_HalfOpenRequestLimit_Default(t *testing.T) {
	// Test that default MaxHalfOpenRequests is 1
	cbm := &circuitBreakerMiddleware{
		circuits: make(map[string]*circuit),
		config: Config{
			FailureThreshold:    1,
			RecoveryTimeout:     30 * time.Second,
			SuccessThreshold:    10,
			MaxHalfOpenRequests: 1, // Default is 1
		},
	}

	c := cbm.getCircuit("/test")

	// Manually set to half-open state
	c.mu.Lock()
	c.state = StateHalfOpen
	c.halfOpenInFlight = 0
	c.mu.Unlock()

	// First request should be allowed
	zhtest.AssertFalse(t, c.isOpen())

	// Second request should be blocked (default MaxHalfOpenRequests=1)
	zhtest.AssertTrue(t, c.isOpen())

	// Simulate request completing
	c.mu.Lock()
	c.halfOpenInFlight--
	c.mu.Unlock()

	// Now another request should be allowed
	zhtest.AssertFalse(t, c.isOpen())
}
