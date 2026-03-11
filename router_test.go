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
	"github.com/alexferl/zerohttp/zhtest"
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

// failWriteRecorder is a ResponseRecorder that fails on Write to simulate
// JSON encoding failures
type failWriteRecorder struct {
	*httptest.ResponseRecorder
	failWrite bool
}

func (f *failWriteRecorder) Write(p []byte) (int, error) {
	if f.failWrite {
		return 0, fmt.Errorf("simulated write failure")
	}
	return f.ResponseRecorder.Write(p)
}

func TestHandlerFunc(t *testing.T) {
	t.Run("success case", func(t *testing.T) {
		router := NewRouter()
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return R.JSON(w, http.StatusOK, M{"message": "success"})
		})
		router.GET("/test", handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK).BodyContains("success")
	})

	t.Run("error case", func(t *testing.T) {
		router := NewRouter()
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("test error")
		})
		router.GET("/error", handler)

		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Errors are now handled directly without panic
		zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)
	})

	t.Run("no error", func(t *testing.T) {
		router := NewRouter()
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			_, err := w.Write([]byte("no error"))
			return err
		})
		router.GET("/noerror", handler)

		req := httptest.NewRequest(http.MethodGet, "/noerror", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Body("no error")
	})

	t.Run("with middleware", func(t *testing.T) {
		var calls []string
		mw := testMiddleware("error-handler-mw", &calls)
		router := NewRouter(mw)
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			calls = append(calls, "handler")
			return R.Text(w, http.StatusOK, "middleware test")
		})
		router.GET("/middleware", handler)

		req := httptest.NewRequest(http.MethodGet, "/middleware", nil)
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

	t.Run("HEAD request discards body writes", func(t *testing.T) {
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			// Try to write body - should be discarded for HEAD requests
			// but headers should still be set
			return R.Text(w, http.StatusOK, "this should not appear in HEAD response")
		})

		router := NewRouter()
		router.GET("/", handler)

		// Make a HEAD request
		req := httptest.NewRequest(http.MethodHead, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Header(HeaderContentType, MIMETextPlainCharset).
			BodyEmpty()
	})

	t.Run("headResponseWriter Unwrap", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		hrw := &headResponseWriter{ResponseWriter: recorder}

		// Test that Unwrap returns the underlying ResponseWriter
		unwrapped, ok := hrw.Unwrap().(*httptest.ResponseRecorder)
		if !ok {
			t.Error("Unwrap did not return the underlying ResponseRecorder")
		}
		if unwrapped != recorder {
			t.Error("Unwrap returned a different ResponseWriter")
		}
	})

	t.Run("interface compatibility", func(t *testing.T) {
		var _ http.Handler = HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})

		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return R.Text(w, http.StatusOK, "interface test")
		})
		mux := http.NewServeMux()
		mux.Handle("/interface", handler)

		req := httptest.NewRequest(http.MethodGet, "/interface", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	})
}

func TestJSONEncodingErrorLogged(t *testing.T) {
	testCases := []struct {
		name           string
		errorType      string
		handlerError   error
		expectedStatus int
	}{
		{
			name:           "validation error encoding failure",
			errorType:      "validation",
			handlerError:   &testValidationError{errors: map[string][]string{"field": {"invalid"}}},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "binding error encoding failure",
			errorType:      "binding",
			handlerError:   &BindError{Err: fmt.Errorf("invalid JSON")},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "generic error encoding failure",
			errorType:      "generic",
			handlerError:   fmt.Errorf("some internal error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test logger that captures log output
			testLogger := log.NewDefaultLogger()

			// Save original logger and restore after test
			originalLogger := log.GetGlobalLogger()
			log.SetGlobalLogger(testLogger)
			defer log.SetGlobalLogger(originalLogger)

			// Create a response recorder that fails on write
			recorder := &failWriteRecorder{
				ResponseRecorder: httptest.NewRecorder(),
				failWrite:        true,
			}

			// Call handleHandlerError directly
			handleHandlerError(recorder, tc.handlerError)

			// Verify the status was written before the write failure
			if recorder.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}

			// Note: We can't easily capture the log output from DefaultLogger
			// since it writes to stdout, but we verify the code path doesn't panic
			// and the status header was written. In a real scenario, the error
			// would be logged to stdout.
		})
	}
}

