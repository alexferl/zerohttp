# WebTransport Example

This example demonstrates a WebTransport server with HTTP/3 using quic-go.

## Features

- HTTP/3 with WebTransport on same port
- Datagram and bidirectional stream support
- Echo server for messages

## Prerequisites

Generate self-signed certificates for localhost:

```bash
mkcert localhost
```

Or use any other tool to generate `localhost+2.pem` and `localhost+2-key.pem`.

## Running the Example

```bash
go mod tidy
go run .
```

The server starts on `https://localhost:8443`.

## Endpoints

| Endpoint      | Description            |
|---------------|------------------------|
| `GET /`       | Web UI                 |
| `CONNECT /wt` | WebTransport endpoint  |

## Test Commands

Open `https://localhost:8443` in your browser to use the Web UI.
