# File Server Example

This example demonstrates serving static files using zerohttp.

## Features

- Serve embedded files from the binary (faster - files are in memory)
- Serve files from a directory

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint          | Description                       |
|-------------------|-----------------------------------|
| `GET /api/health` | Health check                      |
| `GET /static/`    | Serve embedded static files       |
| `GET /uploads/`   | Serve files from ./upload_dir dir |

## Test Commands

### Health check
```bash
curl http://localhost:8080/api/health
```

### Get an embedded static file
```bash
curl http://localhost:8080/static/hello.txt
```

### Get a file from uploads directory
```bash
curl http://localhost:8080/uploads/hello.txt
```
