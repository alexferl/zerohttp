// Package zerohttp provides a lightweight, zero-dependency HTTP framework
// for building REST APIs and web applications in Go.
//
// Built on Go's standard library, zerohttp adds essential features for
// production services: routing, middleware, request binding, validation,
// rendering, metrics, and more - all without external dependencies.
//
// # Quick Start
//
// Create and start a server in a few lines:
//
//	package main
//
//	import (
//	    "log"
//	    "net/http"
//
//	    zh "github.com/alexferl/zerohttp"
//	)
//
//	func main() {
//	    app := zh.New()
//
//	    app.GET("/hello/{name}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	        name := zh.Param(r, "name")
//	        return zh.Render.JSON(w, http.StatusOK, zh.M{"message": "Hello, " + name})
//	    }))
//
//	    log.Fatal(app.Start())
//	}
//
// # Routing
//
// zerohttp uses Go's standard [net/http.ServeMux] for routing, supporting
// path parameters, wildcards, and method-based routes:
//
//	app := zh.New()
//
//	// Path parameters
//	app.GET("/users/{id}", getUserHandler)
//
//	// Wildcards
//	app.GET("/files/{path...}", serveFileHandler)
//
//	// Route groups with middleware
//	app.Group(func(api zh.Router) {
//	    api.Use(basicauth.New(basicauth.Config{
//	        Credentials: map[string]string{"admin": "secret"},
//	    }))
//	    api.GET("/admin/dashboard", dashboardHandler)
//	})
//
// # Handlers
//
// Handlers return errors for cleaner error handling. Errors are automatically
// converted to appropriate HTTP responses:
//
//	app.POST("/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    var req CreateUserRequest
//	    if err := zh.BindAndValidate(r, &req); err != nil {
//	        return err  // Returns 400 for binding errors, 422 for validation errors
//	    }
//
//	    user, err := createUser(req)
//	    if err != nil {
//	        return err  // Returns 500 for unexpected errors
//	    }
//
//	    return zh.Render.JSON(w, http.StatusCreated, user)
//	}))
//
// # Request Binding
//
// Bind request data to structs using [Bind]:
//
// # JSON Binding
//
// Parse JSON request bodies with strict validation (rejects unknown fields):
//
//	var req struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//	if err := zh.Bind.JSON(r.Body, &req); err != nil {
//	    return err  // Returns 400 Bad Request
//	}
//
// # Form Binding
//
// Parse application/x-www-form-urlencoded data:
//
//	var form struct {
//	    Username string   `form:"username"`
//	    Password string   `form:"password"`
//	    Remember bool     `form:"remember"`
//	    Tags     []string `form:"tags"`  // Supports slices
//	}
//	if err := zh.Bind.Form(r, &form); err != nil {
//	    return err
//	}
//
// # Multipart Form Binding
//
// Handle file uploads with multipart/form-data:
//
//	var form struct {
//	    Description string           `form:"description"`
//	    Document    *zh.FileHeader   `form:"document"`   // Single file
//	    Images      []*zh.FileHeader `form:"images"`     // Multiple files
//	}
//
//	// maxMemory: bytes to store in memory before temp files
//	if err := zh.Bind.MultipartForm(r, &form, 32<<20); err != nil {
//	    return err
//	}
//
//	// Access uploaded files
//	if form.Document != nil {
//	    file, err := form.Document.Open()
//	    if err != nil {
//	        return err
//	    }
//	    defer file.Close()
//	    data, _ := io.ReadAll(file)
//	    // Process file data...
//	}
//
// # Query Parameter Binding
//
// Bind query parameters to structs with query tags:
//
//	var req struct {
//	    Query    string   `query:"q"`
//	    Category string   `query:"category"`
//	    Tags     []string `query:"tags"`      // Multiple values: ?tags=a&tags=b
//	    Page     int      `query:"page"`
//	    Limit    int      `query:"limit"`
//	    IsActive *bool    `query:"is_active"` // Pointer = optional
//	}
//
//	if err := zh.Bind.Query(r, &req); err != nil {
//	    return err
//	}
//
// # Embedded Structs
//
// Reuse common patterns like pagination:
//
//	type Pagination struct {
//	    Page  int `query:"page"`
//	    Limit int `query:"limit"`
//	}
//
//	type ListRequest struct {
//	    Pagination
//	    Search string `query:"search"`
//	}
//
//	var req ListRequest
//	if err := zh.Bind.Query(r, &req); err != nil {
//	    return err
//	}
//
// # Path Parameters
//
// Type-safe path parameter extraction:
//
//	// Basic string extraction
//	id := zh.Param(r, "id")
//
//	// Typed extraction (returns error if invalid)
//	itemID, err := zh.ParamAs[int](r, "itemID")
//	if err != nil {
//	    return zh.NewProblemDetail(http.StatusBadRequest, "Invalid itemID").Render(w)
//	}
//
//	// With default value
//	category := zh.ParamOrDefault(r, "category", "all")
//
// # Individual Parameter Extraction
//
// Extract single query parameters:
//
//	// With type conversion
//	userID, err := zh.QueryParamAs[int](r, "user_id")
//
//	// With default value
//	page := zh.QueryParamAsOrDefault(r, "page", 1)
//
//	// Simple string
//	sort := zh.QueryParam(r, "sort")
//
// # Custom Binders
//
// Implement the [Binder] interface for custom binding logic:
//
//	type MyBinder struct{}
//
//	func (b *MyBinder) JSON(r io.Reader, dst any) error {
//	    decoder := json.NewDecoder(r)
//	    decoder.UseNumber() // Use json.Number instead of float64
//	    return decoder.Decode(dst)
//	}
//
//	// Replace default binder
//	zh.Bind = &MyBinder{}
//
// # Type Conversion
//
// Form and query binders automatically convert string values to Go types:
//
//	string   -> "name=John"           -> "John"
//	int      -> "age=25"              -> 25
//	bool     -> "active=true"         -> true
//	[]string -> "tags=a&tags=b"       -> ["a", "b"]
//	[]int    -> "ids=1&ids=2"         -> [1, 2]
//	*string  -> "optional=" or missing -> nil or ""
//
// Supported types: string, all int/uint types, float32, float64, bool, slices, and pointers.
//
// # Validation
//
// Validate structs using struct tags with the built-in validator:
//
//	type CreateUserRequest struct {
//	    Name     string `json:"name"     validate:"required,min=2,max=50"`
//	    Email    string `json:"email"    validate:"required,email"`
//	    Age      int    `json:"age"      validate:"min=13,max=120"`
//	    Password string `json:"password" validate:"required,min=8"`
//	}
//
//	if err := zh.Validate.Struct(&req); err != nil {
//	    // Returns ValidationErrors map keyed by field name
//	    return err
//	}
//
// # Available Validators
//
// Core validators: required, omitempty, eq, ne
//
// String validators: min, max, len, contains, startswith, endswith, excludes,
// alpha, alphanum, lowercase, uppercase, ascii, printascii, numeric, oneof
//
// Numeric validators: min, max, gt, lt, gte, lte
//
// Format validators: email, uuid, datetime, base64, hexadecimal, hexcolor,
// e164, semver, jwt, boolean, json
//
// Network validators: ip, ipv4, ipv6, cidr, hostname, uri, url
//
// Collection validators: unique, each
//
// # Combining Validators
//
// Multiple validators can be combined with commas:
//
//	type Product struct {
//	    Name  string   `validate:"required,min=2,max=100"`
//	    Price float64  `validate:"required,gt=0"`
//	    Tags  []string `validate:"unique,each,min=2,max=20"`
//	}
//
// # Nested Struct Validation
//
// Validation automatically recurses into nested structs:
//
//	type Address struct {
//	    Street string `validate:"required"`
//	    City   string `validate:"required"`
//	}
//
//	type Person struct {
//	    Name    string  `validate:"required"`
//	    Address Address // validated recursively
//	}
//
// For slices of structs, use the each validator:
//
//	type Order struct {
//	    Items []LineItem `validate:"each"` // validates each LineItem
//	}
//
// # Pointer Fields
//
// Pointer fields are dereferenced before validation. Use omitempty to make optional:
//
//	type User struct {
//	    Name     *string `validate:"omitempty,min=2"` // nil or valid
//	    Nickname *string `validate:"required,min=2"`  // must not be nil
//	}
//
// # Custom Validators
//
// Register custom validators with V.Register:
//
//	zh.Validate.Register("even", func(value reflect.Value, param string) error {
//	    if value.Kind() != reflect.Int {
//	        return fmt.Errorf("even only validates integers")
//	    }
//	    if value.Int()%2 != 0 {
//	        return fmt.Errorf("must be even")
//	    }
//	    return nil
//	})
//
//	type Config struct {
//	    Port int `validate:"required,even"`
//	}
//
// # Validation Error Handling
//
// Validate.Struct returns a ValidationErrors map keyed by field name:
//
//	if err := zh.Validate.Struct(&user); err != nil {
//	    var ve zh.ValidationErrors
//	    if errors.As(err, &ve) {
//	        errs := ve.FieldErrors("Email")
//	        for _, e := range errs {
//	            fmt.Println(e) // "required" or "must be a valid email"
//	        }
//
//	        if ve.HasErrors() {
//	            pd := zh.NewValidationProblemDetail("Validation failed", ve)
//	            pd.Render(w)
//	        }
//	    }
//	}
//
// Errors use JSON field names when available.
//
// # Response Rendering
//
// Render responses using [Render]:
//
//	// JSON response
//	zh.Render.JSON(w, http.StatusOK, zh.M{"users": users})
//
//	// Text response
//	zh.Render.Text(w, http.StatusOK, "Hello, World!")
//
//	// HTML response
//	zh.Render.HTML(w, http.StatusOK, "<h1>Hello</h1>")
//
//	// File download
//	zh.Render.File(w, r, "/path/to/file.pdf")
//
//	// Redirect
//	zh.Render.Redirect(w, r, "/new-path", http.StatusFound)
//
// # Error Handling
//
// zerohttp converts errors to RFC 9457 Problem Details responses:
//
//	// Return custom problem detail
//	return zh.NewProblemDetail(http.StatusNotFound, "User not found").Render(w)
//
//	// Return validation errors (422 Unprocessable Entity)
//	return zh.Validate.Struct(&req)
//
// # Middleware
//
// Apply middleware at application, group, or route level:
//
//	// Application-level
//	app.Use(cors.New(cors.DefaultConfig))
//	app.Use(requestid.New())
//
//	// Route-level
//	app.GET("/admin", adminHandler,
//	    basicauth.New(basicauth.Config{
//	        Credentials: map[string]string{"admin": "secret"},
//	    }),
//	)
//
// Available middleware: cors, basicauth, jwtauth, ratelimit, compress,
// requestlogger, circuitbreaker, timeout, and more in subpackages.
// See package middleware for complete documentation.
//
// # Metrics
//
// Prometheus-compatible metrics are automatically collected:
//
//	// Metrics exposed at /metrics by default
//	app := zh.New() // No configuration needed
//
//	// Access registry in handlers
//	reg := metrics.GetRegistry(r.Context())
//	counter := reg.Counter("orders_total", "status")
//	counter.WithLabelValues("completed").Inc()
//
// See package metrics for detailed metrics documentation.
//
// # Pluggable Features
//
// zerohttp provides pluggable interfaces for optional features.
// Configure via Config:
//
//	app := zh.New(zh.Config{
//	    Validator: myValidator,
//	    Tracer:    myTracer,
//	    Extensions: zh.ExtensionsConfig{
//	        AutocertManager:    myCertManager,
//	        HTTP3Server:        myH3Server,
//	        SSEProvider:        mySSEProvider,
//	        WebSocketUpgrader:  myWSUpgrader,
//	        WebTransportServer: myWTServer,
//	    },
//	})
//
// # Custom Validator
//
// Bring your own struct validator (e.g., go-playground/validator/v10):
//
//	type myValidator struct {
//	    v *validator.Validate
//	}
//
//	func (m *myValidator) Struct(dst any) error {
//	    return m.v.Struct(dst)
//	}
//
//	func (m *myValidator) Register(name string, fn func(reflect.Value, string) error) {
//	    m.v.RegisterValidation(name, func(fl validator.FieldLevel) bool {
//	        err := fn(fl.Field(), fl.Param())
//	        return err == nil
//	    })
//	}
//
//	app := zh.New(zh.Config{Validator: &myValidator{v: validator.New()}})
//
// # Distributed Tracing
//
// Integrate your preferred tracing solution:
//
//	app := zh.New(zh.Config{Tracer: myTracer})
//	app.Use(tracer.New(myTracer))
//
//	// In handlers
//	span := trace.SpanFromContext(r.Context())
//	span.SetAttributes(trace.String("user.id", userID))
//
// See package trace for interface details.
//
// # Auto-TLS
//
// Automatic certificate management via Let's Encrypt:
//
//	manager := &autocert.Manager{
//	    Cache:      autocert.DirCache("/tmp/certs"),
//	    Prompt:     autocert.AcceptTOS,
//	    HostPolicy: autocert.HostWhitelist("example.com"),
//	}
//
//	app := zh.New(zh.Config{
//	    Extensions: zh.ExtensionsConfig{
//	        AutocertManager: manager,
//	    },
//	})
//	app.StartAutoTLS()
//
// # HTTP/3
//
// HTTP/3 support over QUIC:
//
//	h3Server := &http3.Server{
//	    Addr:    ":443",
//	    Handler: app,
//	}
//
//	app.SetHTTP3Server(h3Server)
//	app.StartTLS("cert.pem", "key.pem") // HTTP/3 starts automatically
//
// # Server-Sent Events
//
// Real-time unidirectional streaming:
//
//	app := zh.New(zh.Config{
//	    Extensions: zh.ExtensionsConfig{
//	        SSEProvider: sse.NewDefaultProvider(),
//	    },
//	})
//
//	app.GET("/events", func(w http.ResponseWriter, r *http.Request) error {
//	    provider := app.SSEProvider()
//	    stream, err := provider.NewSSE(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer stream.Close()
//
//	    for i := 0; i < 10; i++ {
//	        stream.Send(sse.Event{Name: "message", Data: []byte("hello")})
//	        time.Sleep(1 * time.Second)
//	    }
//	    return nil
//	})
//
// # WebSocket
//
// Real-time bidirectional communication. Bring your own library:
//
//	app := zh.New(zh.Config{
//	    Extensions: zh.ExtensionsConfig{
//	        WebSocketUpgrader: &myUpgrader{upgrader: websocketUpgrader},
//	    },
//	})
//
//	app.GET("/ws", func(w http.ResponseWriter, r *http.Request) error {
//	    ws, err := app.WebSocketUpgrader().Upgrade(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer ws.Close()
//	    // Handle connection...
//	    return nil
//	})
//
// # WebTransport
//
// Low-latency bidirectional communication over HTTP/3:
//
//	h3 := &http3.Server{Addr: ":8443", Handler: app}
//	wtServer := &webtransport.Server{H3: h3, CheckOrigin: func(r *http.Request) bool { return true }}
//	webtransport.ConfigureHTTP3Server(h3)
//
//	app.SetWebTransportServer(wtServer)
//	app.ListenAndServeTLS("cert.pem", "key.pem")
//
// # Configuration
//
// Configure the server using [Config]:
//
//	app := zh.New(zh.Config{
//	    Server: &http.Server{
//	        ReadTimeout:    10 * time.Second,
//	        WriteTimeout:   15 * time.Second,
//	        MaxHeaderBytes: 1 << 20,
//	    },
//	})
//
// # Server Lifecycle
//
// Start the server with various methods:
//
//	// HTTP
//	app.Start()              // Uses config.Addr or :8080
//	app.ListenAndServe()     // Uses configured address
//
//	// HTTPS
//	app.StartTLS("cert.pem", "key.pem")
//	app.StartAutoTLS()       // Let's Encrypt
//
//	// With graceful shutdown
//	go app.Start()
//
//	quit := make(chan os.Signal, 1)
//	signal.Notify(quit, os.Interrupt)
//	<-quit
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := app.Shutdown(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Testing
//
// The zhtest package provides fluent test helpers:
//
//	func TestGetUser(t *testing.T) {
//	    app := setupRouter()
//
//	    req := zhtest.NewRequest(http.MethodGet, "/users/123").Build()
//	    w := zhtest.Serve(app, req)
//
//	    zhtest.AssertWith(t, w).
//	        Status(http.StatusOK).
//	        Header("Content-Type", "application/json").
//	        JSONPathEqual("name", "John Doe")
//	}
//
// See package zhtest for detailed testing documentation.
//
// # Short Aliases
//
// For convenience, common types have short aliases:
//
//	zh.M      // map[string]any - for JSON responses
//	zh.B      // Bind (alias for Bind)
//	zh.R      // Render (alias for Render)
//	zh.V      // Validate (alias for Validate)
package zerohttp
