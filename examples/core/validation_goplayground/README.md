# Validation with go-playground Example

This example demonstrates using go-playground/validator as a custom validator with zerohttp.

## Features

- Custom validator integration
- go-playground validation tags
- Error format conversion

## Running the Example

```bash
go mod tidy
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint | Description                    |
|--------|----------|--------------------------------|
| `POST` | `/users` | Create user with validation    |

## Test Commands

### Valid request
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John","email":"john@example.com","age":25,"username":"johndoe"}'
```

### Validation errors
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"J","email":"bad","age":5,"username":"ab"}'
```
