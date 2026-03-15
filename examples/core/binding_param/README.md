# Path Parameter Binding Example

This example demonstrates path parameter extraction from URLs.

## Features

- String parameter extraction (`zh.Param()`)
- Typed parameter extraction (`zh.ParamAs[T]()`)
- Default values for parameters (`zh.ParamOrDefault()`)
- Multiple path parameters

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint                          | Description                     |
|--------|-----------------------------------|---------------------------------|
| `GET`  | `/users/{id}`                     | String parameter extraction     |
| `GET`  | `/users/{userID}/posts/{postID}`  | Multiple parameters             |
| `GET`  | `/items/{itemID}`                 | Typed int extraction            |
| `GET`  | `/products/{$}`                   | All products (no param)         |
| `GET`  | `/products/{category}`            | Filtered by category            |

## Test Commands

### String parameter
```bash
curl http://localhost:8080/users/123
```

### Multiple parameters
```bash
curl http://localhost:8080/users/42/posts/99
```

### Typed parameter (int conversion)
```bash
curl http://localhost:8080/items/456
```

### No path parameter (all products)
```bash
curl http://localhost:8080/products/
```

### Optional parameter with value
```bash
curl http://localhost:8080/products/electronics
```
