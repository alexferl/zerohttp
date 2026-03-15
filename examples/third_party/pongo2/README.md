# Pongo2 Template Engine Example

This example demonstrates how to integrate [Pongo2](https://github.com/flosch/pongo2) (a Django-syntax template engine for Go) with zerohttp.

Pongo2 brings the familiar Django/Jinja2 template syntax to Go, with features like template inheritance, filters, tags, and more.

## Features

- Django/Jinja2-style template syntax
- Template inheritance with `extends` and `block`
- Embedded templates using Go's `embed` package
- Custom 404 error page rendering

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## API Endpoints

### Home page

```bash
curl http://localhost:8080/
```

Response (HTML):
```html
<!DOCTYPE html>
<html>
<head>
    <title>Welcome</title>
</head>
<body>
    <h1>Hello from Pongo2!</h1>
    <p>This is a zerohttp example using Pongo2 templates.</p>
</body>
</html>
```

### About page

```bash
curl http://localhost:8080/about
```

### Custom 404 page

```bash
curl http://localhost:8080/nonexistent
```

Returns a styled 404 page rendered with Pongo2.
