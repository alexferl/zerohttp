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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 events replayed, got %d", count)
		}
		if len(replayed) != 2 {
			t.Errorf("expected 2 events in slice, got %d", len(replayed))
		}
	})

	t.Run("respects max events limit", func(t *testing.T) {
		replay := NewMemoryReplayer(2, 0)

		replay.Store(Event{Data: []byte("data1")})
		replay.Store(Event{Data: []byte("data2")})
		replay.Store(Event{Data: []byte("data3")}) // Should evict data1

		count, _ := replay.Replay("", func(e Event) error {
			return nil
		})
		if count != 2 {
			t.Errorf("expected 2 events (max), got %d", count)
		}
	})

	t.Run("respects TTL", func(t *testing.T) {
		replay := NewMemoryReplayer(100, 50*time.Millisecond)

		replay.Store(Event{Data: []byte("old")})
		time.Sleep(100 * time.Millisecond)
		replay.Store(Event{Data: []byte("new")})

		count, _ := replay.Replay("", func(e Event) error {
			return nil
		})
		if count != 1 {
			t.Errorf("expected 1 event (old expired), got %d", count)
		}
	})

	t.Run("auto-assigns IDs", func(t *testing.T) {
		replay := NewMemoryReplayer(100, 0)

		event := Event{Data: []byte("test")}
		returnedEvent := replay.Store(event)

		// Check that returned event has ID assigned
		if returnedEvent.ID == "" {
			t.Error("expected returned event to have ID auto-assigned")
		}

		// Check that ID is accessible immediately without replay
		if returnedEvent.ID != "1" {
			t.Errorf("expected ID to be 1, got %s", returnedEvent.ID)
		}

		// Also verify it's stored correctly
		var replayed Event
		_, _ = replay.Replay("", func(e Event) error {
			replayed = e
			return nil
		})
		if replayed.ID == "" {
			t.Error("expected ID to be auto-assigned in storage")
		}
		if replayed.ID != returnedEvent.ID {
			t.Errorf("replay ID %s doesn't match returned ID %s", replayed.ID, returnedEvent.ID)
		}
	})
}

func TestWithReplay(t *testing.T) {
	t.Run("creates stream without replay when no header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		replay := NewMemoryReplayer(100, 0)

		stream, err := WithReplay(w, r, replay)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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
		if err == nil {
			t.Error("expected error for invalid Last-Event-ID")
		}
	})

	t.Run("returns error when Last-Event-ID present but replayer is nil", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set(httpx.HeaderLastEventId, "1")

		_, err := WithReplay(w, r, nil)
		if err == nil {
			t.Error("expected error when Last-Event-ID present but replayer is nil")
		}
		if !strings.Contains(err.Error(), "no replayer configured") {
			t.Errorf("expected error to contain 'no replayer configured', got: %v", err)
		}
	})
}
