package middleware

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkInMemoryStore_Algorithms compares the three rate limiting algorithms
// to help users choose the right one for their use case.
func BenchmarkInMemoryStore_Algorithms(b *testing.B) {
	algorithms := []struct {
		name      string
		algorithm config.RateLimitAlgorithm
	}{
		{"TokenBucket", config.TokenBucket},
		{"FixedWindow", config.FixedWindow},
		{"SlidingWindow", config.SlidingWindow},
	}

	for _, alg := range algorithms {
		b.Run(alg.name, func(b *testing.B) {
			store := NewInMemoryStore(alg.algorithm, time.Minute, 1000, 10000)
			ctx := context.Background()
			now := time.Now()
			key := "single-key"

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				store.CheckAndRecord(ctx, key, now)
			}
		})
	}
}

// BenchmarkInMemoryStore_KeyCount measures performance as the number of
// tracked keys increases. This helps understand memory and lookup overhead.
func BenchmarkInMemoryStore_KeyCount(b *testing.B) {
	keyCounts := []int{1, 10, 100, 1000, 10000}

	for _, count := range keyCounts {
		b.Run(fmt.Sprintf("Keys%d", count), func(b *testing.B) {
			store := NewInMemoryStore(config.TokenBucket, time.Minute, 1000, count*2)
			ctx := context.Background()
			now := time.Now()

			// Pre-populate with keys
			keys := make([]string, count)
			for i := 0; i < count; i++ {
				keys[i] = fmt.Sprintf("key-%d", i)
				store.CheckAndRecord(ctx, keys[i], now)
			}

			b.ReportAllocs()
			b.ResetTimer()

			i := 0
			for b.Loop() {
				store.CheckAndRecord(ctx, keys[i%count], now)
				i++
			}
		})
	}
}

// BenchmarkInMemoryStore_Concurrent measures performance under concurrent access
// which is the typical production scenario.
func BenchmarkInMemoryStore_Concurrent(b *testing.B) {
	concurrencyLevels := []int{1, 10, 100, 1000}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			store := NewInMemoryStore(config.TokenBucket, time.Minute, 1000, 10000)
			ctx := context.Background()
			now := time.Now()

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				key := fmt.Sprintf("key-%d", concurrency)
				for pb.Next() {
					store.CheckAndRecord(ctx, key, now)
				}
			})
		})
	}
}

// BenchmarkInMemoryStore_Concurrent_DifferentKeys measures concurrent performance
// with different key distributions (some keys hot, others cold).
func BenchmarkInMemoryStore_Concurrent_DifferentKeys(b *testing.B) {
	scenarios := []struct {
		name    string
		numKeys int
		hotKeys int // Number of keys that get 90% of traffic
	}{
		{"SingleKey", 1, 1},
		{"10Keys_Hot", 10, 2},
		{"100Keys_Hot", 100, 10},
		{"1000Keys_Uniform", 1000, 1000}, // Uniform distribution
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			store := NewInMemoryStore(config.TokenBucket, time.Minute, 1000, scenario.numKeys*2)
			ctx := context.Background()
			now := time.Now()

			keys := make([]string, scenario.numKeys)
			for i := 0; i < scenario.numKeys; i++ {
				keys[i] = fmt.Sprintf("key-%d", i)
			}

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					// 90% of requests go to hot keys
					var key string
					if i%10 < 9 {
						key = keys[i%scenario.hotKeys]
					} else {
						key = keys[i%scenario.numKeys]
					}
					store.CheckAndRecord(ctx, key, now)
					i++
				}
			})
		})
	}
}

// BenchmarkInMemoryStore_TokenBucketRefill measures token bucket refill performance
// over time as tokens are consumed and refilled.
func BenchmarkInMemoryStore_TokenBucketRefill(b *testing.B) {
	store := NewInMemoryStore(config.TokenBucket, time.Second, 1000, 10000)
	ctx := context.Background()
	baseTime := time.Now()
	key := "test-key"

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		// Simulate time passing and tokens being refilled
		now := baseTime.Add(time.Duration(b.N%1000) * time.Millisecond)
		store.CheckAndRecord(ctx, key, now)
	}
}

