package zerohttp

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/extensions/websocket"
)

// mockWebSocketConn is a mock implementation of WebSocketConn for testing.
type mockWebSocketConn struct {
	readMessages []struct {
		msgType int
		data    []byte
		err     error
	}
	writeMessages []struct {
		msgType int
		data    []byte
	}
	closed     bool
	remoteAddr net.Addr
	readIndex  int
	writeErr   error
	closeErr   error
}

func (m *mockWebSocketConn) ReadMessage() (int, []byte, error) {
	if m.readIndex >= len(m.readMessages) {
		return 0, nil, errors.New("no more messages")
	}
	msg := m.readMessages[m.readIndex]
	m.readIndex++
	return msg.msgType, msg.data, msg.err
}

func (m *mockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writeMessages = append(m.writeMessages, struct {
		msgType int
		data    []byte
	}{messageType, data})
	return nil
}

func (m *mockWebSocketConn) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockWebSocketConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

// mockWebSocketUpgrader is a mock implementation of WebSocketUpgrader for testing.
type mockWebSocketUpgrader struct {
	conn websocket.Connection
	err  error
}

func (m *mockWebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (websocket.Connection, error) {
	return m.conn, m.err
}

func TestWebSocketConnInterface(t *testing.T) {
	// Verify mock implements the interface
	var _ websocket.Connection = (*mockWebSocketConn)(nil)
	var _ websocket.Upgrader = (*mockWebSocketUpgrader)(nil)
}

func TestServerWebSocketUpgrader(t *testing.T) {
	// Test that WebSocket upgrader can be set and retrieved
	app := New()

	// Initially should be nil
	if app.WebSocketUpgrader() != nil {
		t.Error("WebSocketUpgrader should be nil initially")
	}

	// Set upgrader
	mockUpgrader := &mockWebSocketUpgrader{}
	app.SetWebSocketUpgrader(mockUpgrader)

	// Should be retrievable
	if app.WebSocketUpgrader() != mockUpgrader {
		t.Error("WebSocketUpgrader should be retrievable")
	}
}

func TestServerWithWebSocketUpgrader(t *testing.T) {
	// Test that WebSocket upgrader can be set via config option
	mockUpgrader := &mockWebSocketUpgrader{}
	app := New(Config{Extensions: ExtensionsConfig{WebSocketUpgrader: mockUpgrader}})

	if app.WebSocketUpgrader() != mockUpgrader {
		t.Error("WebSocketUpgrader should be set via config option")
	}
}

func TestWebSocketHandler(t *testing.T) {
	// Create mock connection
	mockConn := &mockWebSocketConn{
		readMessages: []struct {
			msgType int
			data    []byte
			err     error
		}{
			{msgType: 1, data: []byte("hello")},
			{msgType: 1, data: []byte("world")},
			{err: errors.New("connection closed")},
		},
	}

	mockUpgrader := &mockWebSocketUpgrader{conn: mockConn}

	app := New(Config{Extensions: ExtensionsConfig{WebSocketUpgrader: mockUpgrader}})

	app.GET("/ws", HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ws, err := app.WebSocketUpgrader().Upgrade(w, r)
		if err != nil {
			return err
		}
		defer func() { _ = ws.Close() }()

		// Echo loop
		for {
			mt, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}
			if err := ws.WriteMessage(mt, msg); err != nil {
				break
			}
		}

		return nil
	}))

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()

	// Serve request
	app.ServeHTTP(rec, req)

	// Verify connection was closed
	if !mockConn.closed {
		t.Error("WebSocket connection should be closed")
	}

	// Verify messages were echoed
	if len(mockConn.writeMessages) != 2 {
		t.Errorf("Expected 2 written messages, got %d", len(mockConn.writeMessages))
	}
}
