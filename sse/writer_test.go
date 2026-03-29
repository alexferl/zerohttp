package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

// flusherWriter wraps a ResponseWriter and adds Flush
type flusherWriter struct {
	http.ResponseWriter
	header http.Header
}

func (f *flusherWriter) Flush() {}

func TestNewWriter(t *testing.T) {
	t.Run("successfully creates SSEWriter", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewWriter(w, r)
		zhtest.AssertNoError(t, err)

		// Check headers
		zhtest.AssertWith(t, w).Header(httpx.HeaderContentType, httpx.MIMETextEventStream)

		_ = writer
	})

	t.Run("returns error if headers already sent", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlainCharset)
		w.WriteHeader(http.StatusOK)

		_, err := NewWriter(w, r)
		zhtest.AssertError(t, err)
	})
}

func TestWriter_WriteEvent(t *testing.T) {
	t.Run("writes event with all fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewWriter(w, r)
		zhtest.AssertNoError(t, err)

		event := Event{
			ID:    "456",
			Name:  "message",
			Data:  []byte("hello"),
			Retry: 3000 * time.Millisecond,
		}

		err = writer.WriteEvent(event)
		zhtest.AssertNoError(t, err)

		zhtest.AssertWith(t, w).
			BodyContains("id: 456").
			BodyContains("event: message").
			BodyContains("retry: 3000").
			BodyContains("data: hello")
	})

	t.Run("rejects event ID with CRLF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewWriter(w, r)
		zhtest.AssertNoError(t, err)

		event := Event{ID: "abc\r\ndef", Data: []byte("test")}
		err = writer.WriteEvent(event)
		zhtest.AssertError(t, err)
	})

	t.Run("rejects event name with LF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewWriter(w, r)
		zhtest.AssertNoError(t, err)

		event := Event{Name: "update\nnotify", Data: []byte("test")}
		err = writer.WriteEvent(event)
		zhtest.AssertError(t, err)
	})
}

func TestWriter_WriteComment(t *testing.T) {
	t.Run("writes comment", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewWriter(w, r)
		zhtest.AssertNoError(t, err)

		err = writer.WriteComment("keepalive")
		zhtest.AssertNoError(t, err)

		zhtest.AssertWith(t, w).BodyContains(": keepalive")
	})

	t.Run("returns error when context cancelled", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

		writer, err := NewWriter(w, r)
		zhtest.AssertNoError(t, err)

		cancel()
		time.Sleep(10 * time.Millisecond)

		err = writer.WriteComment("test")
		zhtest.AssertError(t, err)
		zhtest.AssertTrue(t, strings.Contains(err.Error(), "sse:"))
	})
}

func TestWriter_Flush(t *testing.T) {
	t.Run("flushes without error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewWriter(w, r)
		zhtest.AssertNoError(t, err)

		// Flush should not panic
		writer.Flush()
	})
}