// testValidationError is a test implementation of ValidationErrorer
type testValidationError struct {
	errors map[string][]string
}

func (e *testValidationError) Error() string {
	return "validation failed"
}

func (e *testValidationError) ValidationErrors() map[string][]string {
	return e.errors
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
		if cfg.RequestID.Header != "X-Request-Id" {
			t.Errorf("Expected default header name 'X-Request-Id', got '%s'", cfg.RequestID.Header)
		}
		if cfg.RequestID.Generator == nil {
			t.Error("Expected default GenerateID function to be set")
		}
	})

	t.Run("with global middleware", func(t *testing.T) {
		var calls []string
		middleware1 := testMiddleware("mw1", &calls)
		middleware2 := testMiddleware("mw2", &calls)
		router := NewRouter(middleware1, middleware2)
		router.GET("/test", testHandler("response"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)

		expectedCalls := []string{"mw1", "mw2"}
		if len(calls) != len(expectedCalls) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedCalls), len(calls))
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
	router.CONNECT("/connect", testHandler("connect"))

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodConnect, "/connect", "connect"},
		{http.MethodDelete, "/delete", "delete"},
		{http.MethodGet, "/get", "get"},
		{http.MethodHead, "/head", ""},
		{http.MethodOptions, "/options", "options"},
		{http.MethodPatch, "/patch", "patch"},
		{http.MethodPost, "/post", "post"},
		{http.MethodPut, "/put", "put"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.method == http.MethodHead {
				zhtest.AssertWith(t, w).Status(http.StatusOK)
			} else {
				zhtest.AssertWith(t, w).
					Status(http.StatusOK).
					Body(tt.body)
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

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)

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

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/group/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Body("group response")

		expectedCalls := []string{"global", "group"}
		if len(calls) != len(expectedCalls) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedCalls), len(calls))
		}
		for i, expected := range expectedCalls {
			if i >= len(calls) || calls[i] != expected {
				t.Errorf("Expected middleware call %d to be %s, got %s", i, expected, calls[i])
			}
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
		req := httptest.NewRequest(http.MethodGet, "/outside", nil)
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
		req = httptest.NewRequest(http.MethodGet, "/inside", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Header(HeaderContentType, MIMETextPlainCharset)

		router.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Custom 404"))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req = httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Body("Custom 404")
	})

	t.Run("method not allowed", func(t *testing.T) {
		router := NewRouter()
		router.GET("/test", testHandler("get"))
		router.POST("/test", testHandler("post"))

		req := httptest.NewRequest(http.MethodPut, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusMethodNotAllowed)

		allowHeader := w.Header().Get("Allow")
		if allowHeader == "" {
			t.Error("Expected Allow header to be set")
		}
		if !strings.Contains(allowHeader, http.MethodGet) || !strings.Contains(allowHeader, http.MethodPost) {
			t.Errorf("Expected Allow header to contain GET and POST, got '%s'", allowHeader)
		}

		router.MethodNotAllowed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, err := w.Write([]byte("Custom 405"))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req = httptest.NewRequest(http.MethodPut, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusMethodNotAllowed).
			Body("Custom 405")
	})

	// Test fallback path when ProblemDetail fails
	t.Run("default not found handler fallback", func(t *testing.T) {
		// Use a response writer that fails when writing the JSON body
		w := &failWriteRecorder{ResponseRecorder: httptest.NewRecorder(), failWrite: true}
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)

		// Now call the handler - should trigger fallback path
		defaultNotFoundHandler.ServeHTTP(w, req)

		// Content-Type should be text/plain; charset=utf-8
		contentType := w.Header().Get("Content-Type")
		if contentType != MIMETextPlainCharset {
			t.Errorf("expected Content-Type %q, got %q", MIMETextPlainCharset, contentType)
		}
	})

	t.Run("default method not allowed handler fallback", func(t *testing.T) {
		w := &failWriteRecorder{ResponseRecorder: httptest.NewRecorder(), failWrite: true}
		req := httptest.NewRequest(http.MethodPut, "/test", nil)

		defaultMethodNotAllowedHandler.ServeHTTP(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != MIMETextPlainCharset {
			t.Errorf("expected Content-Type %q, got %q", MIMETextPlainCharset, contentType)
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
		customConfig.RequestID.Header = "X-Custom-Request-Id"
		customConfig.RequestID.Generator = func() string { return "custom-id-12345" }

		router.SetConfig(customConfig)

		// Test 404 response uses custom config
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Header("X-Custom-Request-Id", "custom-id-12345").
			HeaderNotExists("X-Request-Id")
	})
}

func TestRouter_EdgeCases(t *testing.T) {
	t.Run("root path", func(t *testing.T) {
		router := NewRouter()
		router.GET("/", testHandler("root"))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			Body("root")
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
				req := httptest.NewRequest(http.MethodGet, tt.path, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				zhtest.AssertWith(t, w).
					Status(http.StatusOK).
					Body(tt.expected)
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
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
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
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		defaultNotFoundHandler.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Header(HeaderContentType, MIMETextPlainCharset)
	})

	t.Run("defaultMethodNotAllowedHandler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()
		defaultMethodNotAllowedHandler.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusMethodNotAllowed).
			Header(HeaderContentType, MIMETextPlainCharset)
	})

	t.Run("defaultNotFoundHandler body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		defaultNotFoundHandler.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Body("Requested resource was not found\n")
	})

	t.Run("allowedMethods", func(t *testing.T) {
		methods := map[string]bool{
			http.MethodGet:  true,
			http.MethodPost: true,
			http.MethodPut:  true,
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
		req := httptest.NewRequest(http.MethodGet, "/static/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			BodyContains("test file content")

		// Test 404 for non-existent file
		req = httptest.NewRequest(http.MethodGet, "/static/nonexistent.txt", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound)
	})

	t.Run("Files - with trailing slash", func(t *testing.T) {
		router := NewRouter()
		router.Files("/assets", testFilesFS, "testdata/files") // No trailing slash

		req := httptest.NewRequest(http.MethodGet, "/assets/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	})

	t.Run("FilesDir - directory serving", func(t *testing.T) {
		router := NewRouter()
		router.FilesDir("/files/", "testdata/files")

		// Test serving a file
		req := httptest.NewRequest(http.MethodGet, "/files/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			BodyContains("test file content")

		// Test 404 for non-existent file
		req = httptest.NewRequest(http.MethodGet, "/files/nonexistent.txt", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound)
	})

	t.Run("FilesDir - without trailing slash", func(t *testing.T) {
		router := NewRouter()
		router.FilesDir("/downloads", "testdata/files") // No trailing slash

		req := httptest.NewRequest(http.MethodGet, "/downloads/test.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	})
}

//go:embed testdata/static
var testStaticFS embed.FS

func TestRouter_Static(t *testing.T) {
	t.Run("Static - with fallback and custom API prefix", func(t *testing.T) {
		router := NewRouter()
		router.Static(testStaticFS, "testdata/static", true, "/v1/", "/v2/")

		// Test custom API prefix exclusion
		req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound)

		// Test second custom API prefix exclusion
		req = httptest.NewRequest(http.MethodGet, "/v2/users", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound)

		// Test that old API prefix doesn't work (should fallback to index.html)
		req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	})

	t.Run("Static - without fallback", func(t *testing.T) {
		router := NewRouter()
		router.Static(testStaticFS, "testdata/static", false)

		// Set custom 404 handler
		router.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Custom 404 for missing file"))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		// Test serving existing file (should work)
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)

		// Test missing file (should use custom 404, not fallback to index.html)
		req = httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Body("Custom 404 for missing file")
	})

	t.Run("StaticDir - with fallback", func(t *testing.T) {
		router := NewRouter()
		router.StaticDir("testdata/static", true)

		// Test serving index.html for root
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			BodyContains("<!DOCTYPE html>")

		// Test serving static asset
		req = httptest.NewRequest(http.MethodGet, "/app.js", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK)

		// Test Static fallback (should serve index.html)
		req = httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusOK).
			BodyContains("<!DOCTYPE html>")
	})

	t.Run("StaticDir - without fallback", func(t *testing.T) {
		router := NewRouter()
		router.StaticDir("testdata/static", false)

		// Set custom 404 handler
		router.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Static site 404"))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		// Test missing file (should use custom 404)
		req := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).
			Status(http.StatusNotFound).
			Body("Static site 404")
	})

	t.Run("StaticDir - with custom API prefixes and fallback", func(t *testing.T) {
		router := NewRouter()
		router.StaticDir("testdata/static", true, "/custom-api/", "/other-api/")

		// Test custom API prefix exclusion
		req := httptest.NewRequest(http.MethodGet, "/custom-api/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound)

		// Test second custom API prefix exclusion
		req = httptest.NewRequest(http.MethodGet, "/other-api/data", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound)
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

	req := httptest.NewRequest(http.MethodGet, "/direct", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusOK).
		Body("direct handler")
}

func TestRouter_CONNECT_WebTransport(t *testing.T) {
	t.Run("CONNECT handler registration", func(t *testing.T) {
		router := NewRouter()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		router.CONNECT("/wt", handler)

		req := httptest.NewRequest(http.MethodConnect, "/wt", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("CONNECT handler was not called")
		}

		zhtest.AssertWith(t, w).Status(http.StatusOK)
	})

	t.Run("CONNECT with middleware", func(t *testing.T) {
		router := NewRouter()

		var calls []string
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, "middleware")
				next.ServeHTTP(w, r)
			})
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "handler")
			w.WriteHeader(http.StatusOK)
		})

		router.CONNECT("/wt", handler, mw)

		req := httptest.NewRequest(http.MethodConnect, "/wt", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if len(calls) != 2 || calls[0] != "middleware" || calls[1] != "handler" {
			t.Errorf("Expected [middleware, handler], got %v", calls)
		}
	})

	t.Run("CONNECT route not found", func(t *testing.T) {
		router := NewRouter()

		req := httptest.NewRequest(http.MethodConnect, "/not-registered", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound)
	})

	t.Run("CONNECT WebTransport-like upgrade", func(t *testing.T) {
		router := NewRouter()

		upgradeCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodConnect {
				t.Errorf("Expected CONNECT method, got %s", r.Method)
			}
			upgradeCalled = true
			w.WriteHeader(http.StatusOK)
		})

		router.CONNECT("/wt", handler)

		req := httptest.NewRequest(http.MethodConnect, "/wt", nil)
		req.Header.Set("Upgrade", "webtransport")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if !upgradeCalled {
			t.Error("CONNECT handler was not called for WebTransport upgrade")
		}
	})
}

