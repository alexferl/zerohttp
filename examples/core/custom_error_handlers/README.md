# Custom Error Handlers Example

This example demonstrates custom error handlers for 404 Not Found and 405 Method Not Allowed responses.

## Features

- Custom 404 Not Found handler
- Custom 405 Method Not Allowed handler

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint | Description       |
|----------|-------------------|
| `GET /`  | Hello world       |

## Test Commands

### Custom 404
```bash
curl http://localhost:8080/nonexistent
```

Returns custom message:
```
The page you're looking for has gone on vacation. Try a different path!
```

### Custom 405
```bash
curl -X POST http://localhost:8080/
```

Returns custom message:
```
That HTTP method isn't welcome here. Check the allowed methods and try again.
```
