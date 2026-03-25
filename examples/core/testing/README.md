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
- Header assertions using constants (`zh.HeaderAccept`, `zh.MIMEApplicationJSON`)
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

## Response Assertions

Chain HTTP response assertions:

- `Status(code)` - Exact status code
- `IsSuccess()` - 2xx status
- `IsClientError()` - 4xx status
- `Header(key, value)` - Exact header match
- `HeaderContains(key, substring)` - Partial header match
- `JSONPathEqual(path, value)` - JSON path value
- `BodyContains(substring)` - Body content

## General Assertions

Standalone assertion functions for general test use:

### Error Assertions

```go
zhtest.AssertNoError(t, err)
zhtest.AssertError(t, err)
zhtest.AssertErrorIs(t, err, os.ErrNotExist)
zhtest.AssertErrorContains(t, err, "connection refused")
```

### Nil/NotNil Assertions

```go
zhtest.AssertNil(t, ptr)
zhtest.AssertNotNil(t, result)
```

### Equality Assertions

```go
zhtest.AssertEqual(t, 42, result)
zhtest.AssertNotEqual(t, "old", result)
zhtest.AssertDeepEqual(t, []int{1, 2, 3}, result)
```

### Boolean Assertions

```go
zhtest.AssertTrue(t, len(items) > 0)
zhtest.AssertFalse(t, len(items) == 0)
```

### Empty/NotEmpty Assertions

```go
zhtest.AssertEmpty(t, "")
zhtest.AssertEmpty(t, []int{})
zhtest.AssertNotEmpty(t, "hello")
```

### Collection Assertions

```go
zhtest.AssertLen(t, []int{1, 2, 3}, 3)
zhtest.AssertContains(t, []int{1, 2, 3}, 2)
zhtest.AssertNotContains(t, []int{1, 2, 3}, 4)
```

### Type Assertions

```go
zhtest.AssertIsType(t, (*MyError)(nil), err)
zhtest.AssertImplements(t, (*io.Reader)(nil), myReader)
```
