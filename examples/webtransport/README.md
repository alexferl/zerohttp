# WebTransport Example

A complete WebTransport echo server using zerohttp with HTTP/3 support.

## Features

- **Datagrams** - Unreliable, unordered message transmission (UDP-like)
- **Bidirectional Streams** - Reliable, ordered byte streams (TCP-like)
- **HTTP/3** - Runs over QUIC for low-latency connections
- **HTML Client** - Built-in browser client for testing

## Requirements

- [mkcert](https://github.com/FiloSottile/mkcert) for local HTTPS certificates
- Chrome, Firefox, or Safari with WebTransport support

## Examples

This directory contains two examples:

- **`main.go`** - WebTransport with local certificates (for development)
- **`autotls.go`** - WebTransport with Let's Encrypt AutoTLS (for production)

---

## Development Setup (main.go)

Use this for local development with self-signed certificates.

### 1. Install mkcert

```bash
brew install mkcert
mkcert -install
```

### 2. Generate Certificates

```bash
mkcert localhost 127.0.0.1 ::1
```

This creates `localhost+2.pem` and `localhost+2-key.pem` in the current directory.

### 3. Run the Server

```bash
go run main.go
```

## Usage

1. Open https://localhost:8443 in your browser
2. Click **Connect** to establish a WebTransport session
3. Test sending:
   - **Datagrams** - Unreliable messages (may be lost)
   - **Stream Messages** - Reliable messages with ordering guarantees

## How It Works

### Server Architecture

WebTransport runs exclusively over HTTP/3 (QUIC). The server uses:

- **HTTP/3 Server (UDP port 8443)** - Handles both regular HTTP requests and WebTransport connections
- **WebTransport Server** - Managed by zerohttp, started automatically when you call `app.ListenAndServeTLS()`

The integration is pluggable - zerohttp doesn't import webtransport-go directly. Instead, it defines
a `WebTransportServer` interface that `*webtransport.Server` automatically satisfies. This keeps
the dependency optional for users who don't need WebTransport.

### WebTransport Handler

```go
func handleSession(sess *webtransport.Session) {
    defer sess.CloseWithError(0, "done")

    // Handle datagrams in a goroutine
    go func() {
        for {
            msg, err := sess.ReceiveDatagram(context.Background())
            if err != nil {
                return
            }
            sess.SendDatagram(append([]byte("Echo: "), msg...))
        }
    }()

    // Handle bidirectional streams
    for {
        stream, err := sess.AcceptStream(context.Background())
        if err != nil {
            return
        }
        go func(str *webtransport.Stream) {
            defer str.Close()
            buf := make([]byte, 1024)
            for {
                n, err := str.Read(buf)
                if n > 0 {
                    msg := string(buf[:n])
                    response := fmt.Sprintf("[%s] Echo: %s",
                        time.Now().Format("15:04:05"), msg)
                    str.Write([]byte(response))
                }
                if err != nil {
                    return
                }
            }
        }(stream)
    }
}
```

### Key Integration Points

1. **Create HTTP/3 Server with TLS Config**
   ```go
   h3 := &http3.Server{
       Addr: ":8443",
       TLSConfig: &tls.Config{
           NextProtos: []string{"h3"},
       },
   }
   ```

2. **Create WebTransport Server and Configure HTTP/3**
   ```go
   wtServer := &webtransport.Server{
       H3:          h3,
       CheckOrigin: func(r *http.Request) bool { return true },
   }

   // REQUIRED: Wire WebTransport into HTTP/3
   webtransport.ConfigureHTTP3Server(h3)
   ```

3. **Create zerohttp App with Routes**
   ```go
   app := zh.New()

   // Regular HTTP endpoint
   app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
       w.Header().Set("Content-Type", "text/html")
       _, err := w.Write([]byte(html))
       return err
   }))

   // WebTransport endpoint
   app.CONNECT("/wt", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
       sess, err := wtServer.Upgrade(w, r)
       if err != nil {
           return err
       }
       go handleSession(sess)
       return nil
   }))
   ```

4. **Set Handler and Start Server**
   ```go
   // Set zerohttp as the HTTP/3 handler
   h3.Handler = app

   // Set the WebTransport server - zerohttp will start it automatically
   app.SetWebTransportServer(wtServer)

   // Just call app.ListenAndServeTLS - WebTransport starts automatically!
   app.ListenAndServeTLS(certFile, keyFile)
   ```

   **Note:** With zerohttp, you don't need to call `wtServer.ListenAndServeTLS()` directly.
   Just set the WebTransport server with `app.SetWebTransportServer()` and then call
   `app.ListenAndServeTLS()`. The WebTransport server will be started automatically.

## Production Setup (autotls.go)

For production with automatic Let's Encrypt certificates:

```bash
go run autotls.go -domain=your-domain.com
```

### Requirements

- A publicly accessible domain name
- Ports 80 and 443 open and accessible from the internet
- The domain must resolve to the server's IP address

### How It Works

The `autotls.go` example uses `golang.org/x/crypto/acme/autocert` for automatic certificate management:

1. **Certificate Acquisition**: On first start, the server obtains a certificate from Let's Encrypt
2. **Auto-Renewal**: Certificates are automatically renewed before expiry
3. **HTTP Challenge**: The ACME HTTP-01 challenge is handled on port 80
4. **WebTransport**: Runs on port 443 with the auto-obtained certificate

### Key Differences from Development Setup

The `autotls.go` example uses a wrapper to implement the `WebTransportServerWithAutocert` interface:

```go
// Wrapper implements config.WebTransportServerWithAutocert
type webtransportAutocertServer struct {
    server *webtransport.Server
}

func (w *webtransportAutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
    return w.server.ListenAndServeTLS(certFile, keyFile)
}

func (w *webtransportAutocertServer) Close() error {
    return w.server.Close()
}

func (w *webtransportAutocertServer) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
    // Configure TLS with autocert on the underlying HTTP/3 server
    w.server.H3.TLSConfig = &tls.Config{
        GetCertificate: manager.GetCertificate,
        NextProtos:     []string{"h3"},
    }
    return w.server.H3.ListenAndServe()
}
```

Then use the wrapper with zerohttp:

```go
// Create autocert manager
mgr := &autocert.Manager{
    Cache:      autocert.DirCache("/var/cache/certs"),
    Prompt:     autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("your-domain.com"),
}

// Create zerohttp app with autocert manager
app := zh.New(config.Config{
    AutocertManager: mgr,
})

// Create HTTP/3 and WebTransport servers
h3 := &http3.Server{Addr: ":443", Handler: app}
wt := &webtransport.Server{H3: h3, CheckOrigin: ...}
webtransport.ConfigureHTTP3Server(h3)

// Wrap the server to enable AutoTLS support
wtServer := &webtransportAutocertServer{server: wt}
app.SetWebTransportServer(wtServer)

// Start with AutoTLS - WebTransport starts automatically!
app.StartAutoTLS()
```

### Production Considerations

- Implement proper origin checking in `CheckOrigin` (don't allow all origins)
- Add rate limiting for WebTransport connections
- Consider connection limits per client
- Use structured logging for session events
- Store certificates in a persistent location (not `/tmp`)

## Resources

- [WebTransport Working Draft](https://w3c.github.io/webtransport/)
- [WebTransport over HTTP/3](https://datatracker.ietf.org/doc/html/draft-ietf-webtrans-http3/)
- [quic-go/webtransport-go](https://github.com/quic-go/webtransport-go)
