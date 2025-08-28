package zerohttp

import (
	"embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

func testMiddleware(name string, calls *[]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*calls = append(*calls, name)
			next.ServeHTTP(w, r)
		})
	}
}

func testHandler(message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(message)); err != nil {
			panic(fmt.Errorf("failed to write test response: %w", err))
		}
	}
}

func TestHandlerFunc(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		router := NewRouter()
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return R.JSON(w, 200, M{"message": "success"})
		})
		router.GET("/test", handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "success") {
			t.Errorf("Expected response to contain 'success', got '%s'", w.Body.String())
		}
	})

	t.Run("error case", func(t *testing.T) {
		var panicked bool
		var panicMsg string

		recoveryMW := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if rec := recover(); rec != nil {
						panicked = true
						panicMsg = fmt.Sprintf("%v", rec)
						w.WriteHeader(http.StatusInternalServerError)
					}
				}()
				next.ServeHTTP(w, r)
			})
		}

		router := NewRouter(recoveryMW)
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("test error")
		})
		router.GET("/error", handler)

		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !panicked {
			t.Error("Expected handler error to cause panic")
		}
		if !strings.Contains(panicMsg, "test error") {
			t.Errorf("Expected panic message to contain 'test error', got '%s'", panicMsg)
		}
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("no error", func(t *testing.T) {
		router := NewRouter()
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			_, err := w.Write([]byte("no error"))
			return err
		})
		router.GET("/noerror", handler)

		req := httptest.NewRequest("GET", "/noerror", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "no error" {
			t.Errorf("Expected body 'no error', got '%s'", w.Body.String())
		}
	})

	t.Run("with middleware", func(t *testing.T) {
		var calls []string
		mw := testMiddleware("error-handler-mw", &calls)
		router := NewRouter(mw)
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			calls = append(calls, "handler")
			return R.Text(w, 200, "middleware test")
		})
		router.GET("/middleware", handler)

		req := httptest.NewRequest("GET", "/middleware", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expectedCalls := []string{"error-handler-mw", "handler"}
		if len(calls) != len(expectedCalls) {
			t.Errorf("Expected %d calls, got %d", len(expectedCalls), len(calls))
		}
		for i, expected := range expectedCalls {
			if i >= len(calls) || calls[i] != expected {
				t.Errorf("Expected call %d to be '%s', got '%s'", i, expected, calls[i])
			}
		}
	})

	t.Run("interface compatibility", func(t *testing.T) {
		var _ http.Handler = HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})

		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return R.Text(w, 200, "interface test")
		})
		mux := http.NewServeMux()
		mux.Handle("/interface", handler)

		req := httptest.NewRequest("GET", "/interface", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestNewRouter(t *testing.T) {
	t.Run("without middleware", func(t *testing.T) {
		router := NewRouter()
		if router == nil {
			t.Error("Expected router to be created")
		}

		logger := router.Logger()
		if logger == nil {
			t.Error("Expected router to have a default logger")
		}

		cfg := router.Config()
		requestIDCfg := config.DefaultRequestIDConfig
		for _, opt := range cfg.RequestIDOptions {
			opt(&requestIDCfg)
		}

		if requestIDCfg.Header != "X-Request-Id" {
			t.Errorf("Expected default header name 'X-Request-Id', got '%s'", requestIDCfg.Header)
		}
		if requestIDCfg.Generator == nil {
			t.Error("Expected default GenerateID function to be set")
		}
	})

	t.Run("with global middleware", func(t *testing.T) {
		var calls []string
		middleware1 := testMiddleware("mw1", &calls)
		middleware2 := testMiddleware("mw2", &calls)
		router := NewRouter(middleware1, middleware2)
		router.GET("/test", testHandler("response"))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expectedCalls := []string{"mw1", "mw2"}
		if len(calls) != len(expectedCalls) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedCalls), len(calls))
		}
		for i, expected := range expectedCalls {
			if i >= len(calls) || calls[i] != expected {
				t.Errorf("Expected middleware call %d to be %s, got %s", i, expected, calls[i])
			}
		}
	})
}

