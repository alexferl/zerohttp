# Form Binding Example

This example demonstrates form and multipart form binding.

## Features

- URL-encoded form binding (`application/x-www-form-urlencoded`)
- Multipart form binding with file uploads (`multipart/form-data`)
- Query parameter binding
- Type conversion (string to int, bool)
- Slice binding for multiple values

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method   | Endpoint     | Description                          |
|----------|--------------|--------------------------------------|
| `GET`    | `/`          | Overview page with examples          |
| `POST`   | `/login`     | Simple form binding                  |
| `GET`    | `/search`    | Query parameter binding with slices  |
| `GET`    | `/contact`   | Contact form page (HTML)             |
| `POST`   | `/contact`   | Multipart form with single file      |
| `GET`    | `/upload`    | Multi-file upload page (HTML)        |
| `POST`   | `/upload`    | Multipart form with multiple files   |

## Test Commands

### Simple form binding
```bash
curl -X POST http://localhost:8080/login \
  -d "username=johndoe" \
  -d "password=secret" \
  -d "remember=true"
```

### Query binding with slices
```bash
curl "http://localhost:8080/search?q=golang&category=tech&tags=api&tags=web"
```

### Multipart form with file
```bash
curl -X POST http://localhost:8080/contact \
  -F "name=John Doe" \
  -F "email=john@example.com" \
  -F "avatar=@/path/to/avatar.png"
```

### Multiple file upload
```bash
curl -X POST http://localhost:8080/upload \
  -F "description=My documents" \
  -F "documents=@file1.pdf" \
  -F "documents=@file2.pdf"
```
