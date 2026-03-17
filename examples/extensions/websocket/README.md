# WebSocket Example

This example demonstrates a WebSocket server using gorilla/websocket with zerohttp.

## Features

- WebSocket upgrade handling
- Message echo server
- Web UI for testing

## Prerequisites

- The `github.com/gorilla/websocket` package:
  ```bash
  go get github.com/gorilla/websocket
  ```

## Running the Example

```bash
go mod tidy
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint    | Description           |
|-------------|-----------------------|
| `GET /`     | Web UI                |
| `GET /ws`   | WebSocket endpoint    |

## Test Commands

Open `http://localhost:8080` in your browser to use the Web UI, or use a WebSocket client:

```bash
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Key: $(openssl rand -base64 16)" \
  -H "Sec-WebSocket-Version: 13" \
  http://localhost:8080/ws
```
