# SSE Example

Server-Sent Events example with replay and broadcast support.

## Running

```bash
go run .
```

Open http://localhost:8080

## Endpoints

- `GET /` - Web UI
- `GET /time` - Time stream
- `GET /notifications` - Notifications with replay
- `GET /broadcast` - Broadcast channel
- `POST /notify?msg=` - Store notification
- `POST /broadcast?msg=` - Broadcast message
