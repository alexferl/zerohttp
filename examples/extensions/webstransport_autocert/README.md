# WebTransport with AutoTLS Example

This example demonstrates a WebTransport server with automatic Let's Encrypt certificates.

## Features

- WebTransport server with HTTP/3
- Automatic TLS via Let's Encrypt
- Datagram and stream support
- HTTP/3 and WebTransport on same port

## Prerequisites

- A publicly accessible server with a domain name
- Ports 80 and 443 open

## Running the Example

```bash
go mod tidy
go run . -domain example.com
```

The server starts on port 443 with auto-provisioned certificates.

## Endpoints

| Endpoint      | Description            |
|---------------|------------------------|
| `GET /`       | Web UI                 |
| `CONNECT /wt` | WebTransport endpoint  |

## Test Commands

Open `https://your-domain.com` in your browser to use the Web UI.
