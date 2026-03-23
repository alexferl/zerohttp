package zerohttp

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/middleware/requestlogger"
)

var (
	// preEncodedNotFoundJSON is the pre-encoded 404 Not Found JSON response
	preEncodedNotFoundJSON []byte
	// preEncodedMethodNotAllowedJSON is the pre-encoded 405 Method Not Allowed JSON response
	preEncodedMethodNotAllowedJSON []byte
)

func init() {
	// Pre-encode JSON responses at init time to avoid repeated encoding overhead
	notFoundProblem := &ProblemDetail{
		Title:  "Not Found",
		Status: http.StatusNotFound,
		Detail: "Requested resource was not found",
	}
	preEncodedNotFoundJSON, _ = json.Marshal(notFoundProblem)

	methodNotAllowedProblem := &ProblemDetail{
		Title:  "Method Not Allowed",
		Status: http.StatusMethodNotAllowed,
		Detail: "HTTP method is not allowed",
	}
	preEncodedMethodNotAllowedJSON, _ = json.Marshal(methodNotAllowedProblem)
}

// HandlerFunc is a handler function that returns an error.
// It implements [http.Handler], allowing it to be used anywhere a standard
// HTTP handler is expected.
//
// Errors are automatically converted to appropriate HTTP responses:
//   - Validation errors return 422 Unprocessable Entity with field details
//   - Binding errors return 400 Bad Request
//   - Request too large returns 413 Payload Too Large
//   - ProblemDetail errors return their specified status code
//   - All other errors return 500 Internal Server Error
//
// Example:
//
//	app.GET("/users/{id}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    id := zh.Param(r, "id")
//	    user, err := db.GetUser(id)
//	    if err != nil {
//	        return err // Returns 500
//	    }
//	    if user == nil {
//	        return zh.NewProblemDetail(http.StatusNotFound, "User not found")
//	    }
//	    return zh.Render.JSON(w, http.StatusOK, user)
//	}))
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP implements http.Handler interface.
// It handles all errors directly; no panic propagation is used.
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// For HEAD requests, wrap the response writer to discard body writes.
	// This prevents HTTP/2 "request body closed" errors when handlers
	// (like templ) try to write during HEAD requests.
	if r.Method == http.MethodHead {
		hrw := &headResponseWriter{ResponseWriter: w}
		w = hrw
		defer func() { _ = hrw.Close() }() // Ensure Content-Length is set after handler completes
	}

	if err := h(w, r); err != nil {
		// Handle all errors directly - no panic propagation
		handleHandlerError(w, err)
	}
}

// handleHandlerError handles all handler errors.
// Returns appropriate HTTP responses for different error types.
func handleHandlerError(w http.ResponseWriter, err error) {
	// Check for validation errors (422)
	var verr ValidationErrorer
	if errors.As(err, &verr) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
		w.WriteHeader(http.StatusUnprocessableEntity)
		response := map[string]any{
			"title":  "Unprocessable Entity",
			"status": http.StatusUnprocessableEntity,
			"detail": "Validation failed",
			"errors": verr.ValidationErrors(),
		}
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			log.GetGlobalLogger().Error("Failed to encode validation error response", log.E(encErr))
		}
		return
	}

	// Check for binding errors (400)
	if IsBindError(err) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]any{
			"title":  "Bad Request",
			"status": http.StatusBadRequest,
			"detail": "Invalid request body",
		}
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			log.GetGlobalLogger().Error("Failed to encode binding error response", log.E(encErr))
		}
		return
	}

	// Check for request body too large errors (413)
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		response := map[string]any{
			"title":  "Payload Too Large",
			"status": http.StatusRequestEntityTooLarge,
			"detail": "Request body exceeds maximum allowed size",
		}
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			log.GetGlobalLogger().Error("Failed to encode payload too large error response", log.E(encErr))
		}
		return
	}

	// For all other errors, return 500 Internal Server Error
	w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
	w.WriteHeader(http.StatusInternalServerError)
	response := map[string]any{
		"title":  "Internal Server Error",
		"status": http.StatusInternalServerError,
		"detail": "An unexpected error occurred",
	}
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		log.GetGlobalLogger().Error("Failed to encode internal server error response", log.E(encErr))
	}
}

