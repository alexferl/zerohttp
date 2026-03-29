package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultProvider(t *testing.T) {
	provider := NewDefaultProvider()
	zhtest.AssertNotNil(t, provider)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sse", nil)

	conn, err := provider.New(w, r)
	zhtest.AssertNoError(t, err)
	defer func() { _ = conn.Close() }()

	zhtest.AssertNotNil(t, conn)
}
