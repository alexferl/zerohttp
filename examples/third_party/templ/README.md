# Templ Integration Example

This example demonstrates how to integrate [Templ](https://github.com/a-h/templ) (a type-safe HTML templating library for Go) with zerohttp.

Templ provides compile-time type safety for HTML templates, with components written as Go functions.

## Features

- Type-safe HTML component rendering
- Component factory pattern for dynamic data
- Custom 404 error page with Templ
- CSP (Content Security Policy) configuration

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint       | Description                                           |
|----------------|-------------------------------------------------------|
| `GET /`        | Home page with welcome message                        |
| `GET /about`   | About page (reuses home template with different data) |
| Any other path | Custom 404 error page                                 |

## Project Structure

- `main.go` - Server setup and route handlers
- `template_manager.go` - Wrapper for Templ component registration and rendering
- `components_templ.go` - Generated Templ components (from `.templ` files)

## How It Works

Components are registered by name with a factory function:

```go
tm.RegisterComponent("home", ComponentFactory(HomePage))
```

Data is passed to components through a type-safe interface:

```go
data := PageData{
    Message:     "Hello from Templ!",
    Description: "This is a zerohttp example using Templ components.",
}
return tm.Render(w, r, http.StatusOK, "home", data)
```

The `TemplTemplateManager` handles:
- Component lookup by name
- Setting Content-Type header
- Rendering the component to the response writer

## Custom 404 Page

The example includes a custom 404 handler that renders a Templ component:

```go
app.NotFound(zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    data := PageData{
        Message:     "Page Not Found",
        Description: "The page you requested could not be found.",
    }
    return tm.Render(w, r, http.StatusNotFound, "404", data)
}))
```

## Templ Component Example

Templ components are Go functions that return `templ.Component`:

```go
func HomePage(message, description string) templ.Component {
    return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
        // Type-safe HTML rendering
    })
}
```

Components are typically defined in `.templ` files and compiled to Go code using the `templ` CLI.