// headResponseWriter wraps a ResponseWriter and discards body writes for HEAD requests.
// It buffers the response to determine Content-Length before writing headers.
type headResponseWriter struct {
	http.ResponseWriter
	buf  []byte
	code int
}

// Ensure headResponseWriter implements http.ResponseWriter
var _ http.ResponseWriter = (*headResponseWriter)(nil)

func (h *headResponseWriter) Write(p []byte) (int, error) {
	// Buffer the data instead of discarding immediately
	h.buf = append(h.buf, p...)
	return len(p), nil
}

func (h *headResponseWriter) WriteHeader(code int) {
	h.code = code
	// Don't write headers yet - we'll do it in Close()
}

// Close writes the headers with Content-Length and discards the body.
// This must be called after the handler completes.
func (h *headResponseWriter) Close() error {
	if h.code == 0 {
		h.code = http.StatusOK
	}
	// Set Content-Length based on buffered bytes
	if len(h.buf) > 0 {
		h.Header().Set(httpx.HeaderContentLength, fmt.Sprintf("%d", len(h.buf)))
	}
	h.ResponseWriter.WriteHeader(h.code)
	return nil
}

// Flush implements http.Flusher to support streaming responses like SSE.
func (h *headResponseWriter) Flush() {
	if f, ok := h.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap allows middleware to access the underlying ResponseWriter
func (h *headResponseWriter) Unwrap() http.ResponseWriter {
	return h.ResponseWriter
}

// Router interface defines the contract for HTTP routing operations.
// It provides methods for registering HTTP handlers for specific HTTP methods,
// applying middleware, creating route groups, and customizing error handlers.
type Router interface {
	// DELETE registers a handler for HTTP DELETE requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	DELETE(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// GET registers a handler for HTTP GET requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	GET(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// HEAD registers a handler for HTTP HEAD requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	HEAD(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// OPTIONS registers a handler for HTTP OPTIONS requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	OPTIONS(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// PATCH registers a handler for HTTP PATCH requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	PATCH(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// POST registers a handler for HTTP POST requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	POST(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// PUT registers a handler for HTTP PUT requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	PUT(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// CONNECT registers a handler for HTTP CONNECT requests to the specified path.
	// Additional middleware can be provided that will be applied only to this route.
	// CONNECT is typically used for WebSocket and WebTransport upgrades.
	CONNECT(path string, h http.Handler, mw ...func(http.Handler) http.Handler)

	// Use adds middleware to the router's global middleware chain.
	// Middleware is applied to all routes registered after this call.
	Use(mw ...func(http.Handler) http.Handler)

	// Group creates a new router scope that inherits the current middleware chain.
	// This allows for organizing routes and applying middleware to specific groups.
	Group(fn func(Router))

	// NotFound sets a custom handler for 404 Not Found responses.
	// If not set, a default handler that returns a problem detail response is used.
	NotFound(h http.Handler)

	// MethodNotAllowed sets a custom handler for 405 Method Not Allowed responses.
	// If not set, a default handler that returns a problem detail response is used.
	MethodNotAllowed(h http.Handler)

	// Files serves static files from embedded FS at the specified prefix.
	// The prefix is stripped from URLs before looking up files in the embedFS.
	Files(prefix string, embedFS embed.FS, dir string)

	// FilesDir serves static files from a directory at the specified prefix.
	// The prefix is stripped from URLs before looking up files in the directory.
	FilesDir(prefix, dir string)

	// Static serves a static web application from embedded FS with configurable fallback behavior.
	// If fallback is true, falls back to index.html for non-existent files (SPA behavior).
	// If fallback is false, uses the custom NotFound handler for missing files.
	// Requests matching apiPrefix patterns return 404 regardless.
	Static(embedFS embed.FS, distDir string, fallback bool, apiPrefix ...string)

	// StaticDir serves a static web application from a directory with configurable fallback behavior.
	// If fallback is true, falls back to index.html for non-existent files (SPA behavior).
	// If fallback is false, uses the custom NotFound handler for missing files.
	// Requests matching apiPrefix patterns return 404 regardless.
	StaticDir(dir string, fallback bool, apiPrefix ...string)

	// ServeMux returns the underlying http.ServeMux for advanced usage or integration.
	ServeMux() *http.ServeMux

	// ServeHTTP implements the http.Handler interface, making the router compatible
	// with Go's standard HTTP server and middleware ecosystem.
	ServeHTTP(w http.ResponseWriter, req *http.Request)

	// Logger returns the logger instance used by the router for logging
	// requests, errors, and other router-specific events.
	Logger() log.Logger

	// SetLogger configures the logger instance that the router should use
	// for logging operations. This allows for custom logger configuration.
	SetLogger(logger log.Logger)

	// Config returns the current configuration used by the router.
	// This configuration controls various aspects of router behavior
	// including middleware settings and error response handling.
	Config() Config

	// SetConfig updates the router's configuration. This affects how
	// the router handles various behaviors including middleware settings
	// and error response processing.
	//
	// Note: Changing the configuration affects both regular routes
	// and 404/405 error responses.
	SetConfig(config Config)
}

// Ensure defaultRouter implements Router
var _ Router = (*defaultRouter)(nil)

// defaultRouter is the concrete implementation of the Router interface.
// It wraps Go's standard http.ServeMux and adds method-specific routing,
// middleware support, and proper HTTP status code handling.
type defaultRouter struct {
	// mux is the underlying HTTP multiplexer that handles request routing
	mux *http.ServeMux

	// chain contains the middleware functions that will be applied to all routes
	chain []func(http.Handler) http.Handler

	// handlerMu protects notFoundHandler and methodNotAllowedHandler.
	// These handlers can be changed at runtime via NotFound() and MethodNotAllowed().
	handlerMu sync.RWMutex

	// notFoundHandler is called when no route matches the request path
	notFoundHandler http.Handler

	// methodNotAllowedHandler is called when a path exists but the HTTP method is not allowed
	methodNotAllowedHandler http.Handler

	// routesMu protects registeredRoutes. Uses pointer so groups share the same mutex.
	routesMu *sync.RWMutex

	// registeredRoutes tracks which HTTP methods are registered for each path
	// This is used to distinguish between 404 Not Found and 405 Method Not Allowed
	registeredRoutes map[string]map[string]bool // path -> method -> bool

	// logger is the structured logger used by the server and its middleware
	// for recording HTTP requests, errors, and server lifecycle events.
	logger log.Logger

	// config holds the complete configuration for the router including
	// middleware settings and behavioral options. This configuration
	// affects how routes and error responses are handled.
	config Config

	// finalizeOnce ensures the router is finalized exactly once, even with concurrent access.
	// The finalize operation registers the catch-all handler for 404/405 responses.
	finalizeOnce sync.Once
}

// NewRouter creates a new router instance with optional global middleware.
// The middleware provided here will be applied to all routes registered on this router.
//
// Example:
//
//	router := NewRouter(loggingMiddleware, authMiddleware)
func NewRouter(mw ...func(http.Handler) http.Handler) Router {
	cfg := DefaultConfig
	logger := log.NewDefaultLogger()

	// Initialize the package-level logger for error handling
	log.SetGlobalLogger(logger)

	r := &defaultRouter{
		mux:                     &http.ServeMux{},
		chain:                   mw,
		notFoundHandler:         defaultNotFoundHandler,
		methodNotAllowedHandler: defaultMethodNotAllowedHandler,
		routesMu:                &sync.RWMutex{},
		registeredRoutes:        make(map[string]map[string]bool),
		logger:                  logger,
		config:                  cfg,
	}
	return r
}

// Use adds middleware functions to the router's global middleware chain.
// These middleware functions will be applied to all routes registered after this call.
// Middleware is executed in the order it was added (first added, first executed).
//
// Example:
//
//	router.Use(loggingMiddleware, corsMiddleware)
func (r *defaultRouter) Use(mw ...func(http.Handler) http.Handler) {
	r.chain = append(r.chain, mw...)
}

// Group creates a new router scope that inherits the current middleware chain.
// This allows for organizing related routes and applying middleware to specific groups.
// Changes to middleware within the group do not affect the parent router.
//
// Example:
//
//	router.Group(func(api Router) {
//	    api.Use(authMiddleware) // Only applies to routes in this group
//	    api.GET("/users", getUsersHandler)
//	    api.POST("/users", createUserHandler)
//	})
func (r *defaultRouter) Group(fn func(Router)) {
	r.handlerMu.RLock()
	notFoundHandler := r.notFoundHandler
	methodNotAllowedHandler := r.methodNotAllowedHandler
	r.handlerMu.RUnlock()

	groupRouter := &defaultRouter{
		mux:                     r.mux,
		chain:                   slices.Clone(r.chain), // Clone to avoid affecting parent
		notFoundHandler:         notFoundHandler,
		methodNotAllowedHandler: methodNotAllowedHandler,
		routesMu:                r.routesMu,         // Share mutex with parent
		registeredRoutes:        r.registeredRoutes, // Share map with parent
		logger:                  r.logger,
		config:                  r.config,
	}
	fn(groupRouter)
}

// DELETE registers a handler for HTTP DELETE requests to the specified path.
// Additional route-specific middleware can be provided.
func (r *defaultRouter) DELETE(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodDelete, path, h, mw)
}

