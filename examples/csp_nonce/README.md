# CSP Nonce Example

This example demonstrates Content Security Policy (CSP) nonce generation with zerohttp.

## What is a CSP Nonce?

A CSP nonce is a cryptographically random value that allows specific inline scripts and styles to execute while blocking all others. The nonce is:

1. Generated uniquely for each request
2. Added to the CSP header (`script-src 'nonce-abc123'`)
3. Added to inline script tags (`<script nonce="abc123">`)
4. Validated by the browser - only scripts with matching nonces execute

## Why Use CSP Nonces?

- **XSS Protection**: Blocks unauthorized inline scripts
- **Flexibility**: Allows trusted inline scripts without using `'unsafe-inline'`
- **Per-Request Unique**: Makes exploitation harder as nonces change every request

## Running the Example

```bash
go run main.go
```

Visit http://localhost:8080 to see the demo.

## How It Works

### 1. Enable CSP Nonce Generation

```go
app.Use(middleware.SecurityHeaders(config.SecurityHeadersConfig{
    CSPNonceEnabled: true,
    ContentSecurityPolicy: "script-src 'nonce-{{nonce}}'; style-src 'nonce-{{nonce}}'",
}))
```

### 2. Get the Nonce in Your Handler

```go
nonce := middleware.GetCSPNonce(r)
```

### 3. Inject into Your HTML

```html
<script nonce="{{.Nonce}}">
    // This script is allowed
</script>
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `CSPNonceEnabled` | Enable nonce generation | `false` |
| `CSPNonceContextKey` | Custom context key | `"csp_nonce"` |
| `ContentSecurityPolicy` | CSP header with `{{nonce}}` placeholder | - |

## Using with Templates

For template rendering, pass the nonce to your template data:

```go
type PageData struct {
    Title string
    Nonce string
}

data := PageData{
    Title: "My Page",
    Nonce: middleware.GetCSPNonce(r),
}
return tmpl.Execute(w, data)
```

Then in your template:
```html
<script nonce="{{.Nonce}}">
    console.log("Script with nonce executes");
</script>
```

## Report-Only Mode

Test your CSP without blocking:

```go
SecurityHeaders(config.SecurityHeadersConfig{
    CSPNonceEnabled: true,
    ContentSecurityPolicy: "script-src 'nonce-{{nonce}}'",
    ContentSecurityPolicyReportOnly: true,
})
```
