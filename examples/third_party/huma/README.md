# Huma Integration Example

This example demonstrates how to integrate [Huma](https://github.com/danielgtaylor/huma) (an OpenAPI 3.1 framework) with zerohttp.

Huma provides automatic OpenAPI spec generation, request/response validation, and documentation UI. This example shows how to create a Huma adapter that works with zerohttp's router.

## Features

- OpenAPI 3.1 spec generation from Go structs and tags
- Automatic request/response validation
- Type-safe handlers with input/output structs
- Zerohttp native error handling integration

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## API Endpoints

### Standard zerohttp endpoint

```bash
curl http://localhost:8080/
```

Response:
```json
{"message": "Hello, World!"}
```

### Huma OpenAPI endpoint

```bash
# Get a greeting
curl http://localhost:8080/greeting/John
```

Response:
```json
{"message": "Hello, John!"}
```

```bash
# Try with a different name
curl http://localhost:8080/greeting/World
```

Response:
```json
{"message": "Hello, World!"}
```

### OpenAPI Documentation

Huma auto-generates OpenAPI spec and documentation:

```bash
# Get OpenAPI JSON spec
curl http://localhost:8080/openapi.json

# Get OpenAPI YAML spec
curl http://localhost:8080/openapi.yaml

# Swagger UI (if configured)
curl http://localhost:8080/docs
```