// BenchmarkInMemoryStore_SlidingWindowExpiration measures sliding window performance
// with timestamp expiration (removing old entries).
func BenchmarkInMemoryStore_SlidingWindowExpiration(b *testing.B) {
	window := 100 * time.Millisecond
	store := NewInMemoryStore(config.SlidingWindow, window, 100, 10000)
	ctx := context.Background()
	baseTime := time.Now()
	key := "test-key"

	// Pre-fill with some timestamps
	for i := 0; i < 50; i++ {
		store.CheckAndRecord(ctx, key, baseTime.Add(time.Duration(i)*time.Millisecond))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		now := baseTime.Add(time.Duration(b.N%200) * time.Millisecond)
		store.CheckAndRecord(ctx, key, now)
	}
}

// BenchmarkInMemoryStore_Eviction measures performance when keys are being
// evicted due to reaching max keys limit.
func BenchmarkInMemoryStore_Eviction(b *testing.B) {
	maxKeys := 100
	store := NewInMemoryStore(config.TokenBucket, time.Minute, 100, maxKeys)
	ctx := context.Background()
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()

	// Each iteration creates a new key, causing evictions
	for b.Loop() {
		key := fmt.Sprintf("key-%d", b.N)
		store.CheckAndRecord(ctx, key, now)
	}
}

// BenchmarkInMemoryStore_PerAlgorithm_Details provides detailed benchmarks
// for each algorithm under different rates and window sizes.
func BenchmarkInMemoryStore_PerAlgorithm_Details(b *testing.B) {
	testCases := []struct {
		name   string
		alg    config.RateLimitAlgorithm
		rate   int
		window time.Duration
	}{
		{"TokenBucket_HighRate", config.TokenBucket, 10000, time.Second},
		{"TokenBucket_LowRate", config.TokenBucket, 10, time.Second},
		{"TokenBucket_LongWindow", config.TokenBucket, 1000, time.Hour},
		{"FixedWindow_HighRate", config.FixedWindow, 10000, time.Second},
		{"FixedWindow_LowRate", config.FixedWindow, 10, time.Second},
		{"SlidingWindow_HighRate", config.SlidingWindow, 10000, time.Second},
		{"SlidingWindow_LowRate", config.SlidingWindow, 10, time.Second},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			store := NewInMemoryStore(tc.alg, tc.window, tc.rate, 10000)
			ctx := context.Background()
			now := time.Now()
			key := "test-key"

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				store.CheckAndRecord(ctx, key, now)
			}
		})
	}
}

// BenchmarkInMemoryStore_AllAlgorithms_Concurrent compares all three algorithms
// under the same concurrent workload.
func BenchmarkInMemoryStore_AllAlgorithms_Concurrent(b *testing.B) {
	algorithms := []config.RateLimitAlgorithm{
		config.TokenBucket,
		config.FixedWindow,
		config.SlidingWindow,
	}

	for _, alg := range algorithms {
		b.Run(string(alg), func(b *testing.B) {
			store := NewInMemoryStore(alg, time.Minute, 1000, 10000)
			ctx := context.Background()
			now := time.Now()

			// Simulate many different clients (IPs)
			numKeys := 1000
			keys := make([]string, numKeys)
			for i := 0; i < numKeys; i++ {
				keys[i] = fmt.Sprintf("192.168.%d.%d", i/256, i%256)
			}

			b.ReportAllocs()
			b.ResetTimer()

			var counter int
			var mu sync.Mutex

			b.RunParallel(func(pb *testing.PB) {
				mu.Lock()
				idx := counter % numKeys
				counter++
				mu.Unlock()

				for pb.Next() {
					store.CheckAndRecord(ctx, keys[idx], now)
				}
			})
		})
	}
}

// Benchmark evictOldest function directly to measure eviction performance
func BenchmarkEvictOldest(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			// Create a map with entries
			entries := make(map[string]*bucketEntry, size)
			baseTime := time.Now()

			for i := 0; i < size; i++ {
				key := fmt.Sprintf("key-%d", i)
				entries[key] = &bucketEntry{
					tokens:     10,
					capacity:   10,
					lastAccess: baseTime.Add(time.Duration(i) * time.Millisecond),
				}
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				// Make a copy since evictOldest modifies the map
				entriesCopy := make(map[string]*bucketEntry, len(entries))
				for k, v := range entries {
					entriesCopy[k] = v
				}
				evictOldest(entriesCopy)
			}
		})
	}
}
