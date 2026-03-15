# Validation Custom Example

This example demonstrates custom validation using Validate() methods and custom validators.

## Features

- Cross-field validation with Validate() method
- Custom validator registration
- Business logic validation

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint    | Description                    |
|--------|-------------|--------------------------------|
| `POST` | `/orders`   | Create order with cross-field validation |
| `POST` | `/register` | Register with custom password validator  |

## Test Commands

### Create order (valid - total matches items)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"550e8400-e29b-41d4-a716-446655440000","status":"pending","total":59.98,"items":[{"product_id":"550e8400-e29b-41d4-a716-446655440001","quantity":2,"price":29.99}]}'
```

### Create order (fails cross-field validation)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"550e8400-e29b-41d4-a716-446655440000","status":"pending","total":100.00,"items":[{"product_id":"550e8400-e29b-41d4-a716-446655440001","quantity":1,"price":29.99}]}'
```

### Register with strong password
```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"StrongP@ss123","confirm_password":"StrongP@ss123"}'
```

### Register (weak password fails)
```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"123","confirm_password":"123"}'
```
