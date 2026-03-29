package websocket

import (
	"errors"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

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
			zhtest.AssertEqual(t, tt.want, got)
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
			zhtest.AssertEqual(t, tt.want, got)
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
			zhtest.AssertEqual(t, tt.want, got)
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
			zhtest.AssertEqual(t, tt.want, int(tt.code))
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
			zhtest.AssertEqual(t, tt.want, tt.got)
		})
	}
}
