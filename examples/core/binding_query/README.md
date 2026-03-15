# Query Parameter Binding Example

This example demonstrates query parameter binding to structs.

## Features

- Struct binding with `query` tags (`zh.B.Query()`)
- Individual parameter extraction (`zh.QueryParamAs[T]()`)
- Default values (`zh.QueryParamAsOrDefault()`)
- Slice binding for multiple values
- Optional parameters using pointer types
- Type conversion (string to int, float, bool)
- Embedded struct support

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint    | Description                       |
|--------|-------------|-----------------------------------|
| `GET`  | `/`         | Overview page with examples       |
| `GET`  | `/search`   | Basic search with strings/slices  |
| `GET`  | `/items`    | Pagination with optional boolean  |
| `GET`  | `/products` | Filter with int slices and floats |
| `GET`  | `/users`    | Embedded struct binding           |
| `GET`  | `/extract`  | Individual parameter extraction   |

## Test Commands

### Basic search
```bash
curl "http://localhost:8080/search?q=golang&category=tech&tags=api&tags=web"
```

### Pagination with optional boolean
```bash
curl "http://localhost:8080/items?page=2&limit=50&is_admin=true"
```

### Filter with int slices
```bash
curl "http://localhost:8080/products?id=1&id=2&status=active&min_price=10.99"
```

### Embedded struct pagination
```bash
curl "http://localhost:8080/users?search=john&page=1&limit=20"
```

### Individual parameter extraction
```bash
curl "http://localhost:8080/extract?user_id=123&active=true&score=95.5"
```
