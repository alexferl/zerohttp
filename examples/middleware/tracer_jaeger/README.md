# Jaeger Tracing Example

This example demonstrates distributed tracing with Jaeger using OpenTelemetry.

## Features

- Distributed tracing with Jaeger
- OpenTelemetry OTLP exporter
- Request span tracking

## Prerequisites

### Start Jaeger with Docker

```bash
# Start Jaeger container
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 14250:14250 \
  -p 14268:14268 \
  -p 14269:14269 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest

# Verify Jaeger is running
docker ps

# View Jaeger logs
docker logs jaeger
```

Access the Jaeger UI at http://localhost:16686

### Or use Docker Compose

```yaml
version: '3.8'
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "4318:4318"
```

```bash
docker-compose up -d
```

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

Then view traces at http://localhost:16686

Select `zerohttp-jaeger-example` from the Service dropdown.

## Cleanup

```bash
# Stop and remove Jaeger container
docker stop jaeger
docker rm jaeger
```
