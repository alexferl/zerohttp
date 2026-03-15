# Set Header Example

This example demonstrates setting custom response headers.

## Features

- Set global response headers
- Route-specific headers

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint   | Description                   |
|------------|-------------------------------|
| `GET /`    | Global headers only           |
| `GET /api` | Global + API-specific headers |

## Test Commands

### Check global headers
```bash
curl -I http://localhost:8080/
```

Shows:
```
X-Custom-Header: global-value
```

### Check combined headers
```bash
curl -I http://localhost:8080/api
```

Shows:
```
X-Custom-Header: global-value
X-API-Version: v1
```
