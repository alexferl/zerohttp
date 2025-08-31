package zerohttp

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/middleware"
)

// HandlerFunc is a handler function that returns an error
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP implements http.Handler interface
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		panic(fmt.Errorf("handler error: %w", err))
	}
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
	Config() config.Config

	// SetConfig updates the router's configuration. This affects how
	// the router handles various behaviors including middleware settings
	// and error response processing.
	//
	// Note: Changing the configuration affects both regular routes
	// and 404/405 error responses.
	SetConfig(config config.Config)
}

// defaultRouter is the concrete implementation of the Router interface.
// It wraps Go's standard http.ServeMux and adds method-specific routing,
// middleware support, and proper HTTP status code handling.
type defaultRouter struct {
	// mux is the underlying HTTP multiplexer that handles request routing
	mux *http.ServeMux

	// chain contains the middleware functions that will be applied to all routes
	chain []func(http.Handler) http.Handler

	// notFoundHandler is called when no route matches the request path
	notFoundHandler http.Handler

	// methodNotAllowedHandler is called when a path exists but the HTTP method is not allowed
	methodNotAllowedHandler http.Handler

	// registeredRoutes tracks which HTTP methods are registered for each path
	// This is used to distinguish between 404 Not Found and 405 Method Not Allowed
	registeredRoutes map[string]map[string]bool // path -> method -> bool

	// logger is the structured logger used by the server and its middleware
	// for recording HTTP requests, errors, and server lifecycle events.
	logger log.Logger

	// config holds the complete configuration for the router including
	// middleware settings and behavioral options. This configuration
	// affects how routes and error responses are handled.
	config config.Config
}

