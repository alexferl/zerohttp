# Validation Basic Example

This example demonstrates basic struct tag validation for JSON requests.

## Features

- Required field validation
- String length constraints (`min`, `max`)
- Email validation
- Alphanumeric validation
- Integer range validation
- Optional fields with `omitempty`

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method  | Endpoint       | Description                 |
|---------|----------------|-----------------------------|
| `POST`  | `/users`       | Create user with validation |
| `PATCH` | `/users/{id}`  | Update user (partial)       |

## Test Commands

### Create valid user
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","age":25,"username":"johndoe"}'
```

### Create user with validation errors
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"J","email":"bad-email","age":5,"username":"ab"}'
```

### Partial update (only name)
```bash
curl -X PATCH http://localhost:8080/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name":"Jane Doe"}'
```