// GET registers a handler for HTTP GET requests to the specified path.
// Additional route-specific middleware can be provided.
func (r *defaultRouter) GET(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodGet, path, h, mw)
}

// HEAD registers a handler for HTTP HEAD requests to the specified path.
// Additional route-specific middleware can be provided.
func (r *defaultRouter) HEAD(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodHead, path, h, mw)
}

// OPTIONS registers a handler for HTTP OPTIONS requests to the specified path.
// Additional route-specific middleware can be provided.
func (r *defaultRouter) OPTIONS(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodOptions, path, h, mw)
}

// PATCH registers a handler for HTTP PATCH requests to the specified path.
// Additional route-specific middleware can be provided.
func (r *defaultRouter) PATCH(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodPatch, path, h, mw)
}

// POST registers a handler for HTTP POST requests to the specified path.
// Additional route-specific middleware can be provided.
func (r *defaultRouter) POST(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodPost, path, h, mw)
}

// PUT registers a handler for HTTP PUT requests to the specified path.
// Additional route-specific middleware can be provided.
func (r *defaultRouter) PUT(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodPut, path, h, mw)
}

// CONNECT registers a handler for HTTP CONNECT requests to the specified path.
// Additional route-specific middleware can be provided.
// CONNECT is typically used for WebSocket and WebTransport upgrades.
func (r *defaultRouter) CONNECT(path string, h http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodConnect, path, h, mw)
}