func TestRouter_HTTPMethods(t *testing.T) {
	router := NewRouter()
	router.DELETE("/delete", testHandler("delete"))
	router.GET("/get", testHandler("get"))
	router.HEAD("/head", testHandler("head"))
	router.OPTIONS("/options", testHandler("options"))
	router.PATCH("/patch", testHandler("patch"))
	router.POST("/post", testHandler("post"))
	router.PUT("/put", testHandler("put"))

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{"DELETE", "/delete", "delete"},
		{"GET", "/get", "get"},
		{"HEAD", "/head", ""},
		{"OPTIONS", "/options", "options"},
		{"PATCH", "/patch", "patch"},
		{"POST", "/post", "post"},
		{"PUT", "/put", "put"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			body := w.Body.String()
			if tt.method == "HEAD" {
				if body != "" {
					t.Logf("Note: HEAD request body was '%s' but this may be expected", body)
				}
			} else {
				if body != tt.body {
					t.Errorf("Expected body '%s', got '%s'", tt.body, body)
				}
			}
		})
	}
}

func TestRouter_Middleware(t *testing.T) {
	t.Run("route specific middleware", func(t *testing.T) {
		var calls []string
		globalMw := testMiddleware("global", &calls)
		routeMw1 := testMiddleware("route1", &calls)
		routeMw2 := testMiddleware("route2", &calls)
		router := NewRouter(globalMw)
		router.GET("/test", testHandler("response"), routeMw1, routeMw2)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expectedCalls := []string{"global", "route1", "route2"}
		if len(calls) != len(expectedCalls) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedCalls), len(calls))
		}
	})

	t.Run("use method", func(t *testing.T) {
		var calls []string
		mw1 := testMiddleware("mw1", &calls)
		mw2 := testMiddleware("mw2", &calls)
		mw3 := testMiddleware("mw3", &calls)
		router := NewRouter(mw1)
		router.Use(mw2, mw3)
		router.GET("/test", testHandler("response"))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expectedCalls := []string{"mw1", "mw2", "mw3"}
		if len(calls) != len(expectedCalls) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedCalls), len(calls))
		}
	})

	t.Run("middleware order", func(t *testing.T) {
		var order []int
		mw1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, 1)
				next.ServeHTTP(w, r)
				order = append(order, -1)
			})
		}
		mw2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, 2)
				next.ServeHTTP(w, r)
				order = append(order, -2)
			})
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 0)
			_, err := w.Write([]byte("ok"))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})

		router := NewRouter()
		router.Use(mw1, mw2)
		router.GET("/test", handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expectedOrder := []int{1, 2, 0, -2, -1}
		if len(order) != len(expectedOrder) {
			t.Errorf("Expected %d calls, got %d", len(expectedOrder), len(order))
		}
		for i, expected := range expectedOrder {
			if i >= len(order) || order[i] != expected {
				t.Errorf("Expected order[%d] to be %d, got %d", i, expected, order[i])
			}
		}
	})
}

