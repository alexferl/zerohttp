# HTTP/3 Example

This example demonstrates how to add HTTP/3 support to zerohttp using the pluggable HTTP/3 interface.

## Features

- HTTP/3 support via quic-go
- Pluggable HTTP/3 server interface
- Graceful shutdown support

## Prerequisites

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

## Running the Example

```bash
go mod tidy
go run .
```

The server starts on `https://localhost:8443`.

## Test Commands

### Using curl (with HTTP/3 support):
```bash
curl -i --http3 https://localhost:8443
```

### Using a browser:
1. Open Chrome, Firefox, or Safari (all support HTTP/3)
2. Navigate to `https://localhost:8443`
3. Open Developer Tools → Network tab to verify HTTP/3 protocol

### Using quic-go's client:
```bash
go run github.com/quic-go/quic-go/example/client@latest https://localhost:8443
```

## How It Works

zerohttp provides a `config.HTTP3Server` interface that any HTTP/3 implementation can satisfy:

```go
type HTTP3Server interface {
    ListenAndServeTLS(certFile, keyFile string) error
    Shutdown(ctx context.Context) error
    Close() error
}
```

This allows you to inject [quic-go/http3](https://github.com/quic-go/quic-go) or any other HTTP/3 implementation.

## Notes

- HTTP/3 requires TLS (QUIC uses TLS 1.3)
- You can run HTTP/3 alongside HTTP/1 and HTTP/2 on the same port (QUIC handles this)
- The zerohttp `Shutdown()` method will gracefully shut down all servers including HTTP/3
- The `SetHTTP3Server()` method allows injecting the HTTP/3 server after creating the zerohttp instance
- Implement `HTTP3ServerWithAutocert` for automatic Let's Encrypt certificate support