// NotFound sets a custom handler for 404 Not Found responses.
// This handler will be called when no registered route matches the request path.
//
// Example:
//
//	router.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    http.Error(w, "Custom 404 message", http.StatusNotFound)
//	}))
func (r *defaultRouter) NotFound(h http.Handler) {
	r.handlerMu.Lock()
	defer r.handlerMu.Unlock()
	r.notFoundHandler = h
}

// MethodNotAllowed sets a custom handler for 405 Method Not Allowed responses.
// This handler will be called when a path exists but the HTTP method is not registered for it.
// The handler should check the "Allow" header to see which methods are allowed.
//
// Example:
//
//	router.MethodNotAllowed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    allow := w.Header().Get(httpx.HeaderAllow)
//	    http.Error(w, fmt.Sprintf("Method not allowed. Allowed: %s", allow), http.StatusMethodNotAllowed)
//	}))
func (r *defaultRouter) MethodNotAllowed(h http.Handler) {
	r.handlerMu.Lock()
	defer r.handlerMu.Unlock()
	r.methodNotAllowedHandler = h
}

// Files serves static files from embedded FS at the specified prefix.
func (r *defaultRouter) Files(prefix string, embedFS embed.FS, dir string) {
	subFS, err := fs.Sub(embedFS, dir)
	if err != nil {
		panic(fmt.Errorf("failed to create sub-filesystem: %w", err))
	}

	handler := http.StripPrefix(prefix, http.FileServer(http.FS(subFS)))

	// Ensure prefix ends with slash for subtree matching
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	r.mux.Handle("GET "+prefix, r.wrap(handler, nil))
}

