package zerohttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
)

// BenchmarkSSE_Send measures the throughput of the Send() method
// with different event sizes and field combinations.
func BenchmarkSSE_Send(b *testing.B) {
	scenarios := []struct {
		name  string
		event SSEEvent
	}{
		{
			name: "SimpleDataOnly",
			event: SSEEvent{
				Data: []byte("hello world"),
			},
		},
		{
			name: "WithID",
			event: SSEEvent{
				ID:   "12345",
				Data: []byte("hello world"),
			},
		},
		{
			name: "WithName",
			event: SSEEvent{
				Name: "update",
				Data: []byte("hello world"),
			},
		},
		{
			name: "WithRetry",
			event: SSEEvent{
				Data:  []byte("hello world"),
				Retry: 5000 * time.Millisecond,
			},
		},
		{
			name: "FullEvent",
			event: SSEEvent{
				ID:    "12345",
				Name:  "update",
				Data:  []byte("hello world"),
				Retry: 5000 * time.Millisecond,
			},
		},
		{
			name: "LargeData_1KB",
			event: SSEEvent{
				Data: make([]byte, 1024),
			},
		},
		{
			name: "LargeData_10KB",
			event: SSEEvent{
				Data: make([]byte, 10*1024),
			},
		},
		{
			name: "MultiLineData",
			event: SSEEvent{
				Data: []byte("line1\nline2\nline3\nline4\nline5"),
			},
		},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			stream, err := NewSSE(w, r)
			if err != nil {
				b.Fatalf("failed to create SSE: %v", err)
			}
			defer func() { _ = stream.Close() }()

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				if err := stream.Send(s.event); err != nil {
					b.Fatalf("send failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkSSE_SendComment measures the throughput of sending comments.
func BenchmarkSSE_SendComment(b *testing.B) {
	comments := []struct {
		name    string
		comment string
	}{
		{"Short", "ping"},
		{"Medium", "keepalive heartbeat message"},
		{"Long", "this is a longer comment that might be used for keepalive purposes in production scenarios"},
	}

	for _, c := range comments {
		b.Run(c.name, func(b *testing.B) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			stream, err := NewSSE(w, r)
			if err != nil {
				b.Fatalf("failed to create SSE: %v", err)
			}
			defer func() { _ = stream.Close() }()

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				if err := stream.SendComment(c.comment); err != nil {
					b.Fatalf("send comment failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkSSEHub_Broadcast measures the scaling of SSEHub.Broadcast()
// with different connection counts.
func BenchmarkSSEHub_Broadcast(b *testing.B) {
	connectionCounts := []int{10, 100, 1000}

	for _, count := range connectionCounts {
		b.Run(fmt.Sprintf("Connections%d", count), func(b *testing.B) {
			hub := NewSSEHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			// Create and register connections
			recorders := make([]*httptest.ResponseRecorder, count)
			streams := make([]*SSE, count)

			for i := range count {
				recorders[i] = httptest.NewRecorder()
				stream, err := NewSSE(recorders[i], r)
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

			event := SSEEvent{Data: []byte("broadcast message")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				hub.Broadcast(event)
			}
		})
	}
}

// BenchmarkSSEHub_BroadcastTo measures topic-based broadcast performance
// with different numbers of subscribers.
func BenchmarkSSEHub_BroadcastTo(b *testing.B) {
	subscriberCounts := []int{10, 100, 1000}

	for _, count := range subscriberCounts {
		b.Run(fmt.Sprintf("Subscribers%d", count), func(b *testing.B) {
			hub := NewSSEHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			// Create and subscribe connections
			recorders := make([]*httptest.ResponseRecorder, count)
			streams := make([]*SSE, count)

			for i := range count {
				recorders[i] = httptest.NewRecorder()
				stream, err := NewSSE(recorders[i], r)
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

			event := SSEEvent{Data: []byte("topic message")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				hub.BroadcastTo("notifications", event)
			}
		})
	}
}

// BenchmarkSSEHub_BroadcastTo_MultipleTopics measures broadcasting to
// different topic configurations.
func BenchmarkSSEHub_BroadcastTo_MultipleTopics(b *testing.B) {
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
			hub := NewSSEHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			// Create connections and distribute across topics
			for i := 0; i < s.numTopics*s.subscribersEach; i++ {
				w := httptest.NewRecorder()
				stream, err := NewSSE(w, r)
				if err != nil {
					b.Fatalf("failed to create SSE: %v", err)
				}
				topicName := fmt.Sprintf("topic-%d", i%s.numTopics)
				hub.Subscribe(stream, topicName)
			}

			event := SSEEvent{Data: []byte("topic message")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				hub.BroadcastTo("topic-0", event)
			}
		})
	}
}

// BenchmarkInMemoryReplayer_Store measures Store() performance
// with different buffer sizes.
func BenchmarkInMemoryReplayer_Store(b *testing.B) {
	bufferSizes := []int{10, 100, 1000, 0}

	for _, size := range bufferSizes {
		name := fmt.Sprintf("MaxEvents%d", size)
		if size == 0 {
			name = "Unlimited"
		}

		b.Run(name, func(b *testing.B) {
			replayer := NewInMemoryReplayer(size, 0)
			event := SSEEvent{Data: []byte("test data")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				replayer.Store(event)
			}
		})
	}
}

// BenchmarkInMemoryReplayer_Store_WithTTL measures Store() performance
// when TTL-based expiration is enabled.
func BenchmarkInMemoryReplayer_Store_WithTTL(b *testing.B) {
	ttls := []time.Duration{
		0,               // No TTL
		time.Minute,     // 1 minute
		5 * time.Minute, // 5 minutes
	}

	for _, ttl := range ttls {
		name := "NoTTL"
		if ttl > 0 {
			name = fmt.Sprintf("TTL%s", ttl)
		}

		b.Run(name, func(b *testing.B) {
			replayer := NewInMemoryReplayer(1000, ttl)
			event := SSEEvent{Data: []byte("test data")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				replayer.Store(event)
			}
		})
	}
}

// BenchmarkInMemoryReplayer_Replay measures Replay() performance
// with different event counts.
func BenchmarkInMemoryReplayer_Replay(b *testing.B) {
	eventCounts := []int{10, 100, 1000}

	for _, count := range eventCounts {
		b.Run(fmt.Sprintf("Events%d", count), func(b *testing.B) {
			replayer := NewInMemoryReplayer(count, 0)

			// Pre-populate with events
			for i := range count {
				replayer.Store(SSEEvent{Data: fmt.Appendf(nil, "event %d", i)})
			}

			sendFunc := func(e SSEEvent) error { return nil }

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = replayer.Replay("0", sendFunc)
			}
		})
	}
}

// BenchmarkInMemoryReplayer_Replay_WithAfterID measures Replay() performance
// when replaying from a specific ID.
func BenchmarkInMemoryReplayer_Replay_WithAfterID(b *testing.B) {
	eventCounts := []int{100, 1000}

	for _, count := range eventCounts {
		b.Run(fmt.Sprintf("Total%d", count), func(b *testing.B) {
			replayer := NewInMemoryReplayer(count, 0)

			// Pre-populate with events
			for i := range count {
				replayer.Store(SSEEvent{Data: fmt.Appendf(nil, "event %d", i)})
			}

			sendFunc := func(e SSEEvent) error { return nil }

			b.Run("ReplayHalf", func(b *testing.B) {
				afterID := fmt.Sprintf("%d", count/2)

				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					_, _ = replayer.Replay(afterID, sendFunc)
				}
			})

			b.Run("ReplayAll", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					_, _ = replayer.Replay("", sendFunc)
				}
			})
		})
	}
}

// BenchmarkSSEHub_RegisterUnregister measures the overhead of
// connection registration and unregistration.
func BenchmarkSSEHub_RegisterUnregister(b *testing.B) {
	b.Run("Register", func(b *testing.B) {
		hub := NewSSEHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Pre-create streams
		streams := make([]*SSE, b.N)
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			stream, _ := NewSSE(w, r)
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
		hub := NewSSEHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Pre-create and register streams
		streams := make([]*SSE, b.N)
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			stream, _ := NewSSE(w, r)
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
		hub := NewSSEHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Pre-create streams
		streams := make([]*SSE, b.N)
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			stream, _ := NewSSE(w, r)
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

// BenchmarkSSEHub_RegisterUnregister_Concurrent measures concurrent
// registration/unregistration performance.
func BenchmarkSSEHub_RegisterUnregister_Concurrent(b *testing.B) {
	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			hub := NewSSEHub()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					w := httptest.NewRecorder()
					stream, _ := NewSSE(w, r)
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

// BenchmarkSSE_MemoryPerConnection measures memory allocation per connection.
func BenchmarkSSE_MemoryPerConnection(b *testing.B) {
	b.Run("CreateAndClose", func(b *testing.B) {
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			stream, _ := NewSSE(w, r)
			_ = stream.Close()
			stream.WaitDone()
		}
	})

	b.Run("MemoryGrowth_100Connections", func(b *testing.B) {
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Force GC and get baseline
		runtime.GC()
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		streams := make([]*SSE, 100)
		for i := range 100 {
			w := httptest.NewRecorder()
			stream, _ := NewSSE(w, r)
			streams[i] = stream
		}

		runtime.ReadMemStats(&m2)

		// Cleanup
		for _, s := range streams {
			_ = s.Close()
			s.WaitDone()
		}

		// Report bytes per connection
		bytesPerConn := int64(m2.TotalAlloc-m1.TotalAlloc) / 100
		b.ReportMetric(float64(bytesPerConn), "bytes/conn")
	})
}

// BenchmarkSSEWriter_WriteEvent measures SSEWriter performance.
func BenchmarkSSEWriter_WriteEvent(b *testing.B) {
	scenarios := []struct {
		name  string
		event SSEEvent
	}{
		{
			name: "Simple",
			event: SSEEvent{
				Data: []byte("hello world"),
			},
		},
		{
			name: "Full",
			event: SSEEvent{
				ID:    "123",
				Name:  "update",
				Data:  []byte("hello world"),
				Retry: 5000 * time.Millisecond,
			},
		},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			writer, err := NewSSEWriter(w, r)
			if err != nil {
				b.Fatalf("failed to create SSEWriter: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				if err := writer.WriteEvent(s.event); err != nil {
					b.Fatalf("write event failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkSSE_EventTypes compares performance of different event types.
func BenchmarkSSE_EventTypes(b *testing.B) {
	b.Run("SmallData", func(b *testing.B) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := NewSSE(w, r)
		defer func() { _ = stream.Close() }()

		event := SSEEvent{Data: []byte("x")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = stream.Send(event)
		}
	})

	b.Run("MediumData", func(b *testing.B) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := NewSSE(w, r)
		defer func() { _ = stream.Close() }()

		event := SSEEvent{Data: []byte("this is a medium sized message")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = stream.Send(event)
		}
	})

	b.Run("JSONData", func(b *testing.B) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := NewSSE(w, r)
		defer func() { _ = stream.Close() }()

		event := SSEEvent{
			Name: "update",
			Data: []byte(`{"id":123,"type":"notification","message":"Hello World","timestamp":"2024-01-01T00:00:00Z"}`),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = stream.Send(event)
		}
	})
}

// BenchmarkSSEProvider compares SSE provider implementations.
func BenchmarkSSEProvider(b *testing.B) {
	providers := []struct {
		name     string
		provider config.SSEProvider
	}{
		{"DefaultProvider", NewDefaultProvider()},
	}

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				conn, err := p.provider.NewSSE(w, r)
				if err != nil {
					b.Fatalf("failed to create SSE: %v", err)
				}
				_ = conn.Close()
			}
		})
	}
}

// BenchmarkSSEHub_Broadcast_Stress stress tests the broadcast mechanism
// with many concurrent broadcasts.
func BenchmarkSSEHub_Broadcast_Stress(b *testing.B) {
	b.Run("ConcurrentBroadcasts", func(b *testing.B) {
		hub := NewSSEHub()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Create 100 connections
		streams := make([]*SSE, 100)
		for i := range 100 {
			w := httptest.NewRecorder()
			stream, _ := NewSSE(w, r)
			streams[i] = stream
			hub.Register(stream)
		}

		defer func() {
			for _, s := range streams {
				_ = s.Close()
			}
		}()

		event := SSEEvent{Data: []byte("stress test")}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				hub.Broadcast(event)
			}
		})
	})
}

// BenchmarkSSE_Baseline compares SSE performance against raw http.ResponseWriter
// to measure the overhead of the SSE implementation.
func BenchmarkSSE_Baseline(b *testing.B) {
	b.Run("RawResponseWriter_Write", func(b *testing.B) {
		data := []byte("data: hello world\n\n")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		}
	})

	b.Run("RawResponseWriter_WriteWithFlush", func(b *testing.B) {
		data := []byte("data: hello world\n\n")
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.Header().Set(httpx.HeaderContentType, "text/event-stream")
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(data)
				if f, ok := rw.(http.Flusher); ok {
					f.Flush()
				}
			})
			handler.ServeHTTP(w, r)
		}
	})

	b.Run("SSE_Send_SimpleData", func(b *testing.B) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			b.Fatalf("failed to create SSE: %v", err)
		}
		defer func() { _ = stream.Close() }()

		event := SSEEvent{Data: []byte("hello world")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			if err := stream.Send(event); err != nil {
				b.Fatalf("send failed: %v", err)
			}
		}
	})

	b.Run("SSE_OverheadFactor", func(b *testing.B) {
		// Measure the overhead ratio of SSE vs raw writes
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		data := []byte("data: hello world\n\n")
		event := SSEEvent{Data: []byte("hello world")}

		b.Run("Raw", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			}
		})

		b.Run("SSE", func(b *testing.B) {
			w := httptest.NewRecorder()
			stream, _ := NewSSE(w, r)
			defer func() { _ = stream.Close() }()

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = stream.Send(event)
			}
		})
	})
}

