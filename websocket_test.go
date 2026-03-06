package zerohttp

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

// mockWebSocketConn is a mock implementation of config.WebSocketConn for testing.
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

// mockWebSocketUpgrader is a mock implementation of config.WebSocketUpgrader for testing.
type mockWebSocketUpgrader struct {
	conn config.WebSocketConn
	err  error
}

func (m *mockWebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (config.WebSocketConn, error) {
	return m.conn, m.err
}

func TestWebSocketConnInterface(t *testing.T) {
	// Verify mock implements the interface
	var _ config.WebSocketConn = (*mockWebSocketConn)(nil)
	var _ config.WebSocketUpgrader = (*mockWebSocketUpgrader)(nil)
}

func TestCloseError(t *testing.T) {
	tests := []struct {
		name string
		err  *CloseError
		want string
	}{
		{
			name: "with reason",
			err:  &CloseError{Code: 1000, Reason: "normal closure"},
			want: "websocket: close 1000 normal closure",
		},
		{
			name: "without reason",
			err:  &CloseError{Code: 1001},
			want: "websocket: close 1001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("CloseError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCloseError(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		codes []int
		want  bool
	}{
		{
			name:  "nil error",
			err:   nil,
			codes: []int{1000},
			want:  false,
		},
		{
			name:  "matching code",
			err:   &CloseError{Code: 1000},
			codes: []int{1000, 1001},
			want:  true,
		},
		{
			name:  "non-matching code",
			err:   &CloseError{Code: 1002},
			codes: []int{1000, 1001},
			want:  false,
		},
		{
			name:  "not a CloseError",
			err:   errors.New("some error"),
			codes: []int{1000},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCloseError(tt.err, tt.codes...)
			if got != tt.want {
				t.Errorf("IsCloseError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnexpectedCloseError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedCodes []int
		want          bool
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedCodes: []int{1000},
			want:          false,
		},
		{
			name:          "expected code",
			err:           &CloseError{Code: 1000},
			expectedCodes: []int{1000, 1001},
			want:          false,
		},
		{
			name:          "unexpected code",
			err:           &CloseError{Code: 1002},
			expectedCodes: []int{1000, 1001},
			want:          true,
		},
		{
			name:          "not a CloseError",
			err:           errors.New("some error"),
			expectedCodes: []int{1000},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnexpectedCloseError(tt.err, tt.expectedCodes...)
			if got != tt.want {
				t.Errorf("IsUnexpectedCloseError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloseCodeConstants(t *testing.T) {
	// Verify all close code constants are defined correctly
	tests := []struct {
		name string
		code CloseCode
		want int
	}{
		{"CloseNormalClosure", CloseNormalClosure, 1000},
		{"CloseGoingAway", CloseGoingAway, 1001},
		{"CloseProtocolError", CloseProtocolError, 1002},
		{"CloseUnsupportedData", CloseUnsupportedData, 1003},
		{"CloseNoStatusReceived", CloseNoStatusReceived, 1005},
		{"CloseAbnormalClosure", CloseAbnormalClosure, 1006},
		{"CloseInvalidFramePayloadData", CloseInvalidFramePayloadData, 1007},
		{"ClosePolicyViolation", ClosePolicyViolation, 1008},
		{"CloseMessageTooBig", CloseMessageTooBig, 1009},
		{"CloseMandatoryExtension", CloseMandatoryExtension, 1010},
		{"CloseInternalServerErr", CloseInternalServerErr, 1011},
		{"CloseServiceRestart", CloseServiceRestart, 1012},
		{"CloseTryAgainLater", CloseTryAgainLater, 1013},
		{"CloseBadGateway", CloseBadGateway, 1014},
		{"CloseTLSHandshake", CloseTLSHandshake, 1015},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.code) != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}

func TestMessageTypeConstants(t *testing.T) {
	// Verify message type constants have correct values
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"TextMessage", int(TextMessage), 1},
		{"BinaryMessage", int(BinaryMessage), 2},
		{"CloseMessage", int(CloseMessage), 8},
		{"PingMessage", int(PingMessage), 9},
		{"PongMessage", int(PongMessage), 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
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
	app := New(config.Config{WebSocketUpgrader: mockUpgrader})

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

	app := New(config.Config{WebSocketUpgrader: mockUpgrader})

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
