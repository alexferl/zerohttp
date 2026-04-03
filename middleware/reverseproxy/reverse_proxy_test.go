package reverseproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestReverseProxy_SingleTarget(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream response: " + r.URL.Path))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zhtest.AssertFail(t, "next handler should not be called")
	})

	mw(next).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("upstream response: /test")
}

func TestReverseProxy_StripPrefix(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:      upstream.URL,
		StripPrefix: "/api",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("path: /users")
}

func TestReverseProxy_StripPrefixRoot(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:      upstream.URL,
		StripPrefix: "/api",
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("path: /")
}

func TestReverseProxy_AddPrefix(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:    upstream.URL,
		AddPrefix: "/v2",
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("path: /v2/users")
}

func TestReverseProxy_StripAndAddPrefix(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:      upstream.URL,
		StripPrefix: "/api",
		AddPrefix:   "/v2",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("path: /v2/users")
}

func TestReverseProxy_Rewrite(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
		Rewrites: []RewriteRule{
			{Pattern: "/old/*", Replacement: "/new/path"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/old/users", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("path: /new/path")
}

func TestReverseProxy_SetHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.Header.Get("X-Custom")))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
		SetHeaders: map[string]string{
			"X-Custom": "custom-value",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("custom-value")
}

func TestReverseProxy_RemoveHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Remove") == "" {
			_, _ = w.Write([]byte("removed"))
		} else {
			_, _ = w.Write([]byte("present"))
		}
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:        upstream.URL,
		RemoveHeaders: []string{"X-Remove"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Remove", "value")
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("removed")
}

func TestReverseProxy_ForwardHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		xfp := r.Header.Get("X-Forwarded-Proto")
		xfh := r.Header.Get("X-Forwarded-Host")
		_, _ = w.Write([]byte(xff + "|" + xfp + "|" + xfh))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:         upstream.URL,
		ForwardHeaders: config.Bool(true),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
	body := rec.Body.String()
	zhtest.AssertTrue(t, strings.Contains(body, "http"))
	zhtest.AssertTrue(t, strings.Contains(body, "example.com"))
}

func TestReverseProxy_ExcludedPaths(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("upstream"))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:        upstream.URL,
		ExcludedPaths: []string{"/health", "/metrics"},
	})

	// Test excluded path - should call next handler
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		_, _ = w.Write([]byte("excluded"))
	})

	mw(next).ServeHTTP(rec, req)

	zhtest.AssertTrue(t, nextCalled)
	zhtest.AssertWith(t, rec).Body("excluded")

	// Test non-excluded path - should go to upstream
	req2 := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec2 := httptest.NewRecorder()

	mw(next).ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).Body("upstream")
}

func TestReverseProxy_ModifyRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.Header.Get("X-Modified")))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
		ModifyRequest: func(r *http.Request) {
			r.Header.Set("X-Modified", "modified-value")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("modified-value")
}

func TestReverseProxy_ModifyResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Original", "original")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("body"))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
		ModifyResponse: func(r *http.Response) error {
			r.Header.Set("X-Modified", "modified")
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Header("X-Modified", "modified")
}

func TestReverseProxy_ErrorHandler(t *testing.T) {
	mw, _ := New(Config{
		Target: "http://localhost:1",
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("custom error"))
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusServiceUnavailable).Body("custom error")
}

func TestReverseProxy_FallbackHandler(t *testing.T) {
	mw, _ := New(Config{
		Targets: []Backend{
			{Target: "http://localhost:1", Healthy: config.Bool(false)},
		},
		HealthCheckInterval: 0,
		FallbackHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("fallback response"))
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusServiceUnavailable).Body("fallback response")
}

func TestReverseProxy_LoadBalancer_RoundRobin(t *testing.T) {
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend1"))
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend2"))
	}))
	defer upstream2.Close()

	mw, _ := New(Config{
		Targets: []Backend{
			{Target: upstream1.URL, Healthy: config.Bool(true)},
			{Target: upstream2.URL, Healthy: config.Bool(true)},
		},
		LoadBalancer: RoundRobin,
	})

	handler := mw(nil)

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Second request
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Both should get responses (we don't know which order due to round-robin)
	body1 := rec1.Body.String()
	body2 := rec2.Body.String()
	zhtest.AssertTrue(t, body1 == "backend1" || body1 == "backend2")
	zhtest.AssertTrue(t, body2 == "backend1" || body2 == "backend2")
}

func TestReverseProxy_LoadBalancer_Random(t *testing.T) {
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend1"))
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend2"))
	}))
	defer upstream2.Close()

	mw, _ := New(Config{
		Targets: []Backend{
			{Target: upstream1.URL, Healthy: config.Bool(true)},
			{Target: upstream2.URL, Healthy: config.Bool(true)},
		},
		LoadBalancer: Random,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
	body := rec.Body.String()
	zhtest.AssertTrue(t, body == "backend1" || body == "backend2")
}

func TestReverseProxy_LoadBalancer_LeastConnections(t *testing.T) {
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		_, _ = w.Write([]byte("backend1"))
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend2"))
	}))
	defer upstream2.Close()

	mw, _ := New(Config{
		Targets: []Backend{
			{Target: upstream1.URL, Healthy: config.Bool(true)},
			{Target: upstream2.URL, Healthy: config.Bool(true)},
		},
		LoadBalancer: LeastConnections,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

func TestReverseProxy_PanicOnMissingTarget(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		New(Config{
			Target: "",
		})
	})
}

