# OpenTelemetry Tracing Example

This example demonstrates request tracing with OpenTelemetry using a stdout exporter.

## Features

- OpenTelemetry trace collection
- Stdout exporter (prints traces to console)
- Request span tracking

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint     | Description            |
|--------------|------------------------|
| `GET /`      | Successful request     |
| `GET /error` | Request with error     |

## Test Commands

```bash
curl http://localhost:8080/
curl http://localhost:8080/error
```

Watch the console for OpenTelemetry trace output in JSON format.

## Production Exporters

Replace `stdouttrace` with production exporters:

- **Jaeger**: `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`
- **Zipkin**: `go.opentelemetry.io/otel/exporters/zipkin`
- **OTLP/gRPC**: `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