// FilesDir serves static files from a directory at the specified prefix.
func (r *defaultRouter) FilesDir(prefix, dir string) {
	handler := http.StripPrefix(prefix, http.FileServer(http.Dir(dir)))

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	r.mux.Handle("GET "+prefix, r.wrap(handler, nil))
}

// checkAndMarkRoot atomically verifies that GET / is not yet claimed
// and claims it for Static/StaticDir. Panics with the caller's name on conflict.
func (r *defaultRouter) checkAndMarkRoot(caller string) {
	r.routesMu.Lock()
	defer r.routesMu.Unlock()

	if r.registeredRoutes["/"] != nil && r.registeredRoutes["/"][http.MethodGet] {
		panic(fmt.Sprintf("zerohttp: %s conflicts with an existing GET / route", caller))
	}
	if r.registeredRoutes["/"] == nil {
		r.registeredRoutes["/"] = make(map[string]bool)
	}
	r.registeredRoutes["/"][http.MethodGet] = true
}

// Static serves a static web application from embedded FS with fallback to index.html.
func (r *defaultRouter) Static(embedFS embed.FS, distDir string, fallback bool, apiPrefix ...string) {
	// Validate filesystem first before claiming root to avoid blocking recovery
	subFS, err := fs.Sub(embedFS, distDir)
	if err != nil {
		panic(fmt.Errorf("failed to create sub-filesystem: %w", err))
	}

	r.checkAndMarkRoot("Static()")

	handler := r.createStaticHandler(subFS, fallback, apiPrefix)

	r.mux.Handle("GET /{$}", r.wrap(handler, nil))
	r.mux.Handle("GET /{path...}", r.wrap(handler, nil))
}

// StaticDir serves a static web application from a directory with fallback to index.html.
func (r *defaultRouter) StaticDir(dir string, fallback bool, apiPrefix ...string) {
	// Note: os.DirFS does not validate the path at construction time.
	// Unlike fs.Sub in Static(), validation is deferred to Open calls.
	filesystem := os.DirFS(dir)

	r.checkAndMarkRoot("StaticDir()")

	handler := r.createStaticHandler(filesystem, fallback, apiPrefix)

	r.mux.Handle("GET /{$}", r.wrap(handler, nil))
	r.mux.Handle("GET /{path...}", r.wrap(handler, nil))
}

// statusCapture wraps http.ResponseWriter to capture the status code.
// Used by static file handler to log actual response status instead of hardcoded 200.
type statusCapture struct {
	http.ResponseWriter
	status int
}

