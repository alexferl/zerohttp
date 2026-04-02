package zerohttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/log"
)

// noopLogger is a logger that discards all output for benchmarking.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, fields ...log.Field) {}
func (n *noopLogger) Info(msg string, fields ...log.Field)  {}
func (n *noopLogger) Warn(msg string, fields ...log.Field)  {}
func (n *noopLogger) Error(msg string, fields ...log.Field) {}
func (n *noopLogger) Panic(msg string, fields ...log.Field) { panic(msg) }
func (n *noopLogger) Fatal(msg string, fields ...log.Field) {}
func (n *noopLogger) WithFields(fields ...log.Field) log.Logger {
	return n
}

func (n *noopLogger) WithContext(ctx context.Context) log.Logger {
	return n
}

// BenchmarkRouter_SimpleRoute measures the overhead of zerohttp router
// compared to a baseline http.ServeMux for simple route matching.
func BenchmarkRouter_SimpleRoute(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	b.Run("Baseline_ServeMux", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /test", handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_Router", func(b *testing.B) {
		router := NewRouter()
		router.GET("/test", handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_HandlerFunc measures the overhead of HandlerFunc
// (error-returning handlers) compared to standard http.Handler.
func BenchmarkRouter_HandlerFunc(b *testing.B) {
	b.Run("Standard_Handler", func(b *testing.B) {
		router := NewRouter()
		router.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("HandlerFunc_NoError", func(b *testing.B) {
		router := NewRouter()
		router.GET("/test", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
			return nil
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_ParameterizedRoutes measures routing performance
// with parameterized paths (Go 1.22+ pattern syntax).
func BenchmarkRouter_ParameterizedRoutes(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	b.Run("Baseline_ServeMux", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}", handler)

		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_Router", func(b *testing.B) {
		router := NewRouter()
		router.GET("/users/{id}", handler)

		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_NotFound measures 404 handling overhead.
func BenchmarkRouter_NotFound(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	b.Run("Baseline_ServeMux", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /exists", handler)

		req := httptest.NewRequest(http.MethodGet, "/notfound", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_Router", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.GET("/exists", handler)

		req := httptest.NewRequest(http.MethodGet, "/notfound", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_MethodNotAllowed measures 405 handling overhead.
func BenchmarkRouter_MethodNotAllowed(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	b.Run("Baseline_ServeMux", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /test", handler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_Router", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.GET("/test", handler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_MiddlewareOverhead measures the overhead of middleware
// wrapping in zerohttp compared to direct middleware application.
func BenchmarkRouter_MiddlewareOverhead(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	simpleMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	b.Run("Baseline_ServeMux_1Middleware", func(b *testing.B) {
		mux := http.NewServeMux()
		wrapped := simpleMiddleware(handler)
		mux.Handle("GET /test", wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_1Middleware", func(b *testing.B) {
		router := NewRouter(simpleMiddleware)
		router.GET("/test", handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("Baseline_ServeMux_5Middleware", func(b *testing.B) {
		mux := http.NewServeMux()
		var wrapped http.Handler = handler
		for range 5 {
			wrapped = simpleMiddleware(wrapped)
		}
		mux.Handle("GET /test", wrapped)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_5Middleware", func(b *testing.B) {
		middlewares := make([]MiddlewareFunc, 5)
		for i := range 5 {
			middlewares[i] = simpleMiddleware
		}
		router := NewRouter(middlewares...)
		router.GET("/test", handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_ManyRoutes measures routing performance as the number
// of registered routes increases.
func BenchmarkRouter_ManyRoutes(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	routeCounts := []int{10, 50, 100, 500}

	for _, count := range routeCounts {
		b.Run(fmt.Sprintf("Baseline_ServeMux_%dRoutes", count), func(b *testing.B) {
			mux := http.NewServeMux()
			for i := range count {
				mux.HandleFunc(fmt.Sprintf("GET /route%d", i), handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/route0", nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
			}
		})

		b.Run(fmt.Sprintf("Zerohttp_Router_%dRoutes", count), func(b *testing.B) {
			router := NewRouter()
			for i := range count {
				router.GET(fmt.Sprintf("/route%d", i), handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/route0", nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkRouter_RouteGroups measures the overhead of using route groups.
func BenchmarkRouter_RouteGroups(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	b.Run("NoGroups", func(b *testing.B) {
		router := NewRouter()
		router.GET("/api/v1/users", handler)
		router.GET("/api/v1/posts", handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("WithGroups", func(b *testing.B) {
		router := NewRouter()
		router.Group(func(api Router) {
			api.GET("/api/v1/users", handler)
			api.GET("/api/v1/posts", handler)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_HEADRequest measures HEAD request handling overhead.
func BenchmarkRouter_HEADRequest(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body content"))
	})

	b.Run("Baseline_ServeMux", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /test", handler)

		req := httptest.NewRequest(http.MethodHead, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_Router", func(b *testing.B) {
		router := NewRouter()
		router.GET("/test", handler)

		req := httptest.NewRequest(http.MethodHead, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_HandlerFunc", func(b *testing.B) {
		router := NewRouter()
		router.GET("/test", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			return R.Text(w, http.StatusOK, "response body content")
		}))

		req := httptest.NewRequest(http.MethodHead, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_AllHTTPMethods measures routing performance across all HTTP methods.
func BenchmarkRouter_AllHTTPMethods(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}

	for _, method := range methods {
		b.Run(fmt.Sprintf("Baseline_ServeMux_%s", method), func(b *testing.B) {
			mux := http.NewServeMux()
			mux.HandleFunc(method+" /test", handler)

			req := httptest.NewRequest(method, "/test", nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
			}
		})

		b.Run(fmt.Sprintf("Zerohttp_Router_%s", method), func(b *testing.B) {
			router := NewRouter()
			switch method {
			case http.MethodGet:
				router.GET("/test", handler)
			case http.MethodPost:
				router.POST("/test", handler)
			case http.MethodPut:
				router.PUT("/test", handler)
			case http.MethodPatch:
				router.PATCH("/test", handler)
			case http.MethodDelete:
				router.DELETE("/test", handler)
			case http.MethodOptions:
				router.OPTIONS("/test", handler)
			}

			req := httptest.NewRequest(method, "/test", nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkRouter_WildcardPatterns measures routing performance with wildcard
// path patterns (Go 1.22+ "..." syntax).
func BenchmarkRouter_WildcardPatterns(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	b.Run("Baseline_ServeMux", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /files/{path...}", handler)

		req := httptest.NewRequest(http.MethodGet, "/files/docs/readme.txt", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}
	})

	b.Run("Zerohttp_Router", func(b *testing.B) {
		router := NewRouter()
		router.GET("/files/{path...}", handler)

		req := httptest.NewRequest(http.MethodGet, "/files/docs/readme.txt", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("ShallowWildcard_1Segment", func(b *testing.B) {
		router := NewRouter()
		router.GET("/api/{path...}", handler)

		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("DeepWildcard_5Segments", func(b *testing.B) {
		router := NewRouter()
		router.GET("/files/{path...}", handler)

		req := httptest.NewRequest(http.MethodGet, "/files/a/b/c/d/e.txt", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("DeepWildcard_10Segments", func(b *testing.B) {
		router := NewRouter()
		router.GET("/files/{path...}", handler)

		req := httptest.NewRequest(http.MethodGet, "/files/a/b/c/d/e/f/g/h/i/j.txt", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRouter_NestedRouteGroups measures the overhead of deeply nested
// route groups (5+ levels).
func BenchmarkRouter_NestedRouteGroups(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(_ string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	b.Run("NoGroups", func(b *testing.B) {
		router := NewRouter()
		router.GET("/api/v1/users", handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("OneGroup", func(b *testing.B) {
		router := NewRouter()
		router.Group(func(api Router) {
			api.GET("/api/v1/users", handler)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("ThreeGroups", func(b *testing.B) {
		router := NewRouter()
		router.Group(func(api Router) {
			api.Group(func(v1 Router) {
				v1.Group(func(users Router) {
					users.GET("/api/v1/users", handler)
				})
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("FiveGroups", func(b *testing.B) {
		router := NewRouter()
		router.Group(func(a Router) {
			a.Group(func(b Router) {
				b.Group(func(c Router) {
					c.Group(func(d Router) {
						d.Group(func(e Router) {
							e.GET("/api/v1/users", handler)
						})
					})
				})
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("FiveGroups_WithMiddleware", func(b *testing.B) {
		router := NewRouter()
		router.Group(func(a Router) {
			a.Use(middleware("level1"))
			a.Group(func(b Router) {
				b.Use(middleware("level2"))
				b.Group(func(c Router) {
					c.Use(middleware("level3"))
					c.Group(func(d Router) {
						d.Use(middleware("level4"))
						d.Group(func(e Router) {
							e.Use(middleware("level5"))
							e.GET("/api/v1/users", handler)
						})
					})
				})
			})
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}
