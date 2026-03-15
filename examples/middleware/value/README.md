# Value Example

This example demonstrates passing values through request context.

## Features

- Store values in request context
- Type-safe value retrieval
- Route-specific values

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint      | Description                   |
|---------------|-------------------------------|
| `GET /user`   | Returns values from context   |

## Test Commands

```bash
curl http://localhost:8080/user
```

Response:
```json
{
  "user_id": 123,
  "role": "admin"
}
```
