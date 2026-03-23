package sse

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

// BenchmarkSSE_Send measures the throughput of the Send() method
// with different event sizes and field combinations.
func BenchmarkSSE_Send(b *testing.B) {
	scenarios := []struct {
		name  string
		event Event
	}{
		{
			name: "SimpleDataOnly",
			event: Event{
				Data: []byte("hello world"),
			},
		},
		{
			name: "WithID",
			event: Event{
				ID:   "12345",
				Data: []byte("hello world"),
			},
		},
		{
			name: "WithName",
			event: Event{
				Name: "update",
				Data: []byte("hello world"),
			},
		},
		{
			name: "WithRetry",
			event: Event{
				Data:  []byte("hello world"),
				Retry: 5000 * time.Millisecond,
			},
		},
		{
			name: "FullEvent",
			event: Event{
				ID:    "12345",
				Name:  "update",
				Data:  []byte("hello world"),
				Retry: 5000 * time.Millisecond,
			},
		},
		{
			name: "LargeData_1KB",
			event: Event{
				Data: make([]byte, 1024),
			},
		},
		{
			name: "LargeData_10KB",
			event: Event{
				Data: make([]byte, 10*1024),
			},
		},
		{
			name: "MultiLineData",
			event: Event{
				Data: []byte("line1\nline2\nline3\nline4\nline5"),
			},
		},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			stream, err := New(w, r)
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

			stream, err := New(w, r)
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

// BenchmarkSSE_MemoryPerConnection measures memory allocation per connection.
func BenchmarkSSE_MemoryPerConnection(b *testing.B) {
	b.Run("CreateAndClose", func(b *testing.B) {
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			stream, _ := New(w, r)
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
			stream, _ := New(w, r)
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

// BenchmarkSSE_EventTypes compares performance of different event types.
func BenchmarkSSE_EventTypes(b *testing.B) {
	b.Run("SmallData", func(b *testing.B) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		defer func() { _ = stream.Close() }()

		event := Event{Data: []byte("x")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = stream.Send(event)
		}
	})

	b.Run("MediumData", func(b *testing.B) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		defer func() { _ = stream.Close() }()

		event := Event{Data: []byte("this is a medium sized message")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = stream.Send(event)
		}
	})

	b.Run("JSONData", func(b *testing.B) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := New(w, r)
		defer func() { _ = stream.Close() }()

		event := Event{
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
		provider Provider
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
				conn, err := p.provider.New(w, r)
				if err != nil {
					b.Fatalf("failed to create SSE: %v", err)
				}
				_ = conn.Close()
			}
		})
	}
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

		stream, err := New(w, r)
		if err != nil {
			b.Fatalf("failed to create SSE: %v", err)
		}
		defer func() { _ = stream.Close() }()

		event := Event{Data: []byte("hello world")}

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
		event := Event{Data: []byte("hello world")}

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
			stream, _ := New(w, r)
			defer func() { _ = stream.Close() }()

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = stream.Send(event)
			}
		})
	})
}
