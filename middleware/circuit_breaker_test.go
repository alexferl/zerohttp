package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
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
	if _, err := w.Write([]byte("response")); err != nil {
		panic(fmt.Errorf("failed to write test response: %w", err))
	}
}

func (h *circuitTestHandler) getCallCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.callCount
}

func TestCircuitBreaker_FailureThreshold(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(3),
		config.WithCircuitBreakerRecoveryTimeout(100*time.Millisecond),
	)(handler)

	for i := range 3 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusInternalServerError, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected circuit open status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
	if w.Body.String() != "Service temporarily unavailable" {
		t.Errorf("Expected circuit open message, got '%s'", w.Body.String())
	}
	if handler.getCallCount() != 3 {
		t.Errorf("Expected handler to be called 3 times, got %d", handler.getCallCount())
	}
}

func TestCircuitBreaker_RecoveryTimeout(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(2),
		config.WithCircuitBreakerRecoveryTimeout(50*time.Millisecond),
		config.WithCircuitBreakerSuccessThreshold(1),
	)(handler)

	for range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Error("Expected circuit to be open")
	}

	time.Sleep(60 * time.Millisecond)
	handler.statusCode = http.StatusOK

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected successful request after recovery, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected circuit to be closed, got %d", w.Code)
	}
}

func TestCircuitBreaker_HalfOpenSuccessThreshold(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(2),
		config.WithCircuitBreakerRecoveryTimeout(50*time.Millisecond),
		config.WithCircuitBreakerSuccessThreshold(3),
	)(handler)

	for range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	time.Sleep(60 * time.Millisecond)
	handler.statusCode = http.StatusOK

	for i := range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected success in half-open, got %d", i+1, w.Code)
		}
	}

	handler.statusCode = http.StatusInternalServerError
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Error("Expected failure to reopen circuit")
	}

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Error("Expected circuit to be open again")
	}
}

func TestCircuitBreaker_CustomIsFailure(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusBadRequest}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(2),
		config.WithCircuitBreakerIsFailure(func(r *http.Request, statusCode int) bool {
			return statusCode >= 400
		}),
	)(handler)

	for range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Error("Expected circuit to be open with custom failure function")
	}
}

func TestCircuitBreaker_CustomKeyExtractor(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(2),
		config.WithCircuitBreakerKeyExtractor(func(r *http.Request) string {
			return r.Header.Get("X-Service-Key")
		}),
	)(handler)

	for range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Service-Key", "service-a")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Service-Key", "service-a")
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Error("Expected service-a circuit to be open")
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Service-Key", "service-b")
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Error("Expected service-b to still process requests")
	}
}

func TestCircuitBreaker_CustomOpenResponse(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(2),
		config.WithCircuitBreakerOpenStatusCode(http.StatusTooManyRequests),
		config.WithCircuitBreakerOpenMessage("Circuit breaker active"),
	)(handler)

	for range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected custom status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
	if w.Body.String() != "Circuit breaker active" {
		t.Errorf("Expected custom message 'Circuit breaker active', got '%s'", w.Body.String())
	}
}

func TestCircuitBreaker_ZeroConfigValues(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(0),
		config.WithCircuitBreakerRecoveryTimeout(0),
		config.WithCircuitBreakerSuccessThreshold(0),
		config.WithCircuitBreakerIsFailure(nil),
		config.WithCircuitBreakerKeyExtractor(nil),
		config.WithCircuitBreakerOpenStatusCode(0),
		config.WithCircuitBreakerOpenMessage(""),
	)(handler)

	for range 5 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Error("Expected circuit to use default configuration")
	}
}

func TestCircuitBreaker_ConcurrentRequests(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusOK}
	middleware := CircuitBreaker(config.WithCircuitBreakerFailureThreshold(5))(handler)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
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
		req := httptest.NewRequest("GET", "/api/users", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Error("Expected /api/users circuit to be open")
	}

	req = httptest.NewRequest("GET", "/api/posts", nil)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Error("Expected /api/posts to still process requests")
	}
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(2),
		config.WithCircuitBreakerRecoveryTimeout(50*time.Millisecond),
		config.WithCircuitBreakerSuccessThreshold(2),
	)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Error("Expected request to go through in closed state")
	}

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Error("Expected circuit to be open")
	}

	time.Sleep(60 * time.Millisecond)
	handler.statusCode = http.StatusOK

	for i := range 2 {
		req = httptest.NewRequest("GET", "/test", nil)
		w = httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected success in half-open state, iteration %d", i)
		}
	}

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Error("Expected circuit to be closed after successful recovery")
	}
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
				_, err := w.Write([]byte("created"))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}),
			expectedStatus: http.StatusCreated,
			expectedBody:   "created",
		},
		{
			name: "default status code",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := w.Write([]byte("default status"))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}),
			expectedStatus: http.StatusOK,
			expectedBody:   "default status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := CircuitBreaker()(tt.handler)
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestCircuitBreaker_MultipleOptions(t *testing.T) {
	handler := &circuitTestHandler{statusCode: http.StatusInternalServerError}
	middleware := CircuitBreaker(
		config.WithCircuitBreakerFailureThreshold(2),
		config.WithCircuitBreakerFailureThreshold(10),
	)(handler)

	for range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code == http.StatusServiceUnavailable {
		t.Error("Expected circuit to use last option's threshold (not be open yet)")
	}
}

func TestCircuitBreaker_EdgeCases(t *testing.T) {
	t.Run("empty key extractor result", func(t *testing.T) {
		handler := &circuitTestHandler{statusCode: http.StatusOK}
		middleware := CircuitBreaker(
			config.WithCircuitBreakerKeyExtractor(func(r *http.Request) string {
				return ""
			}),
		)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Error("Expected request to work with empty key")
		}
	})

	t.Run("nil request to IsFailure", func(t *testing.T) {
		handler := &circuitTestHandler{statusCode: http.StatusOK}
		middleware := CircuitBreaker(
			config.WithCircuitBreakerIsFailure(func(r *http.Request, statusCode int) bool {
				return statusCode >= 500
			}),
		)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Error("Expected request to work")
		}
	})
}
