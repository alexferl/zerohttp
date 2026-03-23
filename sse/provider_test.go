package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultProvider(t *testing.T) {
	provider := NewDefaultProvider()
	if provider == nil {
		t.Fatal("expected provider to not be nil")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sse", nil)

	conn, err := provider.New(w, r)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func() { _ = conn.Close() }()

	if conn == nil {
		t.Error("expected connection to not be nil")
	}
}
