// Package zerohttp provides WebSocket upgrader configuration. See [Server.SetWebSocketUpgrader] and [Server.WebSocketUpgrader].
package zerohttp

import "github.com/alexferl/zerohttp/extensions/websocket"

// SetWebSocketUpgrader sets the WebSocket upgrader instance. This can be used to inject
// a WebSocket implementation (e.g., gorilla/websocket, nhooyr/websocket) after creating the server.
//
// The WebSocket upgrader provides the Upgrade method for handling WebSocket connections.
// Users bring their own WebSocket library and implement the WebSocketUpgrader interface,
// or use a thin wrapper around their preferred library.
//
// Example with gorilla/websocket:
//
//	import "github.com/gorilla/websocket"
//
//	upgrader := &websocket.Upgrader{
//	    CheckOrigin: func(r *http.Request) bool { return true },
//	}
//
//	app := zerohttp.New()
//	app.SetWebSocketUpgrader(&myUpgrader{upgrader})
//
//	app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    ws, err := app.WebSocketUpgrader().Upgrade(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer ws.Close()
//	    // ... handle connection ...
//	}))
//
// Parameters:
//   - upgrader: A WebSocket upgrader instance implementing the config.WebSocketUpgrader interface
func (s *Server) SetWebSocketUpgrader(upgrader websocket.Upgrader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webSocketUpgrader = upgrader
}

// WebSocketUpgrader returns the configured WebSocket upgrader (if any).
// Returns nil if no WebSocket upgrader has been configured.
func (s *Server) WebSocketUpgrader() websocket.Upgrader {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.webSocketUpgrader
}
