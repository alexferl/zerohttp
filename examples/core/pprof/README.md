# pprof Example

This example demonstrates Go's pprof profiling integration with zerohttp for runtime profiling and debugging.

## Features

- Automatic secure password generation
- Basic authentication protection
- IP allowlist (localhost-only by default)
- CPU, heap, goroutine, and other profile types
- Trace and execution analysis

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

All pprof endpoints are available at `/debug/pprof/`:

| Endpoint                        | Description                     |
|---------------------------------|---------------------------------|
| `GET /debug/pprof/`             | Index page listing all profiles |
| `GET /debug/pprof/cmdline`      | Command line arguments          |
| `GET /debug/pprof/profile`      | CPU profile (30s sampling)      |
| `GET /debug/pprof/symbol`       | Symbol lookup                   |
| `GET /debug/pprof/trace`        | Execution trace                 |
| `GET /debug/pprof/heap`         | Heap memory profile             |
| `GET /debug/pprof/goroutine`    | Goroutine stack dump            |
| `GET /debug/pprof/threadcreate` | OS thread creation profile      |
| `GET /debug/pprof/block`        | Blocking operations profile     |
| `GET /debug/pprof/mutex`        | Mutex contention profile        |

## Test Commands

### View the pprof index page
```bash
curl -u pprof:<password-from-logs> http://localhost:8080/debug/pprof/
```

### Capture a 5-second CPU profile
```bash
curl -u pprof:<password> http://localhost:8080/debug/pprof/profile?seconds=5 -o cpu.prof
```

### Get heap memory profile
```bash
curl -u pprof:<password> http://localhost:8080/debug/pprof/heap -o heap.prof
```

### View goroutine stack dump
```bash
curl -u pprof:<password> http://localhost:8080/debug/pprof/goroutine?debug=1
```

### Analyze profiles with Go's pprof tool
```bash
go tool pprof -http=:8081 cpu.prof
go tool pprof -http=:8081 heap.prof
```

## Configuration Examples

### Custom prefix and credentials
```go
cfg := pprof.DefaultConfig
cfg.Prefix = "/admin/pprof"
cfg.Auth = &pprof.AuthConfig{
    Username: "admin",
    Password: "secret",
}
pp := pprof.New(app, cfg)
```

### Allow specific IP ranges
```go
cfg := pprof.DefaultConfig
cfg.AllowedIPs = []string{"10.0.0.0/8", "192.168.1.100"}
pp := pprof.New(app, cfg)
```

### Disable specific profiles
```go
cfg := pprof.DefaultConfig
cfg.EnableBlock = false
cfg.EnableMutex = false
pp := pprof.New(app, cfg)
```

## Security Notes

- pprof endpoints are **localhost-only** by default
- A random secure password is auto-generated when no auth config is provided
- Always use authentication in production environments
- Consider disabling pprof entirely in production if not needed
