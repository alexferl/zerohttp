# WebTransport Example

WebTransport server with HTTP/3 using quic-go.

## Prerequisites

Generate self-signed certificates for localhost:

```bash
mkcert localhost
```

Or use any other tool to generate `localhost+2.pem` and `localhost+2-key.pem`.

## Running

```bash
go mod tidy
go run .
```

Open https://localhost:8443 in your browser.

## Endpoints

- `GET /` - Web UI
- `CONNECT /wt` - WebTransport endpoint

## Features

- HTTP/3 with WebTransport on same port
- Datagram and bidirectional stream support
- Echo server for messages
