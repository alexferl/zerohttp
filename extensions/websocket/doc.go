// Package websocket provides WebSocket support for zerohttp.
//
// This package defines interfaces for WebSocket connections and upgraders.
// Users bring their own WebSocket library (e.g., github.com/gorilla/websocket)
// by implementing the Connection and Upgrader interfaces.
//
// # Usage
//
// Configure zerohttp with your WebSocket upgrader:
//
//	import (
//	    zh "github.com/alexferl/zerohttp"
//	    "github.com/alexferl/zerohttp/extensions/websocket"
//	    gorillaws "github.com/gorilla/websocket"
//	)
//
//	// Create an adapter for gorilla/websocket
//	type GorillaUpgrader struct {
//	    upgrader gorillaws.Upgrader
//	}
//
//	func (u *GorillaUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (websocket.Connection, error) {
//	    conn, err := u.upgrader.Upgrade(w, r, nil)
//	    // Wrap conn in a type that implements websocket.Connection...
//	}
//
//	app := zh.New(zh.Config{
//	    WebSocketUpgrader: &GorillaUpgrader{},
//	})
//
//	app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    conn, err := app.WebSocketUpgrader().Upgrade(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer conn.Close()
//
//	    // Handle WebSocket connection
//	    for {
//	        mt, msg, err := conn.ReadMessage()
//	        if err != nil {
//	            return err
//	        }
//	        // Process message...
//	    }
//	}))
//
// # Close Codes
//
// The package provides RFC 6455 standard close codes:
//
//	websocket.CloseNormalClosure      // 1000 - Normal closure
//	websocket.CloseGoingAway          // 1001 - Endpoint going away
//	websocket.CloseProtocolError      // 1002 - Protocol error
//	websocket.CloseUnsupportedData    // 1003 - Unsupported data
//	websocket.CloseMessageTooBig      // 1009 - Message too big
//	websocket.CloseInternalServerErr  // 1011 - Server error
//
// Use IsCloseError to check for specific close codes:
//
//	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
//	    // Client disconnected normally
//	}
package websocket
