# Response Rendering Example

Demonstrates all response rendering methods in zerohttp.

## Running

```bash
go run .
```

## Endpoints

- `GET /json` - JSON response with structured data
- `GET /text` - Plain text response
- `GET /html` - HTML response
- `GET /blob` - Binary data (simulated PNG)
- `GET /stream` - Streaming response
- `GET /file` - File download
- `GET /error` - RFC 9457 Problem Detail response

## Available Renderers

| Method                 | Description     | Content-Type               |
|------------------------|-----------------|----------------------------|
| `zh.R.JSON()`          | JSON encoding   | `application/json`         |
| `zh.R.Text()`          | Plain text      | `text/plain`               |
| `zh.R.HTML()`          | HTML content    | `text/html`                |
| `zh.R.Blob()`          | Binary data     | Specified                  |
| `zh.R.Stream()`        | Streaming I/O   | Specified                  |
| `zh.R.File()`          | File serving    | Auto-detected              |
| `zh.R.ProblemDetail()` | RFC 9457 errors | `application/problem+json` |

Use `zh.Render` or the short alias `zh.R`.
