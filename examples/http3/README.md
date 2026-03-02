# HTTP/3 Example

This example demonstrates how to add HTTP/3 support to zerohttp using the pluggable HTTP/3 interface.

## How it works

zerohttp provides a `config.HTTP3Server` interface that any HTTP/3 implementation can satisfy:

```go
type HTTP3Server interface {
    ListenAndServeTLS(certFile, keyFile string) error
    Shutdown(ctx context.Context) error
    Close() error
}
```

This allows you to inject [quic-go/http3](https://github.com/quic-go/quic-go) or any other HTTP/3 implementation.

## Setup

1. Install quic-go:
```bash
go get github.com/quic-go/quic-go
```

2. Install mkcert and generate certificates:
```bash
brew install mkcert
mkcert -install
mkcert localhost 127.0.0.1 ::1
```
This creates: `localhost+2.pem` and `localhost+2-key.pem`

3. Run the server:
```bash
go run main.go
```

## Testing HTTP/3

### Using curl (with HTTP/3 support):
```bash
curl --http3 -k https://localhost:8443
```

### Using a browser:
1. Open Chrome, Firefox, or Safari (all support HTTP/3)
2. Navigate to `https://localhost:8443`
3. Open Developer Tools → Network tab to verify HTTP/3 protocol

### Using quic-go's client:
```bash
go run github.com/quic-go/quic-go/example/client@latest https://localhost:8443
```

## AutoTLS (Let's Encrypt)

For automatic HTTPS certificates, implement the `HTTP3ServerWithAutocert` interface:

```go
// http3AutocertServer wraps quic-go's http3.Server to implement
// config.HTTP3ServerWithAutocert interface
type http3AutocertServer struct {
    server *http3.Server
}

func (h *http3AutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
    return h.server.ListenAndServeTLS(certFile, keyFile)
}

func (h *http3AutocertServer) Shutdown(ctx context.Context) error {
    return h.server.Shutdown(ctx)
}

func (h *http3AutocertServer) Close() error {
    return nil
}

func (h *http3AutocertServer) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
    tlsConfig := &tls.Config{
        GetCertificate: manager.GetCertificate,
        NextProtos:     []string{"h3"},
    }
    h.server.TLSConfig = tlsConfig
    return h.server.ListenAndServe()
}

// Usage with AutoTLS
manager := &autocert.Manager{
    Cache:      autocert.DirCache("/var/cache/certs"),
    Prompt:     autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("example.com"),
}

app := zerohttp.New(
    config.WithAutocertManager(manager),
)

h3Server := &http3AutocertServer{
    server: &http3.Server{Addr: ":443", Handler: app},
}
app.SetHTTP3Server(h3Server)

// Starts HTTP, HTTPS, and HTTP/3 all with AutoTLS
log.Fatal(app.StartAutoTLS())
```

## Key Points

- HTTP/3 requires TLS (QUIC uses TLS 1.3)
- You can run HTTP/3 alongside HTTP/1 and HTTP/2 on the same port (QUIC handles this)
- The zerohttp `Shutdown()` method will gracefully shut down all servers including HTTP/3
- The `SetHTTP3Server()` method allows injecting the HTTP/3 server after creating the zerohttp instance
- Implement `HTTP3ServerWithAutocert` for automatic Let's Encrypt certificate support
