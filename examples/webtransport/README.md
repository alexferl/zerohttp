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

## Setup

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
- The `webtransport.Server` manages the HTTP/3 lifecycle

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

3. **Register Handlers and Start Server**
   ```go
   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
       w.Header().Set("Content-Type", "text/html")
       w.Write([]byte(html))
   })

   http.HandleFunc("/wt", func(w http.ResponseWriter, r *http.Request) {
       sess, err := wtServer.Upgrade(w, r)
       if err != nil {
           w.WriteHeader(http.StatusInternalServerError)
           return
       }
       go handleSession(sess)
   })

   wtServer.ListenAndServeTLS(certFile, keyFile)
   ```

## Production Considerations

- Replace mkcert certificates with proper ones from Let's Encrypt
- Implement proper origin checking in `CheckOrigin`
- Add rate limiting for WebTransport connections
- Consider connection limits per client
- Use structured logging for session events

## Resources

- [WebTransport Working Draft](https://w3c.github.io/webtransport/)
- [WebTransport over HTTP/3](https://datatracker.ietf.org/doc/html/draft-ietf-webtrans-http3/)
- [quic-go/webtransport-go](https://github.com/quic-go/webtransport-go)
