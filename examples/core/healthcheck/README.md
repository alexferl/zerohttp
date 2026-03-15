# Health Check Example

This example demonstrates Kubernetes-style health check endpoints.

## Features

- Liveness probe endpoint
- Readiness probe endpoint
- Startup probe endpoint
- Custom readiness handler

## Default Endpoints

When using `healthcheck.DefaultConfig`:

| Endpoint    | Description       |
|-------------|-------------------|
| `/healthz`  | Liveness check    |
| `/readyz`   | Readiness check   |
| `/startupz` | Startup probe     |

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint          | Description        |
|--------|-------------------|--------------------|
| `GET`  | `/health/live`    | Liveness check     |
| `GET`  | `/health/ready`   | Readiness check    |
| `GET`  | `/health/startup` | Startup probe      |

## Test Commands

### Liveness check
```bash
curl http://localhost:8080/health/live
```

### Readiness check
```bash
curl http://localhost:8080/health/ready
```

### Startup probe
```bash
curl http://localhost:8080/health/startup
```

## Kubernetes Usage

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5

startupProbe:
  httpGet:
    path: /health/startup
    port: 8080
  failureThreshold: 30
  periodSeconds: 10
```
