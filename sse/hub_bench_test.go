package sse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkHub_Broadcast measures the scaling of Hub.Broadcast()
// with different connection counts.
func BenchmarkHub_Broadcast(b *testing.B) {
	connectionCounts := []int{10, 100, 1000}

	for _, count := range connectionCounts {
		b.Run(fmt.Sprintf("Connections%d", count), func(b *testing.B) {
			hub := NewHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			// Create and register connections
			recorders := make([]*httptest.ResponseRecorder, count)
			streams := make([]*SSE, count)

			for i := range count {
				recorders[i] = httptest.NewRecorder()
				stream, err := New(recorders[i], r)
				if err != nil {
					b.Fatalf("failed to create SSE: %v", err)
				}
				streams[i] = stream
				hub.Register(stream)
			}

			// Cleanup
			defer func() {
				for _, s := range streams {
					_ = s.Close()
				}
			}()

			event := Event{Data: []byte("broadcast message")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				hub.Broadcast(event)
			}
		})
	}
}

// BenchmarkHub_BroadcastTo measures topic-based broadcast performance
// with different numbers of subscribers.
func BenchmarkHub_BroadcastTo(b *testing.B) {
	subscriberCounts := []int{10, 100, 1000}

	for _, count := range subscriberCounts {
		b.Run(fmt.Sprintf("Subscribers%d", count), func(b *testing.B) {
			hub := NewHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			// Create and subscribe connections
			recorders := make([]*httptest.ResponseRecorder, count)
			streams := make([]*SSE, count)

			for i := range count {
				recorders[i] = httptest.NewRecorder()
				stream, err := New(recorders[i], r)
				if err != nil {
					b.Fatalf("failed to create SSE: %v", err)
				}
				streams[i] = stream
				hub.Subscribe(stream, "notifications")
			}

			// Cleanup
			defer func() {
				for _, s := range streams {
					_ = s.Close()
				}
			}()

			event := Event{Data: []byte("topic message")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				hub.BroadcastTo("notifications", event)
			}
		})
	}
}

// BenchmarkHub_BroadcastTo_MultipleTopics measures broadcasting to
// different topic configurations.
func BenchmarkHub_BroadcastTo_MultipleTopics(b *testing.B) {
	scenarios := []struct {
		name            string
		numTopics       int
		subscribersEach int
	}{
		{"SingleTopic", 1, 100},
		{"TenTopics", 10, 10},
		{"HundredTopics", 100, 1},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			hub := NewHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			// Create connections and distribute across topics
			for i := 0; i < s.numTopics*s.subscribersEach; i++ {
				w := httptest.NewRecorder()
				stream, err := New(w, r)
				if err != nil {
					b.Fatalf("failed to create SSE: %v", err)
				}
				topicName := fmt.Sprintf("topic-%d", i%s.numTopics)
				hub.Subscribe(stream, topicName)
			}

			event := Event{Data: []byte("topic message")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				hub.BroadcastTo("topic-0", event)
			}
		})
	}
}

// BenchmarkHub_RegisterUnregister measures the overhead of
// connection registration and unregistration.
func BenchmarkHub_RegisterUnregister(b *testing.B) {
	b.Run("Register", func(b *testing.B) {
		hub := NewHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Pre-create streams
		streams := make([]*SSE, b.N)
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			stream, _ := New(w, r)
			streams[i] = stream
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			hub.Register(streams[i])
		}

		// Cleanup
		for _, s := range streams {
			_ = s.Close()
		}
	})

	b.Run("Unregister", func(b *testing.B) {
		hub := NewHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Pre-create and register streams
		streams := make([]*SSE, b.N)
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			stream, _ := New(w, r)
			streams[i] = stream
			hub.Register(stream)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			hub.Unregister(streams[i])
		}

		// Cleanup
		for _, s := range streams {
			_ = s.Close()
		}
	})

	b.Run("Subscribe", func(b *testing.B) {
		hub := NewHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Pre-create streams
		streams := make([]*SSE, b.N)
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			stream, _ := New(w, r)
			streams[i] = stream
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			hub.Subscribe(streams[i], "test-topic")
		}

		// Cleanup
		for _, s := range streams {
			_ = s.Close()
		}
	})
}

// BenchmarkHub_RegisterUnregister_Concurrent measures concurrent
// registration/unregistration performance.
func BenchmarkHub_RegisterUnregister_Concurrent(b *testing.B) {
	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			hub := NewHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					w := httptest.NewRecorder()
					stream, _ := New(w, r)
					hub.Register(stream)

					// Alternate between register/unregister
					if i%2 == 0 {
						hub.Unregister(stream)
					}
					i++
					_ = stream.Close()
				}
			})
		})
	}
}

// BenchmarkHub_Broadcast_Stress stress tests the broadcast mechanism
// with many concurrent broadcasts.
func BenchmarkHub_Broadcast_Stress(b *testing.B) {
	b.Run("ConcurrentBroadcasts", func(b *testing.B) {
		hub := NewHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Create 100 connections
		streams := make([]*SSE, 100)
		for i := range 100 {
			w := httptest.NewRecorder()
			stream, _ := New(w, r)
			streams[i] = stream
			hub.Register(stream)
		}

		defer func() {
			for _, s := range streams {
				_ = s.Close()
			}
		}()

		event := Event{Data: []byte("stress test")}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				hub.Broadcast(event)
			}
		})
	})
}

// BenchmarkHub_Baseline compares Hub broadcast against naive iteration
// to measure the management overhead.
func BenchmarkHub_Baseline(b *testing.B) {
	connectionCounts := []int{10, 100}

	for _, count := range connectionCounts {
		b.Run(fmt.Sprintf("Connections%d", count), func(b *testing.B) {
			b.Run("NaiveIteration", func(b *testing.B) {
				r := httptest.NewRequest(http.MethodGet, "/sse", nil)

				// Create connections
				recorders := make([]*httptest.ResponseRecorder, count)
				streams := make([]*SSE, count)
				for i := range count {
					recorders[i] = httptest.NewRecorder()
					stream, _ := New(recorders[i], r)
					streams[i] = stream
				}

				defer func() {
					for _, s := range streams {
						_ = s.Close()
					}
				}()

				event := Event{Data: []byte("broadcast message")}

				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					for _, s := range streams {
						_ = s.Send(event)
					}
				}
			})

			b.Run("Hub", func(b *testing.B) {
				hub := NewHub()
				r := httptest.NewRequest(http.MethodGet, "/sse", nil)

				// Create and register connections
				recorders := make([]*httptest.ResponseRecorder, count)
				streams := make([]*SSE, count)
				for i := range count {
					recorders[i] = httptest.NewRecorder()
					stream, _ := New(recorders[i], r)
					streams[i] = stream
					hub.Register(stream)
				}

				defer func() {
					for _, s := range streams {
						_ = s.Close()
					}
				}()

				event := Event{Data: []byte("broadcast message")}

				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					hub.Broadcast(event)
				}
			})
		})
	}
}
