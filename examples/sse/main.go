// This example demonstrates how to use Server-Sent Events (SSE) with zerohttp.
// It includes:
//   - Basic SSE streaming (time, counter, notifications)
//   - Event replay (reconnect and resume from last event ID)
//   - Broadcast hub (fan-out messages to multiple clients)
//
// To run this example:
//   1. go run examples/sse/main.go
//   2. Open http://localhost:8080 in your browser

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

func main() {
	// Create a custom HTTP server without write timeout for SSE support
	customServer := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // No write timeout for SSE
		IdleTimeout:  120 * time.Second,
	}

	// Create replayer for event replay (keep last 100 events, 5 min TTL)
	replayer := zh.NewInMemoryReplayer(100, 5*time.Minute)

	// Create broadcast hub for fan-out
	hub := zh.NewSSEHub()
	broadcastCounter := int32(0)

	// Create zerohttp server
	app := zh.New(
		config.Config{
			Server:                    customServer,
			DisableDefaultMiddlewares: true,
			SSEProvider:               zh.NewDefaultProvider(),
		},
	)

	// Serve HTML client
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte(indexHTML))
		return err
	}))

	// SSE endpoint - time stream (basic)
	app.GET("/time", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		provider := app.SSEProvider()
		if provider == nil {
			return fmt.Errorf("SSE not configured")
		}

		stream, err := provider.NewSSE(w, r)
		if err != nil {
			return err
		}
		defer func() { _ = stream.Close() }()

		_ = stream.SetRetry(5 * time.Second)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				log.Println("Time stream: Client disconnected")
				return nil
			case t := <-ticker.C:
				event := zh.SSEEvent{
					Name: "time",
					Data: []byte(t.Format(time.RFC3339)),
					ID:   fmt.Sprintf("%d", t.Unix()),
				}
				if err := stream.Send(event); err != nil {
					return nil
				}
			}
		}
	}))

	// SSE endpoint - notifications with replay support
	// Clients can reconnect and resume from their last event ID
	app.GET("/notifications", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Get Last-Event-ID from header (browser auto-sends) or query param (manual reconnect)
		lastEventID := r.Header.Get("Last-Event-ID")
		if lastEventID == "" {
			lastEventID = r.URL.Query().Get("last_id")
		}

		// Create SSE connection
		stream, err := zh.NewSSE(w, r)
		if err != nil {
			return err
		}
		defer func() { _ = stream.Close() }()

		// Replay missed events if last_id is provided
		if lastEventID != "" {
			count, err := replayer.Replay(lastEventID, func(event zh.SSEEvent) error {
				return stream.Send(event)
			})
			if err != nil {
				log.Printf("Replay error: %v", err)
			} else {
				log.Printf("Replayed %d events", count)
				_ = stream.Send(zh.SSEEvent{
					Name: "info",
					Data: []byte(fmt.Sprintf("Replayed %d missed events", count)),
				})
			}
		}

		// Subscribe to hub for real-time notifications
		hub.Subscribe(stream, "notifications")
		defer hub.Unsubscribe(stream, "notifications")

		log.Printf("Notifications: Client connected (Last-Event-ID: %s)", lastEventID)

		// Send a welcome message
		_ = stream.Send(zh.SSEEvent{
			Name: "info",
			Data: []byte("Connected! Auto-generated events arrive every 10 seconds."),
		})

		// Keep connection alive and wait for context cancellation
		<-r.Context().Done()
		log.Println("Notifications: Client disconnected")
		return nil
	}))

	// SSE endpoint - broadcast subscriber
	app.GET("/broadcast", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Create SSE connection directly for hub compatibility
		stream, err := zh.NewSSE(w, r)
		if err != nil {
			return err
		}
		defer func() { _ = stream.Close() }()

		// Register with hub for broadcast
		hub.Register(stream)
		defer hub.Unregister(stream)

		log.Println("Broadcast: Client subscribed")

		// Send initial welcome
		_ = stream.Send(zh.SSEEvent{
			Name: "info",
			Data: []byte("Subscribed to broadcast channel"),
		})

		// Keep connection alive
		<-r.Context().Done()
		log.Println("Broadcast: Client unsubscribed")
		return nil
	}))

	// API endpoint - publish a notification (stores for replay)
	app.POST("/notify", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		msg := r.URL.Query().Get("msg")
		if msg == "" {
			msg = "System notification"
		}

		event := zh.SSEEvent{
			Name: "notification",
			Data: []byte(msg),
		}

		// Store for replay
		replayer.Store(event)

		_, err := w.Write([]byte("Notification stored for replay"))
		return err
	}))

	// API endpoint - broadcast to all connected clients
	app.POST("/broadcast", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		msg := r.URL.Query().Get("msg")
		if msg == "" {
			msg = fmt.Sprintf("Broadcast #%d", atomic.AddInt32(&broadcastCounter, 1))
		}

		event := zh.SSEEvent{
			Name: "broadcast",
			Data: []byte(msg),
		}

		// Broadcast to all connected clients
		hub.Broadcast(event)

		_, err := fmt.Fprintf(w, "Broadcast sent to %d clients", hub.ConnectionCount())
		return err
	}))

	// Auto-generate notifications every 5 seconds for replay demo
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		counter := 0
		for range ticker.C {
			counter++
			event := zh.SSEEvent{
				Name: "auto",
				Data: []byte(fmt.Sprintf("Auto notification #%d at %s", counter, time.Now().Format("15:04:05"))),
			}
			// Store for replay and get event with assigned ID
			event = replayer.Store(event)
			// Broadcast to connected notification clients
			hub.BroadcastTo("notifications", event)
			log.Printf("Auto-generated notification #%d (stored + broadcast)", counter)
		}
	}()

	log.Println("Starting server on http://localhost:8080")
	log.Println("")
	log.Println("Endpoints:")
	log.Println("  GET  /time          - Basic time stream")
	log.Println("  GET  /notifications - Stream with replay support (try reconnecting!)")
	log.Println("  GET  /broadcast     - Subscribe to broadcasts")
	log.Println("  POST /notify?msg=   - Store notification for replay")
	log.Println("  POST /broadcast?msg= - Broadcast to all connected clients")
	log.Println("")
	log.Println("Open http://localhost:8080 in your browser to test")

	if err := app.ListenAndServe(); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>zerohttp SSE Example - Replay & Broadcast</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 1000px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        h1 { color: #333; }
        h2 { margin-top: 0; }
        .container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .status {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 14px;
            font-weight: bold;
        }
        .status.connected { background: #4caf50; color: white; }
        .status.disconnected { background: #f44336; color: white; }
        .status.connecting { background: #ff9800; color: white; }
        button {
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            background: #0066cc;
            color: white;
            cursor: pointer;
            margin-right: 10px;
            margin-bottom: 10px;
        }
        button:hover { background: #0052a3; }
        button:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        input {
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            width: 250px;
            margin-right: 10px;
        }
        .log {
            background: #1a1a1a;
            color: #00ff00;
            padding: 15px;
            border-radius: 4px;
            font-family: monospace;
            height: 150px;
            overflow-y: auto;
            white-space: pre-wrap;
            font-size: 12px;
        }
        .grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
        }
        @media (max-width: 768px) {
            .grid { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
    <h1>zerohttp SSE Example - Replay & Broadcast</h1>

    <div class="container">
        <h2>Time Stream (Basic SSE)</h2>
        <p>Status: <span id="timeStatus" class="status disconnected">Disconnected</span></p>
        <button id="timeConnect" onclick="connectTime()">Connect</button>
        <button id="timeDisconnect" onclick="disconnectTime()" disabled>Disconnect</button>
        <div id="timeLog" class="log"></div>
    </div>

    <div class="grid">
        <div class="container">
            <h2>Notifications (with Replay)</h2>
            <p>Connect, disconnect, then reconnect - missed events will be replayed!</p>
            <p>Status: <span id="notifStatus" class="status disconnected">Disconnected</span></p>
            <p>Last Event ID: <span id="notifLastID">-</span></p>
            <button id="notifConnect" onclick="connectNotif()">Connect</button>
            <button id="notifReconnect" onclick="reconnectNotif()" disabled>Reconnect with Replay</button>
            <button id="notifDisconnect" onclick="disconnectNotif()" disabled>Disconnect</button>
            <div id="notifLog" class="log"></div>
        </div>

        <div class="container">
            <h2>Broadcast Hub</h2>
            <p>Messages sent to all connected broadcast clients</p>
            <p>Status: <span id="broadcastStatus" class="status disconnected">Disconnected</span></p>
            <button id="broadcastConnect" onclick="connectBroadcast()">Connect</button>
            <button id="broadcastDisconnect" onclick="disconnectBroadcast()" disabled>Disconnect</button>
            <div id="broadcastLog" class="log"></div>
        </div>
    </div>

    <div class="container">
        <h2>Send Messages</h2>
        <div>
            <input type="text" id="notifyInput" placeholder="Notification message...">
            <button onclick="sendNotification()">Store Notification</button>
            <span id="notifyResult"></span>
        </div>
        <div style="margin-top: 10px;">
            <input type="text" id="broadcastInput" placeholder="Broadcast message...">
            <button onclick="sendBroadcast()">Broadcast to All</button>
            <span id="broadcastResult"></span>
        </div>
    </div>

    <script>
        let timeSource = null;
        let notifSource = null;
        let broadcastSource = null;
        let notifLastEventID = '';

        function log(elementId, message) {
            const logEl = document.getElementById(elementId);
            const time = new Date().toLocaleTimeString();
            const div = document.createElement('div');
            div.textContent = '[' + time + '] ' + message;
            logEl.appendChild(div);
            logEl.scrollTop = logEl.scrollHeight;
        }

        function setStatus(elementId, status) {
            const el = document.getElementById(elementId);
            el.className = 'status ' + status;
            el.textContent = status.charAt(0).toUpperCase() + status.slice(1);
        }

        function setButtons(connectId, disconnectId, connected) {
            document.getElementById(connectId).disabled = connected;
            document.getElementById(disconnectId).disabled = !connected;
        }

        // Time stream
        function connectTime() {
            if (timeSource) return;
            setStatus('timeStatus', 'connecting');
            log('timeLog', 'Connecting...');

            timeSource = new EventSource('/time');

            timeSource.addEventListener('time', (e) => {
                log('timeLog', 'Time: ' + e.data);
            });

            timeSource.onopen = () => {
                setStatus('timeStatus', 'connected');
                setButtons('timeConnect', 'timeDisconnect', true);
            };

            timeSource.onerror = () => {
                log('timeLog', 'Error/Disconnected');
            };
        }

        function disconnectTime() {
            if (timeSource) {
                timeSource.close();
                timeSource = null;
                setStatus('timeStatus', 'disconnected');
                setButtons('timeConnect', 'timeDisconnect', false);
                log('timeLog', 'Disconnected');
            }
        }

        // Notifications with replay
        function connectNotif() {
            if (notifSource) return;
            doConnectNotif(null);
        }

        function doConnectNotif(lastID) {
            setStatus('notifStatus', 'connecting');
            log('notifLog', lastID ? 'Reconnecting with Last-Event-ID: ' + lastID + '...' : 'Connecting...');

            // Create EventSource with optional last_id query parameter for replay
            // (Browser EventSource doesn't allow setting headers manually)
            const url = lastID ? '/notifications?last_id=' + encodeURIComponent(lastID) : '/notifications';
            notifSource = new EventSource(url);

            // Catch all message types
            notifSource.onmessage = (e) => {
                notifLastEventID = e.lastEventId;
                document.getElementById('notifLastID').textContent = notifLastEventID;
                log('notifLog', '📨 ' + e.data + ' (ID: ' + e.lastEventId + ')');
            };

            notifSource.addEventListener('notification', (e) => {
                notifLastEventID = e.lastEventId;
                document.getElementById('notifLastID').textContent = notifLastEventID;
                log('notifLog', '📢 ' + e.data + ' (ID: ' + e.lastEventId + ')');
            });

            notifSource.addEventListener('auto', (e) => {
                notifLastEventID = e.lastEventId;
                document.getElementById('notifLastID').textContent = notifLastEventID;
                log('notifLog', '⚡ ' + e.data + ' (ID: ' + e.lastEventId + ')');
            });

            notifSource.onopen = () => {
                setStatus('notifStatus', 'connected');
                document.getElementById('notifConnect').disabled = true;
                document.getElementById('notifReconnect').disabled = true;
                document.getElementById('notifDisconnect').disabled = false;
                log('notifLog', 'Connected! Waiting for events... (auto-generated every 5s)');
            };

            notifSource.onerror = (e) => {
                console.error('SSE Error:', e);
                log('notifLog', 'Error/Disconnected (check console)');
            };
        }

        function disconnectNotif() {
            if (notifSource) {
                notifSource.close();
                notifSource = null;
                setStatus('notifStatus', 'disconnected');
                document.getElementById('notifConnect').disabled = false;
                const hasLastID = notifLastEventID !== '';
                document.getElementById('notifReconnect').disabled = !hasLastID;
                document.getElementById('notifDisconnect').disabled = true;
                log('notifLog', 'Disconnected. Last ID: ' + (notifLastEventID || 'none') + '. Reconnect button: ' + (hasLastID ? 'ENABLED' : 'disabled (need events first)'));
            }
        }

        function reconnectNotif() {
            if (notifSource || notifLastEventID === '') return;
            // Pass last_id as query parameter since EventSource doesn't allow custom headers
            doConnectNotif(notifLastEventID);
        }

        // Broadcast
        function connectBroadcast() {
            if (broadcastSource) return;
            setStatus('broadcastStatus', 'connecting');
            log('broadcastLog', 'Connecting...');

            broadcastSource = new EventSource('/broadcast');

            broadcastSource.addEventListener('info', (e) => {
                log('broadcastLog', 'ℹ️ ' + e.data);
            });

            broadcastSource.addEventListener('broadcast', (e) => {
                log('broadcastLog', '📡 ' + e.data);
            });

            broadcastSource.onopen = () => {
                setStatus('broadcastStatus', 'connected');
                setButtons('broadcastConnect', 'broadcastDisconnect', true);
            };

            broadcastSource.onerror = () => {
                log('broadcastLog', 'Error/Disconnected');
            };
        }

        function disconnectBroadcast() {
            if (broadcastSource) {
                broadcastSource.close();
                broadcastSource = null;
                setStatus('broadcastStatus', 'disconnected');
                setButtons('broadcastConnect', 'broadcastDisconnect', false);
                log('broadcastLog', 'Disconnected');
            }
        }

        // Send messages
        function sendNotification() {
            const msg = document.getElementById('notifyInput').value;
            fetch('/notify?msg=' + encodeURIComponent(msg), {method: 'POST'})
                .then(r => r.text())
                .then(text => {
                    document.getElementById('notifyResult').textContent = text;
                    setTimeout(() => document.getElementById('notifyResult').textContent = '', 3000);
                });
        }

        function sendBroadcast() {
            const msg = document.getElementById('broadcastInput').value;
            fetch('/broadcast?msg=' + encodeURIComponent(msg), {method: 'POST'})
                .then(r => r.text())
                .then(text => {
                    document.getElementById('broadcastResult').textContent = text;
                    setTimeout(() => document.getElementById('broadcastResult').textContent = '', 3000);
                });
        }
    </script>
</body>
</html>`
