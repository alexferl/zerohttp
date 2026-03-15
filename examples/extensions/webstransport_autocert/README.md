# WebTransport with AutoTLS Example

WebTransport server with automatic Let's Encrypt certificates.

## Prerequisites

- Public domain pointing to your server
- Ports 80 and 443 open

## Running

```bash
go mod tidy
go run . -domain example.com
```

## Endpoints

- `GET /` - Web UI
- `CONNECT /wt` - WebTransport endpoint

## Features

- Automatic TLS via Let's Encrypt
- HTTP/3 and WebTransport on same port
- Datagram and stream support