// BenchmarkSSEHub_Baseline compares SSEHub broadcast against naive iteration
// to measure the management overhead.
func BenchmarkSSEHub_Baseline(b *testing.B) {
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
					stream, _ := NewSSE(recorders[i], r)
					streams[i] = stream
				}

				defer func() {
					for _, s := range streams {
						_ = s.Close()
					}
				}()

				event := SSEEvent{Data: []byte("broadcast message")}

				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					for _, s := range streams {
						_ = s.Send(event)
					}
				}
			})

			b.Run("SSEHub", func(b *testing.B) {
				hub := NewSSEHub()
				r := httptest.NewRequest(http.MethodGet, "/sse", nil)

				// Create and register connections
				recorders := make([]*httptest.ResponseRecorder, count)
				streams := make([]*SSE, count)
				for i := range count {
					recorders[i] = httptest.NewRecorder()
					stream, _ := NewSSE(recorders[i], r)
					streams[i] = stream
					hub.Register(stream)
				}

				defer func() {
					for _, s := range streams {
						_ = s.Close()
					}
				}()

				event := SSEEvent{Data: []byte("broadcast message")}

				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					hub.Broadcast(event)
				}
			})
		})
	}
}

// BenchmarkInMemoryReplayer_Baseline compares replayer against a simple slice.
func BenchmarkInMemoryReplayer_Baseline(b *testing.B) {
	b.Run("SimpleSlice_Store", func(b *testing.B) {
		events := make([]SSEEvent, 0, 1000)
		event := SSEEvent{Data: []byte("test data")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			events = append(events, event)
			// Reset when full to simulate circular buffer
			if len(events) >= 1000 {
				events = events[:0]
			}
		}
	})

	b.Run("InMemoryReplayer_Store", func(b *testing.B) {
		replayer := NewInMemoryReplayer(1000, 0)
		event := SSEEvent{Data: []byte("test data")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			replayer.Store(event)
		}
	})

	b.Run("SimpleSlice_Iteration", func(b *testing.B) {
		events := make([]SSEEvent, 100)
		for i := range 100 {
			events[i] = SSEEvent{ID: fmt.Sprintf("%d", i+1), Data: []byte("test")}
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for _, e := range events {
				_ = e
			}
		}
	})

	b.Run("InMemoryReplayer_Replay", func(b *testing.B) {
		replayer := NewInMemoryReplayer(100, 0)
		for range 100 {
			replayer.Store(SSEEvent{Data: []byte("test")})
		}

		sendFunc := func(e SSEEvent) error { return nil }

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_, _ = replayer.Replay("", sendFunc)
		}
	})
}
