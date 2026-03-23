package websocket

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
)

// Connection represents a WebSocket connection.
// This is a minimal interface that can be implemented by wrapping
// any WebSocket library (e.g., gorilla/websocket, nhooyr/websocket).
type Connection interface {
	// ReadMessage reads a message from the connection.
	// Returns message type (text=1, binary=2), payload, and error.
	ReadMessage() (int, []byte, error)

	// WriteMessage writes a message to the connection.
	// messageType is 1 for text, 2 for binary.
	WriteMessage(messageType int, data []byte) error

	// Close closes the connection gracefully.
	Close() error

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr
}

// Upgrader handles upgrading HTTP connections to WebSocket.
// Users provide their own implementation using their preferred WebSocket library.
type Upgrader interface {
	// Upgrade upgrades the HTTP connection to WebSocket.
	// The implementation is responsible for the RFC 6455 handshake
	// and returning a Connection.
	Upgrade(w http.ResponseWriter, r *http.Request) (Connection, error)
}

// CloseCode represents a WebSocket close code as defined in RFC 6455.
type CloseCode int

// WebSocket close code constants.
const (
	CloseNormalClosure           CloseCode = 1000
	CloseGoingAway               CloseCode = 1001
	CloseProtocolError           CloseCode = 1002
	CloseUnsupportedData         CloseCode = 1003
	CloseNoStatusReceived        CloseCode = 1005
	CloseAbnormalClosure         CloseCode = 1006
	CloseInvalidFramePayloadData CloseCode = 1007
	ClosePolicyViolation         CloseCode = 1008
	CloseMessageTooBig           CloseCode = 1009
	CloseMandatoryExtension      CloseCode = 1010
	CloseInternalServerErr       CloseCode = 1011
	CloseServiceRestart          CloseCode = 1012
	CloseTryAgainLater           CloseCode = 1013
	CloseBadGateway              CloseCode = 1014
	CloseTLSHandshake            CloseCode = 1015
)

// CloseError represents a WebSocket close error.
type CloseError struct {
	Code   int
	Reason string
}

// Error implements the error interface.
func (e *CloseError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("websocket: close %d %s", e.Code, e.Reason)
	}
	return fmt.Sprintf("websocket: close %d", e.Code)
}

// MessageType represents the type of WebSocket message.
type MessageType int

// WebSocket message type constants as defined in RFC 6455.
const (
	TextMessage   MessageType = 1
	BinaryMessage MessageType = 2
	CloseMessage  MessageType = 8
	PingMessage   MessageType = 9
	PongMessage   MessageType = 10
)

// IsCloseError returns true if the error is a WebSocket close error
// with one of the specified codes.
func IsCloseError(err error, codes ...int) bool {
	if err == nil {
		return false
	}

	var ce *CloseError
	if errors.As(err, &ce) {
		return slices.Contains(codes, ce.Code)
	}

	return false
}

// IsUnexpectedCloseError returns true if the error is a close error
// with a code not in the expected list.
func IsUnexpectedCloseError(err error, expectedCodes ...int) bool {
	if err == nil {
		return false
	}

	var ce *CloseError
	if errors.As(err, &ce) {
		return !slices.Contains(expectedCodes, ce.Code)
	}

	return false
}