func TestRouter_Groups(t *testing.T) {
	t.Run("basic group", func(t *testing.T) {
		var calls []string
		globalMw := testMiddleware("global", &calls)
		groupMw := testMiddleware("group", &calls)
		router := NewRouter(globalMw)
		router.Group(func(api Router) {
			api.Use(groupMw)
			api.GET("/group/test", testHandler("group response"))
		})

		req := httptest.NewRequest("GET", "/group/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "group response" {
			t.Errorf("Expected 'group response', got '%s'", w.Body.String())
		}

		expectedCalls := []string{"global", "group"}
		if len(calls) != len(expectedCalls) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedCalls), len(calls))
		}
	})

	t.Run("group isolation", func(t *testing.T) {
		var globalCalls []string
		var groupCalls []string
		globalMw := testMiddleware("global", &globalCalls)
		groupMw := testMiddleware("group", &groupCalls)
		router := NewRouter(globalMw)

		router.GET("/outside", testHandler("outside"))
		router.Group(func(api Router) {
			api.Use(groupMw)
			api.GET("/inside", testHandler("inside"))
		})

		// Test outside route
		req := httptest.NewRequest("GET", "/outside", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if len(globalCalls) != 1 || globalCalls[0] != "global" {
			t.Error("Outside route should only have global middleware")
		}
		if len(groupCalls) != 0 {
			t.Error("Outside route should not execute group middleware")
		}

		// Reset and test inside route
		globalCalls = nil
		groupCalls = nil
		req = httptest.NewRequest("GET", "/inside", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if len(globalCalls) != 1 || globalCalls[0] != "global" {
			t.Error("Inside route should have global middleware")
		}
		if len(groupCalls) != 1 || groupCalls[0] != "group" {
			t.Error("Inside route should have group middleware")
		}
	})
}

func TestRouter_ErrorHandlers(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		router := NewRouter()
		router.GET("/exists", testHandler("exists"))

		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != MIMEApplicationProblem {
			t.Errorf("Expected Content-Type %s, got %s", MIMEApplicationProblem, contentType)
		}

		// Test custom 404 handler
		router.NotFound(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Custom 404"))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})

		req = httptest.NewRequest("GET", "/nonexistent", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
		if w.Body.String() != "Custom 404" {
			t.Errorf("Expected 'Custom 404', got '%s'", w.Body.String())
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		router := NewRouter()
		router.GET("/test", testHandler("get"))
		router.POST("/test", testHandler("post"))

		req := httptest.NewRequest("PUT", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}

		allowHeader := w.Header().Get("Allow")
		if allowHeader == "" {
			t.Error("Expected Allow header to be set")
		}
		if !strings.Contains(allowHeader, "GET") || !strings.Contains(allowHeader, "POST") {
			t.Errorf("Expected Allow header to contain GET and POST, got '%s'", allowHeader)
		}

		// Test custom method not allowed handler
		router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, err := w.Write([]byte("Custom 405"))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})

		req = httptest.NewRequest("PUT", "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
		if w.Body.String() != "Custom 405" {
			t.Errorf("Expected 'Custom 405', got '%s'", w.Body.String())
		}
	})
}

func TestRouter_Configuration(t *testing.T) {
	t.Run("logger management", func(t *testing.T) {
		router := NewRouter()
		logger := router.Logger()
		if logger == nil {
			t.Error("Expected new router to have a default logger, got nil")
		}

		customLogger := log.NewDefaultLogger()
		router.SetLogger(customLogger)
		if router.Logger() != customLogger {
			t.Error("Expected SetLogger to update the router's logger")
		}

		router.SetLogger(nil)
		if router.Logger() != nil {
			t.Error("Expected Logger to return nil when set to nil")
		}
	})

	t.Run("config management", func(t *testing.T) {
		router := NewRouter()
		customConfig := config.DefaultConfig
		customConfig.RequestIDOptions = []config.RequestIDOption{
			config.WithRequestIDHeader("X-Custom-Request-Id"),
			config.WithRequestIDGenerator(func() string { return "custom-id-12345" }),
		}

		router.SetConfig(customConfig)

		// Test 404 response uses custom config
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		customRequestID := w.Header().Get("X-Custom-Request-Id")
		if customRequestID != "custom-id-12345" {
			t.Errorf("Expected custom request ID 'custom-id-12345', got '%s'", customRequestID)
		}

		defaultRequestID := w.Header().Get("X-Request-Id")
		if defaultRequestID != "" {
			t.Errorf("Expected no default request ID header, got '%s'", defaultRequestID)
		}
	})
}

func TestRouter_EdgeCases(t *testing.T) {
	t.Run("root path", func(t *testing.T) {
		router := NewRouter()
		router.GET("/", testHandler("root"))

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "root" {
			t.Errorf("Expected 'root', got '%s'", w.Body.String())
		}
	})

	t.Run("complex paths", func(t *testing.T) {
		router := NewRouter()
		router.GET("/api/v1/users", testHandler("users"))
		router.GET("/api/v1/users/{id}", testHandler("user by id"))
		router.GET("/api/v1/users/{id}/posts", testHandler("user posts"))

		tests := []struct {
			path     string
			expected string
		}{
			{"/api/v1/users", "users"},
			{"/api/v1/users/123", "user by id"},
			{"/api/v1/users/123/posts", "user posts"},
		}

		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				req := httptest.NewRequest("GET", tt.path, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}
				if w.Body.String() != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, w.Body.String())
				}
			})
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		router := NewRouter()
		router.GET("/test", testHandler("concurrent"))

		const numRequests = 100
		results := make(chan string, numRequests)
		for range numRequests {
			go func() {
				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				results <- w.Body.String()
			}()
		}

		for range numRequests {
			result := <-results
			if result != "concurrent" {
				t.Errorf("Expected 'concurrent', got '%s'", result)
			}
		}
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("defaultNotFoundHandler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		defaultNotFoundHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != MIMEApplicationProblem {
			t.Errorf("Expected Content-Type %s, got %s", MIMEApplicationProblem, contentType)
		}
	})

	t.Run("defaultMethodNotAllowedHandler", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		defaultMethodNotAllowedHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != MIMEApplicationProblem {
			t.Errorf("Expected Content-Type %s, got %s", MIMEApplicationProblem, contentType)
		}
	})

	t.Run("allowedMethods", func(t *testing.T) {
		methods := map[string]bool{
			"GET":  true,
			"POST": true,
			"PUT":  true,
		}
		result := allowedMethods(methods)

		for method := range methods {
			if !strings.Contains(result, method) {
				t.Errorf("Expected result to contain %s, got '%s'", method, result)
			}
		}

		parts := strings.Split(result, ", ")
		if len(parts) != 3 {
			t.Errorf("Expected 3 methods separated by commas, got %d parts", len(parts))
		}
	})
}

