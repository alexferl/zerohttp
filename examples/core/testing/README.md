# Testing Example

Demonstrates testing zerohttp handlers using the `zhtest` package.

## Running

```bash
# Run the server
go run .

# Run tests
go test -v
```

## Features Demonstrated

- Request building with `zhtest.NewRequest()`
- JSON body assertions with `JSONPathEqual()`
- Status code assertions
- Header assertions using constants (`httpx.HeaderAccept`, `httpx.MIMEApplicationJSON`)
- Testing error responses (404, 422)

## Test Examples

```go
// Build request with JSON body
req := zhtest.NewRequest(http.MethodPost, "/users").
    WithHeader(zh.HeaderAccept, zh.MIMEApplicationJSON).
    WithJSON(map[string]string{"name": "Charlie", "email": "charlie@example.com"}).
    Build()

// Serve request
w := zhtest.Serve(app, req)

// Chain assertions
zhtest.AssertWith(t, w).
    Status(http.StatusCreated).
    HeaderContains(zh.HeaderContentType, zh.MIMEApplicationJSON).
    JSONPathEqual("name", "Charlie")
```