func (s *statusCapture) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusCapture) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *defaultRouter) createStaticHandler(filesystem fs.FS, fallback bool, apiPrefixes []string) http.Handler {
	// Capture config values at handler creation time to avoid data races.
	// notFoundHandler is protected by handlerMu and accessed with locking.
	requestIDHeader := r.config.RequestID.Header
	requestIDGenerator := r.config.RequestID.Generator
	requestLoggerConfig := r.config.RequestLogger
	logger := r.logger

	fileServer := http.FileServer(http.FS(filesystem))

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		requestID := req.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = requestIDGenerator()
		}
		w.Header().Set(requestIDHeader, requestID)

		r.handlerMu.RLock()
		notFoundHandler := r.notFoundHandler
		r.handlerMu.RUnlock()

		// Security: reject any request whose raw path contains ".." before
		// it can be resolved away by cleaning. fs.FS also enforces this at
		// Open time via fs.ValidPath, but blocking early allows accurate logging.
		if strings.Contains(req.URL.Path, "..") {
			logger.Warn("Path traversal attempt blocked", log.F("path", req.URL.Path))
			notFoundHandler.ServeHTTP(w, req)
			requestlogger.Log(logger, requestLoggerConfig, nil, req, http.StatusNotFound, time.Since(start), "", "")
			return
		}

		cleanPath := path.Clean(req.URL.Path)

		// Skip API routes - return 404
		for _, prefix := range apiPrefixes {
			if strings.HasPrefix(cleanPath, prefix) {
				notFoundHandler.ServeHTTP(w, req)
				requestlogger.Log(logger, requestLoggerConfig, nil, req, http.StatusNotFound, time.Since(start), "", "")
				return
			}
		}

		// Check if file exists and is not a directory
		// Close immediately after stat - we only need to verify existence
		if file, err := filesystem.Open(strings.TrimPrefix(cleanPath, "/")); err == nil {
			stat, statErr := file.Stat()
			_ = file.Close() // Close immediately - http.FileServer will open it again
			if statErr == nil && !stat.IsDir() {
				rec := &statusCapture{ResponseWriter: w, status: http.StatusOK}
				fileServer.ServeHTTP(rec, req)
				requestlogger.Log(logger, requestLoggerConfig, nil, req, rec.status, time.Since(start), "", "")
				return
			}
		}

		if fallback {
			// Preserve original path for accurate logging and deferred middleware
			originalPath := req.URL.Path
			req.URL.Path = "/"
			defer func() { req.URL.Path = originalPath }() // Safety net for panics upstream
			rec := &statusCapture{ResponseWriter: w, status: http.StatusOK}
			fileServer.ServeHTTP(rec, req)
			req.URL.Path = originalPath // Restore NOW, before LogRequest reads req.URL.Path
			requestlogger.Log(logger, requestLoggerConfig, nil, req, rec.status, time.Since(start), "", "")
		} else {
			notFoundHandler.ServeHTTP(w, req)
			requestlogger.Log(logger, requestLoggerConfig, nil, req, http.StatusNotFound, time.Since(start), "", "")
		}
	})
}

// ServeMux returns the underlying http.ServeMux instance.
// This can be useful for advanced integration scenarios or when you need
// to access ServeMux-specific functionality.
func (r *defaultRouter) ServeMux() *http.ServeMux {
	return r.mux
}

// ServeHTTP implements the http.Handler interface, making the router compatible
// with Go's standard HTTP server. This is the entry point for all HTTP requests.
func (r *defaultRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Auto-finalize on first use - safe for concurrent access
	r.finalizeOnce.Do(func() {
		r.mux.Handle("/", r.wrap(r.catchAllHandler(), nil))
	})
	r.mux.ServeHTTP(w, req)
}

// Logger returns the logger instance used by the router for logging
// requests, errors, and other router-specific events.
func (r *defaultRouter) Logger() log.Logger {
	return r.logger
}

// SetLogger configures the logger instance that the router should use
// for logging operations. This allows for custom logger configuration
// and ensures consistent logging across the application.
func (r *defaultRouter) SetLogger(logger log.Logger) {
	r.logger = logger
	// Also update the global logger for error handling
	log.SetGlobalLogger(logger)
}

// Config returns the current configuration used by the router.
// This configuration controls various aspects of router behavior
// including middleware settings and error response handling.
func (r *defaultRouter) Config() Config {
	return r.config
}

// SetConfig updates the router's configuration. This affects how
// the router handles various behaviors including middleware settings
// and error response processing.
//
// Note: Changing the configuration affects both regular routes
// and 404/405 error responses.
func (r *defaultRouter) SetConfig(cfg Config) {
	r.config = cfg
}

// wrap applies middleware to a handler function.
// It combines the router's global middleware chain with route-specific middleware.
// Middleware is applied in reverse order so that the first middleware added
// is the outermost (executed first).
func (r *defaultRouter) wrap(fn http.Handler, mw []func(http.Handler) http.Handler) (out http.Handler) {
	out = fn

	// Combine global and route-specific middleware
	allMiddleware := append(slices.Clone(r.chain), mw...)

	// Reverse the order so first added middleware executes first
	slices.Reverse(allMiddleware)

	// Apply middleware from outermost to innermost
	for _, m := range allMiddleware {
		out = m(out)
	}
	return
}

