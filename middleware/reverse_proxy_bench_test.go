package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkReverseProxy_Baseline measures basic proxy overhead
func BenchmarkReverseProxy_Baseline(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	}))
	defer upstream.Close()

	mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
		Target: upstream.URL,
	})
	defer cleanup()

	handler := mw(nil)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkReverseProxy_LoadBalancers compares different load balancing algorithms
func BenchmarkReverseProxy_LoadBalancers(b *testing.B) {
	// Create 3 upstream servers
	upstreams := make([]*httptest.Server, 3)
	for i := range upstreams {
		upstreams[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer upstreams[i].Close()
	}

	algorithms := []struct {
		name string
		algo config.LoadBalancerAlgorithm
	}{
		{"RoundRobin", config.RoundRobin},
		{"Random", config.Random},
		{"LeastConnections", config.LeastConnections},
	}

	for _, tc := range algorithms {
		b.Run(tc.name, func(b *testing.B) {
			targets := []config.Backend{
				{Target: upstreams[0].URL, Weight: 1, Healthy: true},
				{Target: upstreams[1].URL, Weight: 1, Healthy: true},
				{Target: upstreams[2].URL, Weight: 1, Healthy: true},
			}

			mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
				Targets:      targets,
				LoadBalancer: tc.algo,
			})
			defer cleanup()

			handler := mw(nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
		})
	}
}

// BenchmarkReverseProxy_BackendCount measures performance with different numbers of backends
func BenchmarkReverseProxy_BackendCount(b *testing.B) {
	backendCounts := []int{1, 3, 10}

	for _, count := range backendCounts {
		b.Run(fmt.Sprintf("Backends%d", count), func(b *testing.B) {
			// Create upstream servers
			upstreams := make([]*httptest.Server, count)
			targets := make([]config.Backend, count)

			for i := range upstreams {
				upstreams[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				defer upstreams[i].Close()
				targets[i] = config.Backend{Target: upstreams[i].URL, Weight: 1, Healthy: true}
			}

			mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
				Targets:      targets,
				LoadBalancer: config.RoundRobin,
			})
			defer cleanup()

			handler := mw(nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
		})
	}
}

// BenchmarkReverseProxy_StripPrefix measures path stripping overhead
func BenchmarkReverseProxy_StripPrefix(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	b.Run("WithoutStrip", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target: upstream.URL,
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("WithStrip", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target:      upstream.URL,
			StripPrefix: "/api",
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkReverseProxy_AddPrefix measures path prefix addition overhead
func BenchmarkReverseProxy_AddPrefix(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	b.Run("WithoutPrefix", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target: upstream.URL,
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("WithPrefix", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target:    upstream.URL,
			AddPrefix: "/v1",
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkReverseProxy_Rewrites measures URL rewrite overhead
func BenchmarkReverseProxy_Rewrites(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	b.Run("WithoutRewrites", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target: upstream.URL,
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/old/path", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("WithRewrites", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target: upstream.URL,
			Rewrites: []config.RewriteRule{
				{Pattern: "/old/*", Replacement: "/new/path"},
			},
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/old/path", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkReverseProxy_ExcludedPaths measures excluded path checking overhead
func BenchmarkReverseProxy_ExcludedPaths(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	testCases := []struct {
		name          string
		excludedPaths []string
		path          string
	}{
		{"NoExcluded", nil, "/api/users"},
		{"OneExcluded_NoMatch", []string{"/health"}, "/api/users"},
		{"ManyExcluded_NoMatch", []string{"/health", "/metrics", "/ready", "/live"}, "/api/users"},
		{"ExcludedMatch", []string{"/health"}, "/health"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
				Target:        upstream.URL,
				ExcludedPaths: tc.excludedPaths,
			})
			defer cleanup()
			handler := mw(nextHandler)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, tc.path, nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
		})
	}
}

// BenchmarkReverseProxy_ForwardHeaders measures X-Forwarded-* header handling
func BenchmarkReverseProxy_ForwardHeaders(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	b.Run("WithoutForwardHeaders", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target:         upstream.URL,
			ForwardHeaders: false,
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = "example.com"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("WithForwardHeaders", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target:         upstream.URL,
			ForwardHeaders: true,
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = "example.com"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkReverseProxy_SetHeaders measures header manipulation overhead
func BenchmarkReverseProxy_SetHeaders(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	b.Run("NoHeaders", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target: upstream.URL,
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("SetHeaders", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target: upstream.URL,
			SetHeaders: map[string]string{
				"X-Custom-Header": "value",
				"X-Request-ID":    "12345",
			},
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("RemoveHeaders", func(b *testing.B) {
		mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
			Target:        upstream.URL,
			RemoveHeaders: []string{"X-Internal-Token", "X-Secret"},
		})
		defer cleanup()
		handler := mw(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Internal-Token", "secret")
			req.Header.Set("X-Secret", "value")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkReverseProxy_UnhealthyBackends measures behavior when some backends are unhealthy
func BenchmarkReverseProxy_UnhealthyBackends(b *testing.B) {
	upstreams := make([]*httptest.Server, 5)
	for i := range upstreams {
		upstreams[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer upstreams[i].Close()
	}

	scenarios := []struct {
		name           string
		healthyCount   int
		unhealthyCount int
	}{
		{"AllHealthy", 5, 0},
		{"OneUnhealthy", 4, 1},
		{"HalfUnhealthy", 2, 3},
		{"OneHealthy", 1, 4},
	}

	for _, tc := range scenarios {
		b.Run(tc.name, func(b *testing.B) {
			targets := make([]config.Backend, 5)
			for i := range targets {
				targets[i] = config.Backend{
					Target:  upstreams[i].URL,
					Weight:  1,
					Healthy: i < tc.healthyCount,
				}
			}

			mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
				Targets:      targets,
				LoadBalancer: config.RoundRobin,
			})
			defer cleanup()
			handler := mw(nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
		})
	}
}

// BenchmarkReverseProxy_Concurrent measures concurrent proxy performance
func BenchmarkReverseProxy_Concurrent(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	targets := []config.Backend{
		{Target: upstream.URL, Weight: 1, Healthy: true},
		{Target: upstream.URL, Weight: 1, Healthy: true},
		{Target: upstream.URL, Weight: 1, Healthy: true},
	}

	mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
		Targets:      targets,
		LoadBalancer: config.RoundRobin,
	})
	defer cleanup()
	handler := mw(nil)

	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, req)
				}
			})
		})
	}
}

// BenchmarkReverseProxy_BackendSelection benchmarks just the backend selection logic
func BenchmarkReverseProxy_BackendSelection(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	targets := make([]config.Backend, 10)
	for i := range targets {
		targets[i] = config.Backend{
			Target:  upstream.URL,
			Weight:  1,
			Healthy: true,
		}
	}

	mw, cleanup := ReverseProxy(config.ReverseProxyConfig{
		Targets:      targets,
		LoadBalancer: config.RoundRobin,
	})
	defer cleanup()

	// Access the internal reverse proxy to benchmark selection
	handler := mw(nil)

	b.Run("RoundRobin", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}