func TestShouldLogRequest(t *testing.T) {
	t.Run("returns true when Enabled is nil (default)", func(t *testing.T) {
		r := NewRouter().(*defaultRouter)
		r.SetConfig(config.Config{
			RequestLogger: config.RequestLoggerConfig{
				Enabled: nil, // use default
			},
		})

		if !r.shouldLogRequest() {
			t.Error("expected shouldLogRequest to be true when Enabled is nil")
		}
	})

	t.Run("returns true when Enabled is explicitly true", func(t *testing.T) {
		r := NewRouter().(*defaultRouter)
		r.SetConfig(config.Config{
			RequestLogger: config.RequestLoggerConfig{
				Enabled: config.Bool(true),
			},
		})

		if !r.shouldLogRequest() {
			t.Error("expected shouldLogRequest to be true when Enabled is true")
		}
	})

	t.Run("returns false when Enabled is explicitly false", func(t *testing.T) {
		r := NewRouter().(*defaultRouter)
		r.SetConfig(config.Config{
			RequestLogger: config.RequestLoggerConfig{
				Enabled: config.Bool(false),
			},
		})

		if r.shouldLogRequest() {
			t.Error("expected shouldLogRequest to be false when Enabled is false")
		}
	})

	t.Run("returns false when DisableDefaultMiddlewares is true", func(t *testing.T) {
		r := NewRouter().(*defaultRouter)
		r.SetConfig(config.Config{
			DisableDefaultMiddlewares: true,
			RequestLogger: config.RequestLoggerConfig{
				Enabled: config.Bool(true), // even if Enabled is true
			},
		})

		if r.shouldLogRequest() {
			t.Error("expected shouldLogRequest to be false when DisableDefaultMiddlewares is true")
		}
	})

	t.Run("returns true when DisableDefaultMiddlewares is false and Enabled is nil", func(t *testing.T) {
		r := NewRouter().(*defaultRouter)
		r.SetConfig(config.Config{
			DisableDefaultMiddlewares: false,
			RequestLogger: config.RequestLoggerConfig{
				Enabled: nil,
			},
		})

		if !r.shouldLogRequest() {
			t.Error("expected shouldLogRequest to be true when DisableDefaultMiddlewares is false and Enabled is nil")
		}
	})
}
