# SPA (Single Page Application) Example

This example demonstrates serving a Single Page Application with embedded static files.

## Features

- Embedded static files using `embed`
- SPA mode (serves index.html for all non-API routes)
- API routes alongside static files

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Files

The `dist/` directory contains the built SPA:

```
dist/
├── index.html
├── favicon.ico
└── assets/
    ├── app.js
    └── style.css
```

## Endpoints

| Method | Endpoint      | Description                |
|--------|---------------|----------------------------|
| `GET`  | `/`           | SPA (index.html)           |
| `GET`  | `/api/health` | API health check           |
| `GET`  | `/*`          | SPA fallback (index.html)  |

## Test Commands

### View the SPA
```bash
# Opens the SPA
curl http://localhost:8080/
```

### API endpoint
```bash
curl http://localhost:8080/api/health
```

### Static assets
```bash
curl http://localhost:8080/assets/style.css
curl http://localhost:8080/assets/app.js
```

## SPA Behavior

When `spaMode` is `true` (third argument to `Static`):
- All routes that don't match API routes serve `index.html`
- This enables client-side routing (React Router, Vue Router, etc.)
- The client-side router handles the URL, not the server