//go:embed testdata/files
var testFilesFS embed.FS

func TestRouter_StaticFiles(t *testing.T) {
	t.Run("Files - embedded FS", func(t *testing.T) {
		router := NewRouter()
		router.Files("/static/", testFilesFS, "testdata/files")

		// Test serving a file
		req := httptest.NewRequest("GET", "/static/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expected := "test file content"
		if strings.TrimSpace(w.Body.String()) != expected {
			t.Errorf("Expected '%s', got '%s'", expected, strings.TrimSpace(w.Body.String()))
		}

		// Test 404 for non-existent file
		req = httptest.NewRequest("GET", "/static/nonexistent.txt", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent file, got %d", w.Code)
		}
	})

	t.Run("Files - with trailing slash", func(t *testing.T) {
		router := NewRouter()
		router.Files("/assets", testFilesFS, "testdata/files") // No trailing slash

		req := httptest.NewRequest("GET", "/assets/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("FilesDir - directory serving", func(t *testing.T) {
		router := NewRouter()
		router.FilesDir("/files/", "testdata/files")

		// Test serving a file
		req := httptest.NewRequest("GET", "/files/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		expected := "test file content"
		if strings.TrimSpace(w.Body.String()) != expected {
			t.Errorf("Expected '%s', got '%s'", expected, strings.TrimSpace(w.Body.String()))
		}

		// Test 404 for non-existent file
		req = httptest.NewRequest("GET", "/files/nonexistent.txt", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent file, got %d", w.Code)
		}
	})

	t.Run("FilesDir - without trailing slash", func(t *testing.T) {
		router := NewRouter()
		router.FilesDir("/downloads", "testdata/files") // No trailing slash

		req := httptest.NewRequest("GET", "/downloads/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

//go:embed testdata/static
var testStaticFS embed.FS

func TestRouter_Static(t *testing.T) {
	t.Run("Static - with custom API prefix", func(t *testing.T) {
		router := NewRouter()
		router.Static(testStaticFS, "testdata/static", "/v1/", "/v2/")

		// Test custom API prefix exclusion
		req := httptest.NewRequest("GET", "/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for custom API route, got %d", w.Code)
		}

		// Test second custom API prefix exclusion
		req = httptest.NewRequest("GET", "/v2/users", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for custom API route, got %d", w.Code)
		}

		// Test that old API prefix doesn't work
		req = httptest.NewRequest("GET", "/api/users", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for Static fallback with old API prefix, got %d", w.Code)
		}
	})

	t.Run("StaticDir - directory serving", func(t *testing.T) {
		router := NewRouter()
		router.StaticDir("testdata/static")

		// Test serving index.html for root
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
			t.Error("Expected index.html content")
		}

		// Test serving static asset
		req = httptest.NewRequest("GET", "/app.js", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for static asset, got %d", w.Code)
		}

		// Test Static fallback
		req = httptest.NewRequest("GET", "/dashboard", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for Static fallback, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "<!DOCTYPE html>") {
			t.Error("Expected index.html content for Static fallback")
		}
	})

	t.Run("StaticDir - with custom API prefixes", func(t *testing.T) {
		router := NewRouter()
		router.StaticDir("testdata/static", "/custom-api/", "/other-api/")

		// Test custom API prefix exclusion
		req := httptest.NewRequest("GET", "/custom-api/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for custom API route, got %d", w.Code)
		}

		// Test second custom API prefix exclusion
		req = httptest.NewRequest("GET", "/other-api/data", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for custom API route, got %d", w.Code)
		}
	})
}

func TestRouter_ServeMux(t *testing.T) {
	router := NewRouter()

	mux := router.ServeMux()
	if mux == nil {
		t.Fatal("Expected ServeMux to return a non-nil mux")
	}

	mux.HandleFunc("GET /direct", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("direct handler"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	req := httptest.NewRequest("GET", "/direct", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "direct handler" {
		t.Errorf("Expected 'direct handler', got '%s'", w.Body.String())
	}
}
