package zerohttp

import (
	"errors"
	"slices"

	"github.com/alexferl/zerohttp/config"
)

// Re-export WebSocket types from config package for convenience.

// MessageType represents the type of WebSocket message.
type MessageType = config.MessageType

// WebSocket message type constants as defined in RFC 6455.
const (
	TextMessage   = config.TextMessage
	BinaryMessage = config.BinaryMessage
	CloseMessage  = config.CloseMessage
	PingMessage   = config.PingMessage
	PongMessage   = config.PongMessage
)

// WebSocketUpgrader handles upgrading HTTP connections to WebSocket.
// Users provide their own implementation using their preferred WebSocket library
// (e.g., gorilla/websocket, nhooyr/websocket).
type WebSocketUpgrader = config.WebSocketUpgrader

// WebSocketConn represents a WebSocket connection.
// This is a minimal interface that can be implemented by wrapping
// any WebSocket library.
type WebSocketConn = config.WebSocketConn

// CloseCode represents a WebSocket close code as defined in RFC 6455.
type CloseCode = config.CloseCode

// WebSocket close code constants.
const (
	CloseNormalClosure           = config.CloseNormalClosure
	CloseGoingAway               = config.CloseGoingAway
	CloseProtocolError           = config.CloseProtocolError
	CloseUnsupportedData         = config.CloseUnsupportedData
	CloseNoStatusReceived        = config.CloseNoStatusReceived
	CloseAbnormalClosure         = config.CloseAbnormalClosure
	CloseInvalidFramePayloadData = config.CloseInvalidFramePayloadData
	ClosePolicyViolation         = config.ClosePolicyViolation
	CloseMessageTooBig           = config.CloseMessageTooBig
	CloseMandatoryExtension      = config.CloseMandatoryExtension
	CloseInternalServerErr       = config.CloseInternalServerErr
	CloseServiceRestart          = config.CloseServiceRestart
	CloseTryAgainLater           = config.CloseTryAgainLater
	CloseBadGateway              = config.CloseBadGateway
	CloseTLSHandshake            = config.CloseTLSHandshake
)

// CloseError represents a WebSocket close error.
type CloseError = config.CloseError

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
