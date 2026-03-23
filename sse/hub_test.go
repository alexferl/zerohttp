package sse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestHub(t *testing.T) {
	t.Run("registers and unregisters connections", func(t *testing.T) {
		hub := NewHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		hub.Register(stream)

		if hub.ConnectionCount() != 1 {
			t.Errorf("expected 1 connection, got %d", hub.ConnectionCount())
		}

		hub.Unregister(stream)
		if hub.ConnectionCount() != 0 {
			t.Errorf("expected 0 connections, got %d", hub.ConnectionCount())
		}
	})

	t.Run("subscribes to topics", func(t *testing.T) {
		hub := NewHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		hub.Subscribe(stream, "notifications")

		if hub.TopicCount("notifications") != 1 {
			t.Errorf("expected 1 subscriber, got %d", hub.TopicCount("notifications"))
		}

		hub.Unsubscribe(stream, "notifications")
		if hub.TopicCount("notifications") != 0 {
			t.Errorf("expected 0 subscribers, got %d", hub.TopicCount("notifications"))
		}
	})

	t.Run("broadcasts to all connections", func(t *testing.T) {
		hub := NewHub()

		// Create two streams
		w1 := httptest.NewRecorder()
		w2 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream1, _ := New(w1, r)
		stream2, _ := New(w2, r)

		hub.Register(stream1)
		hub.Register(stream2)

		hub.Broadcast(Event{Data: []byte("hello all")})

		zhtest.AssertWith(t, w1).BodyContains("hello all")
		zhtest.AssertWith(t, w2).BodyContains("hello all")
	})

	t.Run("broadcasts to topic subscribers only", func(t *testing.T) {
		hub := NewHub()

		w1 := httptest.NewRecorder()
		w2 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream1, _ := New(w1, r)
		stream2, _ := New(w2, r)

		hub.Subscribe(stream1, "topic1")
		hub.Subscribe(stream2, "topic2")

		hub.BroadcastTo("topic1", Event{Data: []byte("topic1 message")})

		zhtest.AssertWith(t, w1).BodyContains("topic1 message")
		zhtest.AssertWith(t, w2).BodyNotContains("topic1 message")
	})

	t.Run("unsubscribe removes from all topics", func(t *testing.T) {
		hub := NewHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		hub.Subscribe(stream, "topic1")
		hub.Subscribe(stream, "topic2")

		hub.Unregister(stream)

		if hub.TopicCount("topic1") != 0 {
			t.Error("expected stream to be removed from topic1")
		}
		if hub.TopicCount("topic2") != 0 {
			t.Error("expected stream to be removed from topic2")
		}
	})

	t.Run("auto-unregisters failed connections on broadcast", func(t *testing.T) {
		hub := NewHub()

		// Create a working stream
		w1 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream1, _ := New(w1, r)
		hub.Register(stream1)

		// Create a stream with an error writer that will fail on send
		header := make(http.Header)
		ew := &errorWriter{header: header}
		fw := &flusherWriter{ResponseWriter: ew, header: header}
		badStream := &SSE{
			w:       fw,
			flusher: fw,
			ctx:     r.Context(),
			closed:  make(chan struct{}),
			done:    make(chan struct{}),
			cancel:  func() {},
		}
		hub.Register(badStream)

		if hub.ConnectionCount() != 2 {
			t.Errorf("expected 2 connections, got %d", hub.ConnectionCount())
		}

		// Broadcast should auto-unregister the failed connection
		hub.Broadcast(Event{Data: []byte("test")})

		if hub.ConnectionCount() != 1 {
			t.Errorf("expected 1 connection after broadcast, got %d", hub.ConnectionCount())
		}
	})

	t.Run("auto-unregisters failed connections on broadcast to topic", func(t *testing.T) {
		hub := NewHub()

		// Create a working stream
		w1 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream1, _ := New(w1, r)
		hub.Subscribe(stream1, "test-topic")

		// Create a stream with an error writer
		header := make(http.Header)
		ew := &errorWriter{header: header}
		fw := &flusherWriter{ResponseWriter: ew, header: header}
		badStream := &SSE{
			w:       fw,
			flusher: fw,
			ctx:     r.Context(),
			closed:  make(chan struct{}),
			done:    make(chan struct{}),
			cancel:  func() {},
		}
		hub.Subscribe(badStream, "test-topic")

		if hub.TopicCount("test-topic") != 2 {
			t.Errorf("expected 2 subscribers, got %d", hub.TopicCount("test-topic"))
		}

		// Broadcast should auto-unregister the failed connection
		hub.BroadcastTo("test-topic", Event{Data: []byte("test")})

		if hub.TopicCount("test-topic") != 1 {
			t.Errorf("expected 1 subscriber after broadcast, got %d", hub.TopicCount("test-topic"))
		}
	})
}

