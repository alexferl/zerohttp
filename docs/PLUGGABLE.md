# Pluggable Features

zerohttp provides several pluggable features that extend core functionality through interfaces. These features require external dependencies and are opt-in.

## Table of Contents

- [Interface Overview](#interface-overview)
- [Validator](#validator)
- [Distributed Tracing](#distributed-tracing)
- [Auto-TLS](#auto-tls)
- [HTTP/3](#http3)
- [Server-Sent Events (SSE)](#server-sent-events-sse)
- [WebSocket](#websocket)
- [WebTransport](#webtransport)

## Interface Overview

zerohttp defines minimal interfaces for each pluggable feature:

```go
// Validator - Struct validation with custom libraries
type Validator interface {
    Struct(dst any) error
    Register(name string, fn func(reflect.Value, string) error)
}

// Tracer - Distributed tracing interface
type Tracer interface {
    Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// AutocertManager - Automatic certificate management
type AutocertManager interface {
    GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)
    HTTPHandler(fallback http.Handler) http.Handler
}

// HTTP3Server - HTTP/3 over QUIC support
type HTTP3Server interface {
    ListenAndServeTLS(certFile, keyFile string) error
    Shutdown(ctx context.Context) error
    Close() error
}

// SSEProvider - Server-Sent Events provider
type SSEProvider interface {
    NewSSE(w http.ResponseWriter, r *http.Request) (SSEStream, error)
}

// WebSocketUpgrader - WebSocket connection upgrade
type WebSocketUpgrader interface {
    Upgrade(w http.ResponseWriter, r *http.Request) (WebSocketConn, error)
}

// WebTransportServer - WebTransport over HTTP/3
type WebTransportServer interface {
    ListenAndServeTLS(certFile, keyFile string) error
    Close() error
}
```

All pluggable features are configured via the Config struct:

```go
app := zh.New(config.Config{
    Validator:          myValidator,
    Tracer:             myTracer,
    AutocertManager:    myCertManager,
    HTTP3Server:        myH3Server,
    SSEProvider:        mySSEProvider,
    WebSocketUpgrader:  myWSUpgrader,
    WebTransportServer: myWTServer,
})
```

## Validator

Struct validation using custom validator libraries. zerohttp includes a built-in validator, but you can plug in alternatives like `go-playground/validator/v10`.

```go
import (
    "reflect"

    "github.com/go-playground/validator/v10"
    zh "github.com/alexferl/zerohttp"
)

// Wrap go-playground/validator to implement the interface
type myValidator struct {
    v *validator.Validate
}

func (m *myValidator) Struct(dst any) error {
    return m.v.Struct(dst)
}

func (m *myValidator) Register(name string, fn func(reflect.Value, string) error) {
    m.v.RegisterValidation(name, func(fl validator.FieldLevel) bool {
        err := fn(fl.Field(), fl.Param())
        return err == nil
    })
}

func main() {
    app := zh.New(config.Config{
        Validator: &myValidator{v: validator.New()},
    })

    app.POST("/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        var req CreateUserRequest
        if err := zh.Bind.JSON(r, &req); err != nil {
            return err
        }

        // Use the custom validator
        if err := app.Validator().Struct(&req); err != nil {
            return err
        }

        return zh.R.JSON(w, 201, req)
    }))

    app.ListenAndServe()
}
```

See [`examples/validation/goplayground.go`](../examples/validation/goplayground.go) for a complete example.

## Distributed Tracing

zerohttp provides a pluggable tracing interface for distributed tracing. No external dependencies are required - you can implement your own tracer or use OpenTelemetry.

### Basic Usage (No External Dependencies)

```go
package main

import (
    zh "github.com/alexferl/zerohttp"
    "github.com/alexferl/zerohttp/config"
    "github.com/alexferl/zerohttp/middleware"
    "github.com/alexferl/zerohttp/trace"
)

// SimpleTracer logs spans to stdout
type SimpleTracer struct{}

func (t *SimpleTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
    span := &SimpleSpan{name: name}
    return trace.ContextWithSpan(ctx, span), span
}

type SimpleSpan struct{ name string }
func (s *SimpleSpan) End() {}
func (s *SimpleSpan) SetStatus(code trace.Code, description string) {}
func (s *SimpleSpan) SetAttributes(attrs ...trace.Attribute) {}
func (s *SimpleSpan) RecordError(err error, opts ...trace.ErrorOption) {}

func main() {
    tracer := &SimpleTracer{}

    app := zh.New(config.Config{
        Tracer: tracer,
    })

    // Add tracing middleware
    app.Use(middleware.Tracing(tracer))

    // Access span in handlers
    app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        span := trace.SpanFromContext(r.Context())
        span.SetAttributes(trace.String("user.id", "123"))
        return zh.R.JSON(w, 200, zh.M{"message": "ok"})
    }))

    app.ListenAndServe()
}
```

### With OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Create tracer provider with Jaeger exporter
exp, _ := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))
otel.SetTracerProvider(tp)

// Wrap OTel tracer to implement zerohttp's interface
tracer := &OTelTracer{tracer: tp.Tracer("myapp")}

app := zh.New(config.Config{Tracer: tracer})
app.Use(middleware.Tracing(tracer))
```

See [`examples/tracing/`](../examples/tracing/) for complete working examples including custom and OpenTelemetry implementations.

## Auto-TLS

Automatic certificate management via Let's Encrypt or other ACME providers:

```go
import "golang.org/x/crypto/acme/autocert"

manager := &autocert.Manager{
    Cache:      autocert.DirCache("/tmp/certs"),
    Prompt:     autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("example.com"),
}

app := zh.New(config.Config{
    AutocertManager: manager,
})

// Starts HTTP (for ACME) and HTTPS servers
app.StartAutoTLS()
```

## HTTP/3

HTTP/3 support over QUIC. HTTP/2 is enabled by default for TLS; HTTP/3 requires a pluggable implementation.

```go
import "github.com/quic-go/quic-go/http3"

app := zh.New()

h3Server := &http3.Server{
    Addr:    ":443",
    Handler: app,
}

app.SetHTTP3Server(h3Server)
app.StartTLS("cert.pem", "key.pem") // HTTP/3 starts automatically
```

## Server-Sent Events (SSE)

Real-time unidirectional server-to-client streaming:

```go
import (
    "time"
    zh "github.com/alexferl/zerohttp"
    "github.com/alexferl/zerohttp/config"
)

func main() {
    app := zh.New(config.Config{
        SSEProvider: zh.NewDefaultProvider(),
    })

    app.GET("/events", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        stream, err := app.SSEProvider().NewSSE(w, r)
        if err != nil {
            return err
        }
        defer stream.Close()

        for i := 0; i < 10; i++ {
            stream.Send(zh.SSEEvent{
                Name: "message",
                Data: []byte("hello"),
            })
            time.Sleep(1 * time.Second)
        }
        return nil
    }))

    app.ListenAndServe()
}
```

See [`examples/sse/`](../examples/sse/) for a complete example with event replay and broadcast hub.

## WebSocket

Real-time bidirectional communication. You bring your own WebSocket library.

```go
import (
    "github.com/gorilla/websocket"
    zh "github.com/alexferl/zerohttp"
)

// Wrap gorilla/websocket to implement the interface
type myUpgrader struct {
    upgrader *websocket.Upgrader
}

func (m *myUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (zh.WebSocketConn, error) {
    conn, err := m.upgrader.Upgrade(w, r, nil)
    if err != nil {
        return nil, err
    }
    return &myConn{conn: conn}, nil
}

type myConn struct {
    conn *websocket.Conn
}

func (c *myConn) ReadMessage() (int, []byte, error)  { return c.conn.ReadMessage() }
func (c *myConn) WriteMessage(mt int, data []byte) error { return c.conn.WriteMessage(mt, data) }
func (c *myConn) Close() error                        { return c.conn.Close() }
func (c *myConn) RemoteAddr() net.Addr               { return c.conn.RemoteAddr() }

func main() {
    app := zh.New(config.Config{
        WebSocketUpgrader: &myUpgrader{
            upgrader: &websocket.Upgrader{
                CheckOrigin: func(r *http.Request) bool { return true },
            },
        },
    })

    app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        ws, err := app.WebSocketUpgrader().Upgrade(w, r)
        if err != nil {
            return err
        }
        defer ws.Close()

        // Echo loop
        for {
            mt, msg, err := ws.ReadMessage()
            if err != nil {
                break
            }
            if err := ws.WriteMessage(mt, msg); err != nil {
                break
            }
        }
        return nil
    }))

    app.ListenAndServe()
}
```

See [`examples/websocket/`](../examples/websocket/) for a complete example.

## WebTransport

Low-latency bidirectional communication over HTTP/3:

```go
import (
    "github.com/quic-go/quic-go/http3"
    webtransport "github.com/quic-go/webtransport-go"
)

func main() {
    app := zh.New()

    h3 := &http3.Server{Addr: ":8443", Handler: app}
    wtServer := &webtransport.Server{
        H3:          h3,
        CheckOrigin: func(r *http.Request) bool { return true },
    }
    webtransport.ConfigureHTTP3Server(h3)

    app.SetWebTransportServer(wtServer)

    app.CONNECT("/wt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        sess, err := wtServer.Upgrade(w, r)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        go handleSession(sess)
    }))

    app.ListenAndServeTLS("cert.pem", "key.pem")
}
```

See [`examples/webtransport/`](../examples/webtransport/) for a complete example.
