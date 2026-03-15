# Validation Nested Example

This example demonstrates validation of nested structs and slice elements.

## Features

- Nested struct validation
- Slice element validation with `each` tag
- Conditional validation based on field values

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint         | Description                  |
|--------|------------------|------------------------------|
| `POST` | `/bulk`          | Bulk create with slice validation |
| `POST` | `/organizations` | Deep nested struct validation     |

## Test Commands

### Bulk create (valid)
```bash
curl -X POST http://localhost:8080/bulk \
  -H "Content-Type: application/json" \
  -d '{"tags":["golang","api"],"recipients":["admin@example.com"],"products":[{"sku":"PROD123456","name":"Product One","price":29.99}]}'
```

### Bulk create (fails each validation)
```bash
curl -X POST http://localhost:8080/bulk \
  -H "Content-Type: application/json" \
  -d '{"tags":["a","way-too-long"],"products":[{"sku":"prod123","name":"X","price":0}]}'
```

### Create organization
```bash
curl -X POST http://localhost:8080/organizations \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp","slug":"acme-corp","owner":{"name":"John","email":"john@example.com","role":"admin"},"billing":{"plan":"pro","card_token":"tok_visa"}}'
```

### Create organization (fails billing validation)
```bash
curl -X POST http://localhost:8080/organizations \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Corp","slug":"acme-corp","owner":{"name":"John","email":"john@example.com","role":"admin"},"billing":{"plan":"pro"}}'
```
