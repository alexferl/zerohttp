# CRUD Example

This example demonstrates a RESTful CRUD API for managing users.

## Features

- In-memory user store with thread-safe operations
- Full CRUD endpoints (Create, Read, Update, Delete)
- JSON request/response handling

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method   | Endpoint       | Description        |
|----------|----------------|--------------------|
| `GET`    | `/users`       | List all users     |
| `GET`    | `/users/{id}`  | Get user by ID     |
| `POST`   | `/users`       | Create a user      |
| `PUT`    | `/users/{id}`  | Update a user      |
| `DELETE` | `/users/{id}`  | Delete a user      |

## Test Commands

### List users
```bash
curl http://localhost:8080/users
```

### Get user by ID
```bash
curl http://localhost:8080/users/1
```

### Create user
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Charlie", "age": 35}'
```

### Update user
```bash
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Smith", "age": 31}'
```

### Delete user
```bash
curl -X DELETE http://localhost:8080/users/1
```
