package sse

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkMemoryReplayer_Store measures Store() performance
// with different buffer sizes.
func BenchmarkMemoryReplayer_Store(b *testing.B) {
	bufferSizes := []int{10, 100, 1000, 0}

	for _, size := range bufferSizes {
		name := fmt.Sprintf("MaxEvents%d", size)
		if size == 0 {
			name = "Unlimited"
		}

		b.Run(name, func(b *testing.B) {
			replayer := NewMemoryReplayer(size, 0)
			event := Event{Data: []byte("test data")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				replayer.Store(event)
			}
		})
	}
}

// BenchmarkMemoryReplayer_Store_WithTTL measures Store() performance
// when TTL-based expiration is enabled.
func BenchmarkMemoryReplayer_Store_WithTTL(b *testing.B) {
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
			replayer := NewMemoryReplayer(1000, ttl)
			event := Event{Data: []byte("test data")}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				replayer.Store(event)
			}
		})
	}
}

// BenchmarkMemoryReplayer_Replay measures Replay() performance
// with different event counts.
func BenchmarkMemoryReplayer_Replay(b *testing.B) {
	eventCounts := []int{10, 100, 1000}

	for _, count := range eventCounts {
		b.Run(fmt.Sprintf("Events%d", count), func(b *testing.B) {
			replayer := NewMemoryReplayer(count, 0)

			// Pre-populate with events
			for i := range count {
				replayer.Store(Event{Data: fmt.Appendf(nil, "event %d", i)})
			}

			sendFunc := func(e Event) error { return nil }

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = replayer.Replay("0", sendFunc)
			}
		})
	}
}

// BenchmarkMemoryReplayer_Replay_WithAfterID measures Replay() performance
// when replaying from a specific ID.
func BenchmarkMemoryReplayer_Replay_WithAfterID(b *testing.B) {
	eventCounts := []int{100, 1000}

	for _, count := range eventCounts {
		b.Run(fmt.Sprintf("Total%d", count), func(b *testing.B) {
			replayer := NewMemoryReplayer(count, 0)

			// Pre-populate with events
			for i := range count {
				replayer.Store(Event{Data: fmt.Appendf(nil, "event %d", i)})
			}

			sendFunc := func(e Event) error { return nil }

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

// BenchmarkMemoryReplayer_Baseline compares replayer against a simple slice.
func BenchmarkMemoryReplayer_Baseline(b *testing.B) {
	b.Run("SimpleSlice_Store", func(b *testing.B) {
		events := make([]Event, 0, 1000)
		event := Event{Data: []byte("test data")}

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

	b.Run("MemoryReplayer_Store", func(b *testing.B) {
		replayer := NewMemoryReplayer(1000, 0)
		event := Event{Data: []byte("test data")}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			replayer.Store(event)
		}
	})

	b.Run("SimpleSlice_Iteration", func(b *testing.B) {
		events := make([]Event, 100)
		for i := range 100 {
			events[i] = Event{ID: fmt.Sprintf("%d", i+1), Data: []byte("test")}
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for _, e := range events {
				_ = e
			}
		}
	})

	b.Run("MemoryReplayer_Replay", func(b *testing.B) {
		replayer := NewMemoryReplayer(100, 0)
		for range 100 {
			replayer.Store(Event{Data: []byte("test")})
		}

		sendFunc := func(e Event) error { return nil }

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_, _ = replayer.Replay("", sendFunc)
		}
	})
}
