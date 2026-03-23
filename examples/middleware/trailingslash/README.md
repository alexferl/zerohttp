# Trailing Slash Example

This example demonstrates trailing slash normalization.

## Features

- Strip trailing slashes
- Append trailing slashes

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint         | Middleware | Handler Receives |
|------------------|------------|------------------|
| `GET /api/users` | Strip      | `/api/users`     |
| `GET /docs`      | Append     | `/docs/`         |

## Test Commands

```bash
# Strip: /api/users/ becomes /api/users internally
curl http://localhost:8080/api/users/

# Append: /docs becomes /docs/ internally
curl http://localhost:8080/docs
```

## Actions

- **strip**: Remove trailing slash, continue processing
- **append**: Add trailing slash, continue processing
- **redirect**: 301 redirect to canonical URL
