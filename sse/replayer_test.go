package sse

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestMemoryReplayer(t *testing.T) {
	t.Run("stores and replays events", func(t *testing.T) {
		replay := NewMemoryReplayer(100, 0)

		// Store some events
		replay.Store(Event{Name: "test1", Data: []byte("data1")})
		replay.Store(Event{Name: "test2", Data: []byte("data2")})
		replay.Store(Event{Name: "test3", Data: []byte("data3")})

		// Replay from event 1
		var replayed []Event
		count, err := replay.Replay("1", func(e Event) error {
			replayed = append(replayed, e)
			return nil
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 2, count)
		zhtest.AssertEqual(t, 2, len(replayed))
	})

	t.Run("respects max events limit", func(t *testing.T) {
		replay := NewMemoryReplayer(2, 0)

		replay.Store(Event{Data: []byte("data1")})
		replay.Store(Event{Data: []byte("data2")})
		replay.Store(Event{Data: []byte("data3")}) // Should evict data1

		count, _ := replay.Replay("", func(e Event) error {
			return nil
		})
		zhtest.AssertEqual(t, 2, count)
	})

	t.Run("respects TTL", func(t *testing.T) {
		replay := NewMemoryReplayer(100, 50*time.Millisecond)

		replay.Store(Event{Data: []byte("old")})
		time.Sleep(100 * time.Millisecond)
		replay.Store(Event{Data: []byte("new")})

		count, _ := replay.Replay("", func(e Event) error {
			return nil
		})
		zhtest.AssertEqual(t, 1, count)
	})

	t.Run("auto-assigns IDs", func(t *testing.T) {
		replay := NewMemoryReplayer(100, 0)

		event := Event{Data: []byte("test")}
		returnedEvent := replay.Store(event)

		// Check that returned event has ID assigned
		zhtest.AssertNotEmpty(t, returnedEvent.ID)

		// Check that ID is accessible immediately without replay
		zhtest.AssertEqual(t, "1", returnedEvent.ID)

		// Also verify it's stored correctly
		var replayed Event
		_, _ = replay.Replay("", func(e Event) error {
			replayed = e
			return nil
		})
		zhtest.AssertNotEmpty(t, replayed.ID)
		zhtest.AssertEqual(t, returnedEvent.ID, replayed.ID)
	})
}

func TestWithReplay(t *testing.T) {
	t.Run("creates stream without replay when no header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		replay := NewMemoryReplayer(100, 0)

		stream, err := WithReplay(w, r, replay)
		zhtest.AssertNoError(t, err)
		defer func() { _ = stream.Close() }()

		// Should not replay anything
	})

	t.Run("replays events when Last-Event-ID present", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set(httpx.HeaderLastEventId, "0")
		replay := NewMemoryReplayer(100, 0)

		replay.Store(Event{Data: []byte("event1")})
		replay.Store(Event{Data: []byte("event2")})

		stream, err := WithReplay(w, r, replay)
		zhtest.AssertNoError(t, err)
		defer func() { _ = stream.Close() }()

		zhtest.AssertWith(t, w).
			BodyContains("event1").
			BodyContains("event2")
	})

	t.Run("returns error for invalid Last-Event-ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set(httpx.HeaderLastEventId, "not-a-number")
		replay := NewMemoryReplayer(100, 0)

		_, err := WithReplay(w, r, replay)
		zhtest.AssertError(t, err)
	})

	t.Run("returns error when Last-Event-ID present but replayer is nil", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set(httpx.HeaderLastEventId, "1")

		_, err := WithReplay(w, r, nil)
		zhtest.AssertError(t, err)
		zhtest.AssertTrue(t, strings.Contains(err.Error(), "no replayer configured"))
	})
}
