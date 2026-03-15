# File Upload Example

This example demonstrates file upload handling with multipart forms.

## Features

- Multipart form parsing with file binding
- Multiple file upload support
- File size limits (10 MB per file, 32 MB total)
- File download endpoint

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method   | Endpoint               | Description           |
|----------|------------------------|-----------------------|
| `GET`    | `/`                    | HTML upload form      |
| `POST`   | `/upload`              | Upload files          |
| `GET`    | `/files/{filename}`    | Download a file       |

## Test Commands

### Open the HTML form
Visit `http://localhost:8080` in your browser.

### Upload via curl
```bash
curl -X POST http://localhost:8080/upload \
  -F "files=@/path/to/file.txt" \
  -F "description=My file"
```

### Download a file
```bash
curl http://localhost:8080/files/{filename}
```
