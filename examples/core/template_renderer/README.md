# Template Renderer Example

This example demonstrates using zerohttp's `TemplateManager` for convenient template rendering with layout support.

## Features

- `TemplateManager` for simplified rendering
- Template layouts with `{{define}}` blocks
- Custom 404 error page

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint | Description      |
|--------|----------|------------------|
| `GET`  | `/`      | Home page        |
| `GET`  | `/about` | About page       |
| any    | `/*`     | 404 error page   |

## Template Structure

- `base.html` - Layout template with shared structure
- `index.html` - Content template that extends base
- `404.html` - Error page template

## Test Commands

### View home page
```bash
curl http://localhost:8080
```

### View about page
```bash
curl http://localhost:8080/about
```
