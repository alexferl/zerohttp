# Static Website Example

This example demonstrates serving static files from an embedded filesystem with SPA-style fallback to index.html.

## Features

- Static file serving using `embed.FS`
- Fallback to index.html for missing files (SPA behavior)
- Files served from `public/` directory

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint | Description                        |
|--------|----------|------------------------------------|
| `GET`  | `/`      | Serves `public/index.html`         |
| `GET`  | `/*`     | Serves files or falls back to SPA  |

## Test Commands

### View the index page
```bash
curl http://localhost:8080
```

### Request a specific file
```bash
curl http://localhost:8080/index.html
```

### Missing files fallback to index.html (SPA behavior)
```bash
curl http://localhost:8080/not-found
```