// handle is the internal method that registers a handler for a specific HTTP method and path.
// It tracks registered routes for proper 404/405 handling and registers the handler with ServeMux.
func (r *defaultRouter) handle(method, path string, fn http.Handler, mw []func(http.Handler) http.Handler) {
	// Track the route and method for 404/405 determination
	r.routesMu.Lock()
	if r.registeredRoutes[path] == nil {
		r.registeredRoutes[path] = make(map[string]bool)
	}
	// Detect duplicate route registration before overwriting
	if r.registeredRoutes[path][method] {
		r.routesMu.Unlock()
		panic(fmt.Sprintf("zerohttp: route %s %s already registered", method, path))
	}
	r.registeredRoutes[path][method] = true
	r.routesMu.Unlock()

	// Special handling for root path to prevent catch-all behavior
	// The {$} pattern ensures exact match for the root path
	if path == "/" {
		r.mux.Handle(method+" /{$}", r.wrap(fn, mw))
	} else {
		r.mux.Handle(method+" "+path, r.wrap(fn, mw))
	}
}

// shouldLogRequest returns true if request logging should be enabled.
// It checks both the RequestLogger.Enabled setting and DisableDefaultMiddlewares.
func (r *defaultRouter) shouldLogRequest() bool {
	if r.config.DisableDefaultMiddlewares {
		return false
	}
	return r.config.RequestLogger.Enabled == nil || *r.config.RequestLogger.Enabled
}

// catchAllHandler returns a handler that processes unmatched requests.
// It determines whether to return a 404 Not Found or 405 Method Not Allowed response
// based on whether any methods are registered for the requested path.
func (r *defaultRouter) catchAllHandler() http.HandlerFunc {
	// Capture config values at handler creation time to avoid data races.
	// Handler fields are protected by handlerMu and accessed with locking.
	shouldLog := r.shouldLogRequest()
	requestIDHeader := r.config.RequestID.Header
	requestIDGenerator := r.config.RequestID.Generator
	requestLoggerConfig := r.config.RequestLogger
	logger := r.logger

	return func(w http.ResponseWriter, req *http.Request) {
		var start time.Time

		if shouldLog {
			start = time.Now()

			// Lazy request ID generation - only when logging
			requestID := req.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = requestIDGenerator()
			}
			w.Header().Set(requestIDHeader, requestID)
		}

		// Access registered routes with proper locking.
		// Must hold lock while reading from the inner map to avoid races
		// with handle() which writes to the same inner map.
		r.routesMu.RLock()

		// Check if this path matches any registered route pattern
		// For parameterized routes, we need to match the pattern, not exact path
		methods, exists := r.findMatchingRoute(req.URL.Path)

		if exists {
			// Auto-generate OPTIONS response
			if req.Method == http.MethodOptions {
				allowHeader := allowedMethods(methods)
				r.routesMu.RUnlock()
				w.Header().Set(httpx.HeaderAllow, allowHeader)
				w.WriteHeader(http.StatusNoContent)
				if shouldLog {
					requestlogger.Log(logger, requestLoggerConfig, nil, req, http.StatusNoContent, time.Since(start), "", "")
				}
				return
			}

			methodAllowed := methods[req.Method]
			var allowHeader string
			if !methodAllowed {
				allowHeader = allowedMethods(methods)
			}
			r.routesMu.RUnlock()

			if !methodAllowed {
				w.Header().Set(httpx.HeaderAllow, allowHeader)

				r.handlerMu.RLock()
				methodNotAllowedHandler := r.methodNotAllowedHandler
				r.handlerMu.RUnlock()
				methodNotAllowedHandler.ServeHTTP(w, req)
				return
			}

			// This path should be unreachable: if the method is registered,
			// ServeMux should have routed to its handler before the catch-all.
			// Log a warning to help diagnose route registration issues.
			logger.Warn("Catch-all reached for registered route - route table out of sync",
				log.F("path", req.URL.Path),
				log.F("method", req.Method))
		} else {
			r.routesMu.RUnlock()
		}

		r.handlerMu.RLock()
		notFoundHandler := r.notFoundHandler
		r.handlerMu.RUnlock()
		notFoundHandler.ServeHTTP(w, req)
	}
}

