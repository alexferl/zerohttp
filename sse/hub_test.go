package sse

import (
	"context"
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
		hub.Register(context.Background(), stream)

		zhtest.AssertEqual(t, 1, hub.ConnectionCount())

		hub.Unregister(context.Background(), stream)
		zhtest.AssertEqual(t, 0, hub.ConnectionCount())
	})

	t.Run("subscribes to topics", func(t *testing.T) {
		hub := NewHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		hub.Subscribe(context.Background(), stream, "notifications")

		zhtest.AssertEqual(t, 1, hub.TopicCount("notifications"))

		hub.Unsubscribe(context.Background(), stream, "notifications")
		zhtest.AssertEqual(t, 0, hub.TopicCount("notifications"))
	})

	t.Run("broadcasts to all connections", func(t *testing.T) {
		hub := NewHub()

		// Create two streams
		w1 := httptest.NewRecorder()
		w2 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream1, _ := New(w1, r)
		stream2, _ := New(w2, r)

		hub.Register(context.Background(), stream1)
		hub.Register(context.Background(), stream2)

		hub.Broadcast(context.Background(), Event{Data: []byte("hello all")})

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

		hub.Subscribe(context.Background(), stream1, "topic1")
		hub.Subscribe(context.Background(), stream2, "topic2")

		hub.BroadcastTo(context.Background(), "topic1", Event{Data: []byte("topic1 message")})

		zhtest.AssertWith(t, w1).BodyContains("topic1 message")
		zhtest.AssertWith(t, w2).BodyNotContains("topic1 message")
	})

	t.Run("unsubscribe removes from all topics", func(t *testing.T) {
		hub := NewHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		hub.Subscribe(context.Background(), stream, "topic1")
		hub.Subscribe(context.Background(), stream, "topic2")

		hub.Unregister(context.Background(), stream)

		zhtest.AssertEqual(t, 0, hub.TopicCount("topic1"))
		zhtest.AssertEqual(t, 0, hub.TopicCount("topic2"))
	})

	t.Run("auto-unregisters failed connections on broadcast", func(t *testing.T) {
		hub := NewHub()

		// Create a working stream
		w1 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream1, _ := New(w1, r)
		hub.Register(context.Background(), stream1)

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
		hub.Register(context.Background(), badStream)

		zhtest.AssertEqual(t, 2, hub.ConnectionCount())

		// Broadcast should auto-unregister the failed connection
		hub.Broadcast(context.Background(), Event{Data: []byte("test")})

		zhtest.AssertEqual(t, 1, hub.ConnectionCount())
	})

	t.Run("auto-unregisters failed connections on broadcast to topic", func(t *testing.T) {
		hub := NewHub()

		// Create a working stream
		w1 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream1, _ := New(w1, r)
		hub.Subscribe(context.Background(), stream1, "test-topic")

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
		hub.Subscribe(context.Background(), badStream, "test-topic")

		zhtest.AssertEqual(t, 2, hub.TopicCount("test-topic"))

		// Broadcast should auto-unregister the failed connection
		hub.BroadcastTo(context.Background(), "test-topic", Event{Data: []byte("test")})

		zhtest.AssertEqual(t, 1, hub.TopicCount("test-topic"))
	})

	t.Run("calls OnBroadcast hook", func(t *testing.T) {
		hub := NewHub()

		var called bool
		var received Event
		hub.OnBroadcast = func(event Event) {
			called = true
			received = event
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream, _ := New(w, r)
		hub.Register(context.Background(), stream)

		event := Event{Data: []byte("hook test")}
		hub.Broadcast(context.Background(), event)

		zhtest.AssertEqual(t, true, called)
		zhtest.AssertEqual(t, string(event.Data), string(received.Data))
	})

	t.Run("calls OnBroadcastTo hook", func(t *testing.T) {
		hub := NewHub()

		var called bool
		var receivedTopic string
		var received Event
		hub.OnBroadcastTo = func(topic string, event Event) {
			called = true
			receivedTopic = topic
			received = event
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream, _ := New(w, r)
		hub.Subscribe(context.Background(), stream, "test-topic")

		event := Event{Data: []byte("hook test")}
		hub.BroadcastTo(context.Background(), "test-topic", event)

		zhtest.AssertEqual(t, true, called)
		zhtest.AssertEqual(t, "test-topic", receivedTopic)
		zhtest.AssertEqual(t, string(event.Data), string(received.Data))
	})

	t.Run("custom broadcaster implements interface", func(t *testing.T) {
		// Ensure a custom implementation can satisfy the Broadcaster interface
		var _ Broadcaster = (*customBroadcaster)(nil)
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
			zhtest.AssertNoError(t, err)
			streams = append(streams, stream)
			hub.Register(context.Background(), stream)
		}

		zhtest.AssertEqual(t, 10, hub.ConnectionCount())

		// Run multiple iterations to increase chance of race detection
		for iter := 0; iter < 100; iter++ {
			var wg sync.WaitGroup
			wg.Add(2)

			// Broadcast from one goroutine
			go func() {
				defer wg.Done()
				hub.Broadcast(context.Background(), Event{Data: []byte("test")})
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
			zhtest.AssertNoError(t, err)
			streams = append(streams, stream)
			hub.Subscribe(context.Background(), stream, "test-topic")
		}

		zhtest.AssertEqual(t, 10, hub.TopicCount("test-topic"))

		// Run multiple iterations to increase chance of race detection
		for iter := 0; iter < 100; iter++ {
			var wg sync.WaitGroup
			wg.Add(2)

			// BroadcastTo from one goroutine
			go func() {
				defer wg.Done()
				hub.BroadcastTo(context.Background(), "test-topic", Event{Data: []byte("test")})
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
			zhtest.AssertNoError(t, err)
			streams = append(streams, stream)
			hub.Register(context.Background(), stream)
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
					hub.Broadcast(context.Background(), Event{Data: []byte("test"), ID: fmt.Sprintf("worker-%d-iter-%d", id, j)})
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
						hub.Register(context.Background(), streams[idx])
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
					hub.Register(context.Background(), streams[idx])
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

// customBroadcaster is a minimal implementation of the Broadcaster interface for testing.
type customBroadcaster struct{}

func (c *customBroadcaster) Register(_ context.Context, s *SSE)                       {}
func (c *customBroadcaster) Unregister(_ context.Context, s *SSE)                     {}
func (c *customBroadcaster) Subscribe(_ context.Context, s *SSE, topic string)        {}
func (c *customBroadcaster) Unsubscribe(_ context.Context, s *SSE, topic string)      {}
func (c *customBroadcaster) Broadcast(_ context.Context, event Event)                 {}
func (c *customBroadcaster) BroadcastTo(_ context.Context, topic string, event Event) {}
func (c *customBroadcaster) ConnectionCount() int                                     { return 0 }
func (c *customBroadcaster) TopicCount(topic string) int                              { return 0 }
