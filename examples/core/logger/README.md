# Logger Example

This example demonstrates how to use log level filtering with zerohttp's built-in logger.

## Running

```bash
go run main.go
```

## Endpoints

### GET /demo

Logs messages at all levels (Debug, Info, Warn, Error) since the logger is set to `DebugLevel`.

```bash
curl http://localhost:8080/demo
```

Expected output in console:
```
[DBG] This is a debug message
[INF] This is an info message
[WRN] This is a warning message
[ERR] This is an error message
```

### GET /filter

Demonstrates log filtering by changing the level to `WarnLevel`. Only Warn and Error messages are logged.

```bash
curl http://localhost:8080/filter
```

Expected output in console:
```
[WRN] This warning message WILL be logged
[ERR] This error message WILL be logged
```

## Available Log Levels

| Level      | Value | Description                    |
|------------|-------|--------------------------------|
| DebugLevel | 0     | Most verbose - logs everything |
| InfoLevel  | 1     | Default - logs Info and above  |
| WarnLevel  | 2     | Logs warnings and errors       |
| ErrorLevel | 3     | Logs only errors               |
| PanicLevel | 4     | Used for panic messages        |
| FatalLevel | 5     | Used for fatal messages        |

## Usage

```go
logger := log.NewDefaultLogger()
logger.SetLevel(log.DebugLevel)  // Show all messages

app := zh.New(config.Config{
    Logger: logger,
})
```

To change the level at runtime:

```go
app.Logger().(*log.DefaultLogger).SetLevel(log.WarnLevel)
```
