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

This allows you to inject [quic-go/http3](https://github.com/quic-go/quic-go) or any other HTTP/3 implementation without adding dependencies to zerohttp.

## Setup

1. Install quic-go:
```bash
go get github.com/quic-go/quic-go
```

2. Generate self-signed TLS certificates:
```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes
```

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
// Wrap quic-go's server to add autocert support
type HTTP3AutocertServer struct {
    *http3.Server
}

func (s *HTTP3AutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
    return s.Server.ListenAndServeTLS(certFile, keyFile)
}

func (s *HTTP3AutocertServer) ListenAndServeTLSWithAutocert(manager *autocert.Manager) error {
    if s.Server.TLSConfig == nil {
        s.Server.TLSConfig = &tls.Config{}
    }
    s.Server.TLSConfig.GetCertificate = manager.GetCertificate
    return s.Server.ListenAndServeTLS("", "")
}

// Usage with AutoTLS
manager := autocert.NewManager(autocert.DirCache("/var/cache/certs"), autocert.HostWhitelist("example.com"))

h3Server := &HTTP3AutocertServer{
    Server: &http3.Server{Addr: ":443", Handler: router},
}

srv := zerohttp.New(
    config.WithAutocertManager(manager),
    config.WithHTTP3Server(h3Server),
)

// Starts HTTP, HTTPS, and HTTP/3 all with AutoTLS
srv.StartAutoTLS("example.com")
```

## Key Points

- HTTP/3 requires TLS (QUIC uses TLS 1.3)
- You can run HTTP/3 alongside HTTP/1 and HTTP/2 on the same port (QUIC handles this)
- The zerohttp `Shutdown()` method will gracefully shut down all servers including HTTP/3
- The `SetHTTP3Server()` method allows injecting the HTTP/3 server after creating the zerohttp instance
- Implement `HTTP3ServerWithAutocert` for automatic Let's Encrypt certificate support