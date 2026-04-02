# Pagination Example

This example demonstrates pagination with the pagination package.

## Features

- Offset-based pagination with standardized response headers
- Query parameter binding for page and per_page
- Search functionality with pagination
- RFC 5988 Link headers for navigation

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint              | Description                        |
|--------|-----------------------|------------------------------------|
| `GET`  | `/`                   | Index page with documentation      |
| `GET`  | `/products`           | List products (paginated)          |
| `GET`  | `/products/search`    | Search products with pagination    |

## Response Headers

The following headers are included in paginated responses:

| Header            | Description                          |
|-------------------|--------------------------------------|
| `X-Total`         | Total number of items available      |
| `X-Total-Pages`   | Total number of pages                |
| `X-Page`          | Current page number                  |
| `X-Per-Page`      | Items per page                       |
| `X-Prev-Page`     | Previous page number (if available)  |
| `X-Next-Page`     | Next page number (if available)      |
| `Link`            | RFC 5988 Link header                 |

## Test Commands

### List products (default pagination)
```bash
curl -i "http://localhost:8080/products"
```

### Get page 2 with 10 items per page
```bash
curl -i "http://localhost:8080/products?page=2&per_page=10"
```

### Search products with pagination
```bash
curl -i "http://localhost:8080/products/search?q=product%201"
```

### Search with custom page size
```bash
curl -i "http://localhost:8080/products/search?q=product&page=1&per_page=5"
```
