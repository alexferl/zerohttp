package circuitbreaker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/metrics"
)

// BenchmarkCircuitBreaker_States measures performance in different circuit states
func BenchmarkCircuitBreaker_States(b *testing.B) {
	states := []struct {
		name       string
		setupFunc  func(*circuit)
		expectOpen bool
	}{
		{
			name: "Closed",
			setupFunc: func(c *circuit) {
				c.state = StateClosed
			},
			expectOpen: false,
		},
		{
			name: "Open",
			setupFunc: func(c *circuit) {
				c.state = StateOpen
				c.lastFailureTime = time.Now()
			},
			expectOpen: true,
		},
		{
			name: "HalfOpen",
			setupFunc: func(c *circuit) {
				c.state = StateHalfOpen
				c.halfOpenInFlight = 0
			},
			expectOpen: false,
		},
	}

	for _, s := range states {
		b.Run(s.name, func(b *testing.B) {
			cbm := &circuitBreakerMiddleware{
				circuits: make(map[string]*circuit),
				config:   DefaultConfig,
			}

			// Pre-create circuit in desired state
			c := cbm.getCircuit("test-key")
			s.setupFunc(c)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				c := cbm.getCircuit("test-key")
				c.isOpen()
			}
		})
	}
}

// BenchmarkCircuitBreaker_FullMiddleware measures the complete middleware overhead
func BenchmarkCircuitBreaker_FullMiddleware(b *testing.B) {
	scenarios := []struct {
		name string
		code int
	}{
		{"Success", http.StatusOK},
		{"Failure", http.StatusInternalServerError},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			cbm := New(Config{
				FailureThreshold: 100, // High threshold to prevent tripping
			})

			handler := cbm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(s.code)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkCircuitBreaker_Concurrent measures concurrent performance
func BenchmarkCircuitBreaker_Concurrent(b *testing.B) {
	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			cbm := New(Config{
				FailureThreshold: 100,
			})

			handler := cbm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, req)
				}
			})
		})
	}
}

// BenchmarkCircuitBreaker_RecordResult measures result recording performance
func BenchmarkCircuitBreaker_RecordResult(b *testing.B) {
	states := []struct {
		name   string
		state  CircuitState
		status int
	}{
		{"Closed_Success", StateClosed, http.StatusOK},
		{"Closed_Failure", StateClosed, http.StatusInternalServerError},
		{"HalfOpen_Success", StateHalfOpen, http.StatusOK},
		{"HalfOpen_Failure", StateHalfOpen, http.StatusInternalServerError},
	}

	for _, s := range states {
		b.Run(s.name, func(b *testing.B) {
			cbm := &circuitBreakerMiddleware{
				circuits: make(map[string]*circuit),
				config:   DefaultConfig,
			}

			c := cbm.getCircuit("test-key")
			c.state = s.state
			c.halfOpenInFlight = 1 // Simulate in-flight request

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			reg := metrics.NewRegistry()

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				c.recordResult(req, s.status, reg, "test-key")
			}
		})
	}
}

// BenchmarkCircuitBreaker_GetCircuit measures circuit lookup/creation performance
func BenchmarkCircuitBreaker_GetCircuit(b *testing.B) {
	scenarios := []struct {
		name     string
		numKeys  int
		keyIndex int // Which key to look up (0 = first created, numKeys-1 = last)
	}{
		{"SingleKey", 1, 0},
		{"10Keys_First", 10, 0},
		{"10Keys_Last", 10, 9},
		{"100Keys_Last", 100, 99},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			cbm := &circuitBreakerMiddleware{
				circuits: make(map[string]*circuit),
				config:   DefaultConfig,
			}

			// Pre-populate circuits
			keys := make([]string, s.numKeys)
			for i := 0; i < s.numKeys; i++ {
				keys[i] = fmt.Sprintf("key-%d", i)
				_ = cbm.getCircuit(keys[i])
			}

			targetKey := keys[s.keyIndex]

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				cbm.getCircuit(targetKey)
			}
		})
	}
}