// NewRouter creates a new router instance with optional global middleware.
// The middleware provided here will be applied to all routes registered on this router.
//
// Example:
//
//	router := NewRouter(loggingMiddleware, authMiddleware)
func NewRouter(mw ...func(http.Handler) http.Handler) Router {
	cfg := config.DefaultConfig
	cfg.Build()

	r := &defaultRouter{
		mux:                     &http.ServeMux{},
		chain:                   mw,
		notFoundHandler:         defaultNotFoundHandler,
		methodNotAllowedHandler: defaultMethodNotAllowedHandler,
		registeredRoutes:        make(map[string]map[string]bool),
		logger:                  log.NewDefaultLogger(),
		config:                  cfg,
	}
	// Register a catch-all handler to handle 404 and 405 responses
	r.mux.HandleFunc("/", r.catchAllHandler())
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
	groupRouter := &defaultRouter{
		mux:                     r.mux,
		chain:                   slices.Clone(r.chain), // Clone to avoid affecting parent
		notFoundHandler:         r.notFoundHandler,
		methodNotAllowedHandler: r.methodNotAllowedHandler,
		registeredRoutes:        r.registeredRoutes,
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
func (r *defaultRouter) PATCH(path string, fn http.Handler, mw ...func(http.Handler) http.Handler) {
	r.handle(http.MethodPatch, path, fn, mw)
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

// NotFound sets a custom handler for 404 Not Found responses.
// This handler will be called when no registered route matches the request path.
//
// Example:
//
//	router.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    http.Error(w, "Custom 404 message", http.StatusNotFound)
//	}))
func (r *defaultRouter) NotFound(h http.Handler) {
	r.notFoundHandler = h
}

// MethodNotAllowed sets a custom handler for 405 Method Not Allowed responses.
// This handler will be called when a path exists but the HTTP method is not registered for it.
// The handler should check the "Allow" header to see which methods are allowed.
//
// Example:
//
//	router.MethodNotAllowed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    allow := w.Header().Get("Allow")
//	    http.Error(w, fmt.Sprintf("Method not allowed. Allowed: %s", allow), http.StatusMethodNotAllowed)
//	}))
func (r *defaultRouter) MethodNotAllowed(h http.Handler) {
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

// Static serves a static web application from embedded FS with fallback to index.html.
func (r *defaultRouter) Static(embedFS embed.FS, distDir string, fallback bool, apiPrefix ...string) {
	subFS, err := fs.Sub(embedFS, distDir)
	if err != nil {
		panic(fmt.Errorf("failed to create sub-filesystem: %w", err))
	}

	handler := r.createStaticHandler(subFS, fallback, apiPrefix)

	r.mux.Handle("GET /{$}", r.wrap(handler, nil))
	r.mux.Handle("GET /{path...}", r.wrap(handler, nil))
}

// StaticDir serves a static web application from a directory with fallback to index.html.
func (r *defaultRouter) StaticDir(dir string, fallback bool, apiPrefix ...string) {
	filesystem := os.DirFS(dir)
	handler := r.createStaticHandler(filesystem, fallback, apiPrefix)

	r.mux.Handle("GET /{$}", r.wrap(handler, nil))
	r.mux.Handle("GET /{path...}", r.wrap(handler, nil))
}

// createStaticHandler creates an HTTP handler for static routing with API prefix exclusions.
func (r *defaultRouter) createStaticHandler(filesystem fs.FS, fallback bool, apiPrefixes []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		requestID := req.Header.Get(r.config.RequestID.Header)
		if requestID == "" {
			requestID = r.config.RequestID.Generator()
		}
		w.Header().Set(r.config.RequestID.Header, requestID)

		cleanPath := path.Clean(req.URL.Path)

		// Skip API routes - return 404
		for _, prefix := range apiPrefixes {
			if strings.HasPrefix(cleanPath, prefix) {
				r.notFoundHandler.ServeHTTP(w, req)
				middleware.LogRequest(r.logger, r.config.RequestLogger, req, http.StatusNotFound, time.Since(start))
				return
			}
		}

		if file, err := filesystem.Open(strings.TrimPrefix(cleanPath, "/")); err == nil {
			defer func() {
				if cErr := file.Close(); cErr != nil {
					r.logger.Error("Failed to close file", log.F("error", cErr), log.F("path", cleanPath))
				}
			}()

			if stat, err := file.Stat(); err == nil && !stat.IsDir() {
				http.FileServer(http.FS(filesystem)).ServeHTTP(w, req)
				middleware.LogRequest(r.logger, r.config.RequestLogger, req, http.StatusOK, time.Since(start))
				return
			}
		}

		if fallback {
			// Fallback to index.html for client-side routing
			req.URL.Path = "/"
			http.FileServer(http.FS(filesystem)).ServeHTTP(w, req)
			middleware.LogRequest(r.logger, r.config.RequestLogger, req, http.StatusOK, time.Since(start))
		} else {
			// Use custom 404 handler for missing files
			r.notFoundHandler.ServeHTTP(w, req)
			middleware.LogRequest(r.logger, r.config.RequestLogger, req, http.StatusNotFound, time.Since(start))
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
}

// Config returns the current configuration used by the router.
// This configuration controls various aspects of router behavior
// including middleware settings and error response handling.
func (r *defaultRouter) Config() config.Config {
	return r.config
}

// SetConfig updates the router's configuration. This affects how
// the router handles various behaviors including middleware settings
// and error response processing.
//
// Note: Changing the configuration affects both regular routes
// and 404/405 error responses.
func (r *defaultRouter) SetConfig(cfg config.Config) {
	cfg.Build()
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
	if r.registeredRoutes[path] == nil {
		r.registeredRoutes[path] = make(map[string]bool)
	}
	r.registeredRoutes[path][method] = true

	// Special handling for root path to prevent catch-all behavior
	// The {$} pattern ensures exact match for the root path
	if path == "/" {
		r.mux.Handle(method+" /{$}", r.wrap(fn, mw))
	} else {
		r.mux.Handle(method+" "+path, r.wrap(fn, mw))
	}
}

// catchAllHandler returns a handler that processes unmatched requests.
// It determines whether to return a 404 Not Found or 405 Method Not Allowed response
// based on whether any methods are registered for the requested path.
func (r *defaultRouter) catchAllHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		requestID := req.Header.Get(r.config.RequestID.Header)
		if requestID == "" {
			requestID = r.config.RequestID.Generator()
		}

		w.Header().Set(r.config.RequestID.Header, requestID)

		if methods, exists := r.registeredRoutes[req.URL.Path]; exists {
			if !methods[req.Method] {
				w.Header().Set(HeaderAllow, allowedMethods(methods))
				r.methodNotAllowedHandler.ServeHTTP(w, req)
				middleware.LogRequest(r.logger, r.config.RequestLogger, req, http.StatusMethodNotAllowed, time.Since(start))
				return
			}
		}

		r.notFoundHandler.ServeHTTP(w, req)
		middleware.LogRequest(r.logger, r.config.RequestLogger, req, http.StatusNotFound, time.Since(start))
	}
}

// defaultNotFoundHandler is the default handler for 404 Not Found responses.
// It returns a problem detail response indicating that the requested resource was not found.
var defaultNotFoundHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	problem := NewProblemDetail(http.StatusNotFound, "The requested resource was not found")
	if err := R.ProblemDetail(w, problem); err != nil {
		panic(fmt.Errorf("failed to write 404 problem detail: %w", err))
	}
})

// defaultMethodNotAllowedHandler is the default handler for 405 Method Not Allowed responses.
// It returns a problem detail response indicating that the HTTP method is not allowed.
// The "Allow" header should be set by the caller to indicate which methods are allowed.
var defaultMethodNotAllowedHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	problem := NewProblemDetail(http.StatusMethodNotAllowed, "The HTTP method is not allowed")
	if err := R.ProblemDetail(w, problem); err != nil {
		panic(fmt.Errorf("failed to write 405 problem detail: %w", err))
	}
})

// allowedMethods converts a map of HTTP methods to a comma-separated string
// suitable for the "Allow" header in 405 Method Not Allowed responses.
func allowedMethods(methods map[string]bool) string {
	var result []string
	for method := range methods {
		result = append(result, method)
	}
	return strings.Join(result, ", ")
}
