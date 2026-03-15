# Template Example

This example demonstrates HTML template rendering using Go's standard `html/template` package with embedded template files.

## Features

- Embedded templates using `//go:embed`
- Standard Go html/template integration
- Custom 404 error page

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint | Description          |
|--------|----------|----------------------|
| `GET`  | `/`      | Home page            |
| any    | `/*`     | 404 error page       |

## Test Commands

### View the home page
```bash
curl http://localhost:8080
```

### View the 404 page
```bash
curl http://localhost:8080/not-found
```