func TestReverseProxy_PanicOnInvalidURL(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		New(Config{
			Target: "://invalid-url",
		})
	})
}

func TestReverseProxy_HealthCheck(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(true)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if healthy.Load() {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Targets: []Backend{
			{Target: upstream.URL, Healthy: config.Bool(true)},
		},
		HealthCheckInterval: 100 * time.Millisecond,
		HealthCheckPath:     "/health",
		HealthCheckTimeout:  1 * time.Second,
	})

	handler := mw(nil)

	// First request should work
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	zhtest.AssertWith(t, rec1).Status(http.StatusOK).Body("ok")

	// Mark backend as unhealthy
	healthy.Store(false)

	// Wait for health check to run
	time.Sleep(200 * time.Millisecond)

	// Now request should fail
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).Status(http.StatusBadGateway)
}

func TestReverseProxy_FlushInterval(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("response"))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:        upstream.URL,
		FlushInterval: 100 * time.Millisecond,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

func TestReverseProxy_CustomTransport(t *testing.T) {
	customTransport := &customTestTransport{
		onRoundTrip: func(r *http.Request) (*http.Response, error) {
			return http.DefaultTransport.RoundTrip(r)
		},
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:    upstream.URL,
		Transport: customTransport,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

type customTestTransport struct {
	onRoundTrip func(*http.Request) (*http.Response, error)
}

func (t *customTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.onRoundTrip(r)
}

func TestReverseProxy_WithNextHandler(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("upstream"))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		_, _ = w.Write([]byte("next"))
	})

	mw(next).ServeHTTP(rec, req)

	zhtest.AssertFalse(t, nextCalled)
	zhtest.AssertWith(t, rec).Body("upstream")
}

func TestReverseProxy_NoHealthyBackends(t *testing.T) {
	mw, _ := New(Config{
		Targets: []Backend{
			{Target: "http://localhost:1", Healthy: config.Bool(false)},
		},
		HealthCheckInterval: 0,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusBadGateway)
}

func TestReverseProxy_ResponseHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "value")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusCreated).Header("X-Upstream", "value")
}

func TestReverseProxy_PostBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request body"))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMETextPlain)
	rec := httptest.NewRecorder()
	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("request body")
}

func TestReverseProxy_QueryParams(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.RawQuery))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target: upstream.URL,
	})

	req := httptest.NewRequest(http.MethodGet, "/?foo=bar&baz=qux", nil)
	rec := httptest.NewRecorder()
	mw(nil).ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK).Body("foo=bar&baz=qux")
}

func TestReverseProxy_HealthCheckCleanup(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// Create reverse proxy with health checks
	mw, cleanup := New(Config{
		Targets: []Backend{
			{Target: upstream.URL, Healthy: config.Bool(true)},
		},
		HealthCheckInterval: 50 * time.Millisecond,
		HealthCheckPath:     "/health",
		HealthCheckTimeout:  100 * time.Millisecond,
	})

	handler := mw(nil)

	// Make a request to ensure handler is working
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Wait for a health check cycle
	time.Sleep(75 * time.Millisecond)

	// Call cleanup - this should stop health checks
	cleanup()

	// Wait a bit and verify no panic occurs
	time.Sleep(100 * time.Millisecond)

	// Handler should still work after cleanup
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertEqual(t, http.StatusOK, rec2.Code)
}

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

func TestReverseProxy_proxyResponseRecorder_Flush(t *testing.T) {
	tests := []struct {
		name              string
		underlyingFlusher bool
		expectFlushCalled bool
	}{
		{
			name:              "flush passes through to underlying Flusher",
			underlyingFlusher: true,
			expectFlushCalled: true,
		},
		{
			name:              "flush no-op when underlying doesn't implement Flusher",
			underlyingFlusher: false,
			expectFlushCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var base http.ResponseWriter
			var flushCalled *bool

			if tt.underlyingFlusher {
				rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}
				base = rec
				flushCalled = &rec.flushed
			} else {
				rec := httptest.NewRecorder()
				base = rec
				flushCalled = new(bool)
			}

			// Wrap with proxyResponseRecorder
			prr := &proxyResponseRecorder{
				ResponseWriter: base,
			}

			// Call Flush
			prr.Flush()

			zhtest.AssertEqual(t, tt.expectFlushCalled, *flushCalled)
		})
	}
}

func TestReverseProxy_IncludedPaths(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream response: " + r.URL.Path))
	}))
	defer upstream.Close()

	mw, _ := New(Config{
		Target:        upstream.URL,
		IncludedPaths: []string{"/api/", "/admin"},
	})

	tests := []struct {
		name           string
		path           string
		expectUpstream bool
		expectBody     string
	}{
		{"allowed path - goes to upstream", "/api/users", true, "upstream response: /api/users"},
		{"allowed exact path", "/admin", true, "upstream response: /admin"},
		{"non-allowed path - calls next", "/health", false, "next handler"},
		{"non-allowed path 2", "/metrics", false, "next handler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				_, _ = w.Write([]byte("next handler"))
			})

			mw(next).ServeHTTP(rec, req)

			if tt.expectUpstream {
				zhtest.AssertFalse(t, nextCalled)
			} else {
				zhtest.AssertTrue(t, nextCalled)
			}
			zhtest.AssertWith(t, rec).Body(tt.expectBody)
		})
	}
}

func TestReverseProxy_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	zhtest.AssertPanic(t, func() {
		_, _ = New(Config{
			Target:        upstream.URL,
			ExcludedPaths: []string{"/health"},
			IncludedPaths: []string{"/api"},
		})
	})
}