// BenchmarkCircuitBreaker_ConfigVariations measures impact of different configurations
func BenchmarkCircuitBreaker_ConfigVariations(b *testing.B) {
	configs := []struct {
		name string
		cfg  Config
	}{
		{
			name: "Default",
			cfg:  DefaultConfig,
		},
		{
			name: "HighThreshold",
			cfg: Config{
				FailureThreshold: 1000,
				RecoveryTimeout:  time.Minute,
				SuccessThreshold: 100,
			},
		},
		{
			name: "LowThreshold",
			cfg: Config{
				FailureThreshold: 2,
				RecoveryTimeout:  time.Second,
				SuccessThreshold: 1,
			},
		},
		{
			name: "CustomKeyExtractor",
			cfg: Config{
				KeyExtractor: func(r *http.Request) string {
					return r.Header.Get("X-Service-Key")
				},
			},
		},
	}

	for _, c := range configs {
		b.Run(c.name, func(b *testing.B) {
			cbm := New(c.cfg)

			handler := cbm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Service-Key", "service-a")

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkCircuitBreaker_HalfOpenLimiting measures half-open request limiting
func BenchmarkCircuitBreaker_HalfOpenLimiting(b *testing.B) {
	limits := []int{1, 5, 10}

	for _, limit := range limits {
		b.Run(fmt.Sprintf("Limit%d", limit), func(b *testing.B) {
			cbm := &circuitBreakerMiddleware{
				circuits: make(map[string]*circuit),
				config: Config{
					MaxHalfOpenRequests: limit,
				},
			}

			c := cbm.getCircuit("test-key")
			c.state = StateHalfOpen
			c.halfOpenInFlight = 0

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				c.isOpen()
			}
		})
	}
}

// BenchmarkCircuitBreaker_StateTransition measures state transition overhead
func BenchmarkCircuitBreaker_StateTransition(b *testing.B) {
	transitions := []struct {
		name         string
		initialState CircuitState
		failures     int
		successes    int
	}{
		{
			name:         "ClosedToOpen",
			initialState: StateClosed,
			failures:     5,
			successes:    0,
		},
		{
			name:         "HalfOpenToClosed",
			initialState: StateHalfOpen,
			failures:     0,
			successes:    5,
		},
		{
			name:         "HalfOpenToOpen",
			initialState: StateHalfOpen,
			failures:     1,
			successes:    0,
		},
	}

	for _, t := range transitions {
		b.Run(t.name, func(b *testing.B) {
			cfg := DefaultConfig
			cfg.FailureThreshold = 5
			cfg.SuccessThreshold = 5
			cbm := &circuitBreakerMiddleware{
				circuits: make(map[string]*circuit),
				config:   cfg,
			}

			c := cbm.getCircuit("test-key")
			c.state = t.initialState
			c.halfOpenInFlight = 1

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			reg := metrics.NewRegistry()

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				// Reset state
				c.state = t.initialState
				c.failureCount = 0
				c.successCount = 0

				// Record failures
				for i := 0; i < t.failures; i++ {
					c.recordResult(req, http.StatusInternalServerError, reg, "test-key")
				}

				// Record successes
				for i := 0; i < t.successes; i++ {
					c.recordResult(req, http.StatusOK, reg, "test-key")
				}
			}
		})
	}
}

// BenchmarkCircuitBreaker_Baseline compares against no middleware
func BenchmarkCircuitBreaker_Baseline(b *testing.B) {
	b.Run("NoMiddleware", func(b *testing.B) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})

	b.Run("WithCircuitBreaker", func(b *testing.B) {
		cbm := New(Config{
			FailureThreshold: 100,
		})

		handler := cbm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkCircuitBreaker_MultipleCircuits measures performance with many circuits
func BenchmarkCircuitBreaker_MultipleCircuits(b *testing.B) {
	circuitCounts := []int{1, 10, 100, 1000}

	for _, count := range circuitCounts {
		b.Run(fmt.Sprintf("Circuits%d", count), func(b *testing.B) {
			cbm := &circuitBreakerMiddleware{
				circuits: make(map[string]*circuit),
				config:   DefaultConfig,
			}

			// Pre-create circuits
			keys := make([]string, count)
			for i := 0; i < count; i++ {
				keys[i] = fmt.Sprintf("key-%d", i)
				_ = cbm.getCircuit(keys[i])
			}

			b.ReportAllocs()
			b.ResetTimer()

			i := 0
			for b.Loop() {
				c := cbm.getCircuit(keys[i%count])
				c.isOpen()
				i++
			}
		})
	}
}
