# Docker Example

This example demonstrates how to run a zerohttp application in Docker.

## Features

- Docker containerization
- Multi-stage build for minimal image size

## Running the Example

### Build the Docker image:

```bash
docker build -t zerohttp .
```

### Run the container:

```bash
docker run --name zerohttp -p 8080:8080 --rm zerohttp
```

The server starts on `http://localhost:8080`.

## Test Commands

```bash
curl http://localhost:8080/
```

Returns:
```json
{"message": "Hello, World!"}
```
