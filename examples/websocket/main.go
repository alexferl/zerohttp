//go:build ignore

// This example demonstrates how to use WebSocket support with zh.
// It uses gorilla/websocket as the WebSocket library.
//
// To run this example:
//   1. go get github.com/gorilla/websocket
//   2. go run examples/websocket/main.go
//   3. Open http://localhost:8080 in your browser

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
)

// myUpgrader wraps gorilla/websocket to implement config.WebSocketUpgrader
type myUpgrader struct {
	upgrader *websocket.Upgrader
}

func (m *myUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (config.WebSocketConn, error) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &myConn{conn: conn}, nil
}

// myConn wraps gorilla/websocket.Conn to implement config.WebSocketConn
type myConn struct {
	conn *websocket.Conn
}

func (c *myConn) ReadMessage() (int, []byte, error) {
	return c.conn.ReadMessage()
}

func (c *myConn) WriteMessage(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

func (c *myConn) Close() error {
	return c.conn.Close()
}

func (c *myConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func main() {
	// Create gorilla websocket upgrader
	gupgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for demo
		},
	}

	// Create zerohttp server with WebSocket support
	// Disable default middlewares to avoid CSP blocking inline styles/scripts in the demo
	app := zh.New(
		config.Config{
			DisableDefaultMiddlewares: true,
			Extensions: config.ExtensionsConfig{
				WebSocketUpgrader: &myUpgrader{upgrader: gupgrader},
			},
		},
	)

	// Serve HTML client
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(indexHTML))
		return nil
	}))

	// WebSocket endpoint
	app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Get the WebSocket upgrader
		upgrader := app.WebSocketUpgrader()
		if upgrader == nil {
			return fmt.Errorf("websocket upgrader not configured")
		}

		// Upgrade the connection
		ws, err := upgrader.Upgrade(w, r)
		if err != nil {
			return err
		}
		defer ws.Close()

		clientAddr := ws.RemoteAddr().String()
		log.Printf("WebSocket client connected: %s", clientAddr)

		// Echo loop
		for {
			mt, msg, err := ws.ReadMessage()
			if err != nil {
				log.Printf("WebSocket client disconnected: %s (%v)", clientAddr, err)
				break
			}

			log.Printf("Received from %s: %s", clientAddr, string(msg))

			// Echo back with prefix
			response := fmt.Sprintf("Echo: %s", string(msg))
			if err := ws.WriteMessage(mt, []byte(response)); err != nil {
				log.Printf("Write error: %v", err)
				break
			}
		}

		return nil
	}))

	log.Println("Starting server on http://localhost:8080")
	log.Println("Open http://localhost:8080 in your browser to test WebSocket")
	if err := app.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>zerohttp WebSocket Example</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        h1 { color: #333; }
        .container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        #status {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 14px;
            font-weight: bold;
        }
        .status.connected { background: #4caf50; color: white; }
        .status.disconnected { background: #f44336; color: white; }
        .status.connecting { background: #ff9800; color: white; }
        input[type="text"] {
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            width: 300px;
            margin-right: 10px;
        }
        button {
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            background: #0066cc;
            color: white;
            cursor: pointer;
        }
        button:hover { background: #0052a3; }
        button:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        #messages {
            background: #1a1a1a;
            color: #00ff00;
            padding: 15px;
            border-radius: 4px;
            font-family: monospace;
            height: 300px;
            overflow-y: auto;
            white-space: pre-wrap;
        }
        .message {
            margin: 4px 0;
        }
        .message.sent { color: #ffff00; }
        .message.received { color: #00ff00; }
    </style>
</head>
<body>
    <h1>zerohttp WebSocket Example</h1>

    <div class="container">
        <h2>Connection</h2>
        <p>Status: <span id="status" class="status disconnected">Disconnected</span></p>
        <button id="connectBtn" onclick="connect()">Connect</button>
        <button id="disconnectBtn" onclick="disconnect()" disabled>Disconnect</button>
    </div>

    <div class="container">
        <h2>Send Message</h2>
        <input type="text" id="messageInput" placeholder="Type a message..." disabled>
        <button id="sendBtn" onclick="sendMessage()" disabled>Send</button>
    </div>

    <div class="container">
        <h2>Messages</h2>
        <div id="messages"></div>
        <button onclick="clearMessages()">Clear</button>
    </div>

    <script>
        let ws = null;
        const statusEl = document.getElementById('status');
        const messagesEl = document.getElementById('messages');
        const connectBtn = document.getElementById('connectBtn');
        const disconnectBtn = document.getElementById('disconnectBtn');
        const messageInput = document.getElementById('messageInput');
        const sendBtn = document.getElementById('sendBtn');

        function setStatus(status, text) {
            statusEl.className = 'status ' + status;
            statusEl.textContent = text;
        }

        function updateUI(connected) {
            connectBtn.disabled = connected;
            disconnectBtn.disabled = !connected;
            messageInput.disabled = !connected;
            sendBtn.disabled = !connected;
        }

        function addMessage(text, type) {
            const div = document.createElement('div');
            div.className = 'message ' + type;
            const time = new Date().toLocaleTimeString();
            div.textContent = '[' + time + '] ' + text;
            messagesEl.appendChild(div);
            messagesEl.scrollTop = messagesEl.scrollHeight;
        }

        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = protocol + '//' + window.location.host + '/ws';

            setStatus('connecting', 'Connecting...');
            addMessage('Connecting to ' + wsUrl + '...', 'sent');

            ws = new WebSocket(wsUrl);

            ws.onopen = function() {
                setStatus('connected', 'Connected');
                updateUI(true);
                addMessage('Connected!', 'received');
            };

            ws.onmessage = function(event) {
                addMessage('Received: ' + event.data, 'received');
            };

            ws.onclose = function() {
                setStatus('disconnected', 'Disconnected');
                updateUI(false);
                addMessage('Disconnected', 'sent');
                ws = null;
            };

            ws.onerror = function(error) {
                addMessage('Error: ' + error, 'sent');
            };
        }

        function disconnect() {
            if (ws) {
                ws.close();
            }
        }

        function sendMessage() {
            if (!ws || ws.readyState !== WebSocket.OPEN) {
                return;
            }

            const text = messageInput.value.trim();
            if (!text) return;

            ws.send(text);
            addMessage('Sent: ' + text, 'sent');
            messageInput.value = '';
        }

        function clearMessages() {
            messagesEl.innerHTML = '';
        }

        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') sendMessage();
        });
    </script>
</body>
</html>`
