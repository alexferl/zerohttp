package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkWriter_WriteEvent measures Writer performance.
func BenchmarkWriter_WriteEvent(b *testing.B) {
	scenarios := []struct {
		name  string
		event Event
	}{
		{
			name: "Simple",
			event: Event{
				Data: []byte("hello world"),
			},
		},
		{
			name: "Full",
			event: Event{
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

			writer, err := NewWriter(w, r)
			if err != nil {
				b.Fatalf("failed to create Writer: %v", err)
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