// Test for race condition in Broadcast where connections are closed during broadcast
func TestHub_BroadcastRaceCondition(t *testing.T) {
	t.Run("broadcast with concurrent close", func(t *testing.T) {
		hub := NewHub()

		// Create multiple connections
		var streams []*SSE
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)
			stream, err := New(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			streams = append(streams, stream)
			hub.Register(stream)
		}

		if hub.ConnectionCount() != 10 {
			t.Errorf("expected 10 connections, got %d", hub.ConnectionCount())
		}

		// Run multiple iterations to increase chance of race detection
		for iter := 0; iter < 100; iter++ {
			var wg sync.WaitGroup
			wg.Add(2)

			// Broadcast from one goroutine
			go func() {
				defer wg.Done()
				hub.Broadcast(Event{Data: []byte("test")})
			}()

			// Close connections from another goroutine
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					if i%2 == 0 {
						_ = streams[i].Close()
					}
				}
			}()

			wg.Wait()
		}

		// Clean up remaining connections
		for _, stream := range streams {
			_ = stream.Close()
		}
	})

	t.Run("broadcastTo with concurrent close", func(t *testing.T) {
		hub := NewHub()

		// Create multiple connections subscribed to a topic
		var streams []*SSE
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)
			stream, err := New(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			streams = append(streams, stream)
			hub.Subscribe(stream, "test-topic")
		}

		if hub.TopicCount("test-topic") != 10 {
			t.Errorf("expected 10 subscribers, got %d", hub.TopicCount("test-topic"))
		}

		// Run multiple iterations to increase chance of race detection
		for iter := 0; iter < 100; iter++ {
			var wg sync.WaitGroup
			wg.Add(2)

			// BroadcastTo from one goroutine
			go func() {
				defer wg.Done()
				hub.BroadcastTo("test-topic", Event{Data: []byte("test")})
			}()

			// Close connections from another goroutine
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					if i%2 == 0 {
						_ = streams[i].Close()
					}
				}
			}()

			wg.Wait()
		}

		// Clean up remaining connections
		for _, stream := range streams {
			_ = stream.Close()
		}
	})

	// Stress test with many concurrent operations
	t.Run("concurrent broadcast stress test", func(t *testing.T) {
		hub := NewHub()
		var streams []*SSE

		// Create many connections
		for i := 0; i < 50; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)
			stream, err := New(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			streams = append(streams, stream)
			hub.Register(stream)
		}

		var wg sync.WaitGroup
		numWorkers := 10
		iterations := 100

		// Multiple broadcasters
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					hub.Broadcast(Event{Data: []byte("test"), ID: fmt.Sprintf("worker-%d-iter-%d", id, j)})
				}
			}(i)
		}

		// Multiple closers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					idx := (id*iterations + j) % len(streams)
					_ = streams[idx].Close()
					// Re-register sometimes to keep the pool active
					if j%3 == 0 {
						hub.Register(streams[idx])
					}
				}
			}(i)
		}

		// Multiple registrars
		for i := 0; i < numWorkers/2; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					idx := (id*iterations + j) % len(streams)
					hub.Register(streams[idx])
				}
			}(i)
		}

		wg.Wait()

		// Clean up
		for _, stream := range streams {
			_ = stream.Close()
		}
	})
}
