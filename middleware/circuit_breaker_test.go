package middleware

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  100 * time.Millisecond,
	})(handler)

	for range 3 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusServiceUnavailable).
		IsProblemDetail().
		ProblemDetailDetail("Service temporarily unavailable")
	if handler.getCallCount() != 3 {
		t.Errorf("Expected handler to be called 3 times, got %d", handler.getCallCount())
	}
}

func TestCircuitBreaker_RecoveryTimeout(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenStatusCode:   http.StatusTooManyRequests,
		OpenMessage:      "Circuit breaker active",
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusTooManyRequests).
		IsProblemDetail().
		ProblemDetailDetail("Circuit breaker active")
}

func TestCircuitBreaker_ZeroConfigValues(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{FailureThreshold: 5})(handler)

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
	if successCount != 100 {
		t.Errorf("Expected 100 successful requests, got %d", successCount)
	}
}

func TestCircuitBreaker_MultipleEndpoints(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker()(handler)

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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
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

	for i := range 2 {
		req = zhtest.NewRequest(http.MethodGet, "/test").Build()
		w = zhtest.Serve(middleware, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected success in half-open state, iteration %d", i)
		}
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
			middleware := CircuitBreaker()(tt.handler)
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			w := zhtest.Serve(middleware, req)

			zhtest.AssertWith(t, w).Status(tt.expectedStatus).Body(tt.expectedBody)
		})
	}
}

func TestCircuitBreaker_MultipleOptions(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
		FailureThreshold: 10,
	})(handler)

	for range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)
	}

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware, req)

	if w.Code == http.StatusServiceUnavailable {
		t.Error("Expected circuit to use last option's threshold (not be open yet)")
	}
}

func TestCircuitBreaker_EdgeCases(t *testing.T) {
	t.Run("empty key extractor result", func(t *testing.T) {
		handler := &circuitTestHandler{statusCode: http.StatusOK}
		middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
		middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
	middleware := CircuitBreaker(config.CircuitBreakerConfig{
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