// findMatchingRoute checks if the request path matches any registered route pattern.
// It returns the methods map and true if a match is found.
// This handles parameterized routes like /hello/{name} matching /hello/as.
func (r *defaultRouter) findMatchingRoute(path string) (map[string]bool, bool) {
	// First try exact match (fast path for static routes)
	if methods, exists := r.registeredRoutes[path]; exists {
		return methods, true
	}

	// For parameterized routes, we need to check each pattern
	// Go's ServeMux uses patterns like /hello/{name}
	// We need to check if our path matches any registered pattern
	for pattern, methods := range r.registeredRoutes {
		if matchPattern(pattern, path) {
			return methods, true
		}
	}

	return nil, false
}

// matchPattern checks if a path matches a route pattern.
// It handles parameterized segments like {name} and wildcards like {...}
func matchPattern(pattern, path string) bool {
	// Split pattern and path into segments
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	// Different number of segments means no match (unless pattern ends with wildcard)
	if len(patternParts) != len(pathParts) {
		// Check for wildcard at end
		if len(patternParts) > 0 && patternParts[len(patternParts)-1] == "..." {
			// Wildcard matches everything, check prefix
			if len(pathParts) >= len(patternParts)-1 {
				patternParts = patternParts[:len(patternParts)-1]
				pathParts = pathParts[:len(patternParts)]
			} else {
				return false
			}
		} else {
			return false
		}
	}

	// Compare each segment
	for i, p := range patternParts {
		if i >= len(pathParts) {
			return false
		}

		// Parameterized segment like {name} or {name...}
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			// This is a parameter, it matches any value
			continue
		}

		// Parameterized segment with wildcard like {name...}
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "...}") {
			// This matches the rest of the path
			return true
		}

		// Wildcard segment
		if p == "..." {
			return true
		}

		// Exact match required
		if p != pathParts[i] {
			return false
		}
	}

	return true
}

// defaultNotFoundHandler is the default handler for 404 Not Found responses.
// It checks the Accept header and returns JSON problem detail when requested.
var defaultNotFoundHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// Check if client accepts JSON/problem detail
	accept := r.Header.Get(httpx.HeaderAccept)
	if strings.Contains(accept, httpx.MIMEApplicationJSON) || strings.Contains(accept, httpx.MIMEApplicationProblemJSON) {
		jsonNotFoundHandler(w, r)
		return
	}
	// Default to plain text
	w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlainCharset)
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Requested resource was not found\n"))
})

// defaultMethodNotAllowedHandler is the default handler for 405 Method Not Allowed responses.
// It checks the Accept header and returns JSON problem detail when requested.
// The "Allow" header should be set by the caller to indicate which methods are allowed.
var defaultMethodNotAllowedHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// Check if client accepts JSON/problem detail
	accept := r.Header.Get(httpx.HeaderAccept)
	if strings.Contains(accept, httpx.MIMEApplicationJSON) || strings.Contains(accept, httpx.MIMEApplicationProblemJSON) {
		jsonMethodNotAllowedHandler(w, r)
		return
	}
	// Default to plain text
	w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlainCharset)
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = w.Write([]byte("HTTP method is not allowed\n"))
})

// jsonNotFoundHandler returns a JSON problem detail 404 response.
func jsonNotFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(preEncodedNotFoundJSON)
}

// jsonMethodNotAllowedHandler returns a JSON problem detail 405 response.
// The "Allow" header should be set by the caller to indicate which methods are allowed.
func jsonMethodNotAllowedHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = w.Write(preEncodedMethodNotAllowedJSON)
}

// allowedMethods converts a map of HTTP methods to a comma-separated string
// suitable for the "Allow" header in 405 Method Not Allowed responses.
// Implicit HEAD is included if GET is present. OPTIONS is always included.
func allowedMethods(methods map[string]bool) string {
	result := make([]string, 0, len(methods)+2)
	for method := range methods {
		result = append(result, method)
	}
	// Implicit HEAD when GET is registered
	if methods[http.MethodGet] && !methods[http.MethodHead] {
		result = append(result, http.MethodHead)
	}
	// OPTIONS is always implicitly allowed
	if !methods[http.MethodOptions] {
		result = append(result, http.MethodOptions)
	}
	slices.Sort(result)
	return strings.Join(result, ", ")
}
