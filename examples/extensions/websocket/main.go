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

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.File(w, r, "static/index.html")
	}))

	app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		upgrader := app.WebSocketUpgrader()
		if upgrader == nil {
			return fmt.Errorf("websocket upgrader not configured")
		}

		ws, err := upgrader.Upgrade(w, r)
		if err != nil {
			return err
		}
		defer ws.Close()

		clientAddr := ws.RemoteAddr().String()
		log.Printf("WebSocket client connected: %s", clientAddr)

		for {
			mt, msg, err := ws.ReadMessage()
			if err != nil {
				log.Printf("WebSocket client disconnected: %s (%v)", clientAddr, err)
				break
			}

			log.Printf("Received from %s: %s", clientAddr, string(msg))

			response := fmt.Appendf(nil, "Echo: %s", msg)
			if err := ws.WriteMessage(mt, response); err != nil {
				log.Printf("Write error: %v", err)
				break
			}
		}

		return nil
	}))

	log.Fatal(app.ListenAndServe())
}
