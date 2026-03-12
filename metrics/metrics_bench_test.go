package metrics

import (
	"fmt"
	"testing"
)

// BenchmarkCounter_Increment measures counter Inc() overhead.
func BenchmarkCounter_Increment(b *testing.B) {
	b.Run("SimpleInc", func(b *testing.B) {
		reg := NewRegistry()
		counter := reg.Counter("test_counter")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			counter.Inc()
		}
	})

	b.Run("AddValue", func(b *testing.B) {
		reg := NewRegistry()
		counter := reg.Counter("test_counter")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			counter.Add(5)
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		reg := NewRegistry()
		counter := reg.Counter("test_counter")

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Inc()
			}
		})
	})
}

// BenchmarkCounter_WithLabels measures counter operations with label lookups.
func BenchmarkCounter_WithLabels(b *testing.B) {
	testCases := []struct {
		name   string
		labels []string
	}{
		{"NoLabels", nil},
		{"OneLabel", []string{"method"}},
		{"TwoLabels", []string{"method", "status"}},
		{"ThreeLabels", []string{"method", "status", "path"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			reg := NewRegistry()
			counter := reg.Counter("test_counter", tc.labels...)

			labelValues := make([]string, len(tc.labels))
			for i := range tc.labels {
				labelValues[i] = fmt.Sprintf("value%d", i)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				counter.WithLabelValues(labelValues...).Inc()
			}
		})
	}
}

// BenchmarkCounter_LabelCardinality measures performance with many label values.
func BenchmarkCounter_LabelCardinality(b *testing.B) {
	cardinalities := []int{1, 10, 100, 1000}

	for _, card := range cardinalities {
		b.Run(fmt.Sprintf("Cardinality%d", card), func(b *testing.B) {
			reg := NewRegistry(RegistryConfig{MaxCardinality: card})
			counter := reg.Counter("test_counter", "id")

			// Pre-populate with all label values
			for i := range card {
				counter.WithLabelValues(fmt.Sprintf("id%d", i)).Inc()
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				counter.WithLabelValues("id0").Inc()
			}
		})
	}
}

// BenchmarkGauge_Operations measures gauge operation overhead.
func BenchmarkGauge_Operations(b *testing.B) {
	b.Run("Set", func(b *testing.B) {
		reg := NewRegistry()
		gauge := reg.Gauge("test_gauge")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			gauge.Set(42.5)
		}
	})

	b.Run("Inc", func(b *testing.B) {
		reg := NewRegistry()
		gauge := reg.Gauge("test_gauge")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			gauge.Inc()
		}
	})

	b.Run("Dec", func(b *testing.B) {
		reg := NewRegistry()
		gauge := reg.Gauge("test_gauge")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			gauge.Dec()
		}
	})

	b.Run("Add", func(b *testing.B) {
		reg := NewRegistry()
		gauge := reg.Gauge("test_gauge")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			gauge.Add(1.5)
		}
	})

	b.Run("Sub", func(b *testing.B) {
		reg := NewRegistry()
		gauge := reg.Gauge("test_gauge")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			gauge.Sub(1.5)
		}
	})

	b.Run("ConcurrentSet", func(b *testing.B) {
		reg := NewRegistry()
		gauge := reg.Gauge("test_gauge")

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				gauge.Set(42.5)
			}
		})
	})
}

// BenchmarkGauge_WithLabels measures gauge operations with label lookups.
func BenchmarkGauge_WithLabels(b *testing.B) {
	testCases := []struct {
		name   string
		labels []string
	}{
		{"NoLabels", nil},
		{"OneLabel", []string{"region"}},
		{"TwoLabels", []string{"region", "zone"}},
		{"ThreeLabels", []string{"region", "zone", "instance"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			reg := NewRegistry()
			gauge := reg.Gauge("test_gauge", tc.labels...)

			labelValues := make([]string, len(tc.labels))
			for i := range tc.labels {
				labelValues[i] = fmt.Sprintf("value%d", i)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				gauge.WithLabelValues(labelValues...).Set(42.5)
			}
		})
	}
}

// BenchmarkHistogram_Observe measures histogram observation overhead.
func BenchmarkHistogram_Observe(b *testing.B) {
	buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	b.Run("DefaultBuckets", func(b *testing.B) {
		reg := NewRegistry()
		hist := reg.Histogram("test_histogram", buckets)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			hist.Observe(0.5)
		}
	})

	b.Run("FewBuckets", func(b *testing.B) {
		reg := NewRegistry()
		hist := reg.Histogram("test_histogram", []float64{0.1, 1, 10})

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			hist.Observe(0.5)
		}
	})

	b.Run("ManyBuckets", func(b *testing.B) {
		reg := NewRegistry()
		hist := reg.Histogram("test_histogram", []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 50})

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			hist.Observe(0.5)
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		reg := NewRegistry()
		hist := reg.Histogram("test_histogram", buckets)

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				hist.Observe(0.5)
			}
		})
	})
}

// BenchmarkHistogram_WithLabels measures histogram operations with label lookups.
func BenchmarkHistogram_WithLabels(b *testing.B) {
	buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	testCases := []struct {
		name   string
		labels []string
	}{
		{"NoLabels", nil},
		{"OneLabel", []string{"endpoint"}},
		{"TwoLabels", []string{"endpoint", "method"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			reg := NewRegistry()
			hist := reg.Histogram("test_histogram", buckets, tc.labels...)

			labelValues := make([]string, len(tc.labels))
			for i := range tc.labels {
				labelValues[i] = fmt.Sprintf("value%d", i)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				hist.WithLabelValues(labelValues...).Observe(0.5)
			}
		})
	}
}

// BenchmarkRegistry_Operations measures registry creation and lookup.
func BenchmarkRegistry_Operations(b *testing.B) {
	b.Run("CounterCreate", func(b *testing.B) {
		reg := NewRegistry()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			reg.Counter(fmt.Sprintf("counter%d", b.N))
		}
	})

	b.Run("CounterLookup", func(b *testing.B) {
		reg := NewRegistry()
		reg.Counter("test_counter")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			reg.Counter("test_counter")
		}
	})

	b.Run("GaugeCreate", func(b *testing.B) {
		reg := NewRegistry()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			reg.Gauge(fmt.Sprintf("gauge%d", b.N))
		}
	})

	b.Run("GaugeLookup", func(b *testing.B) {
		reg := NewRegistry()
		reg.Gauge("test_gauge")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			reg.Gauge("test_gauge")
		}
	})

	b.Run("HistogramCreate", func(b *testing.B) {
		reg := NewRegistry()
		buckets := []float64{0.1, 1, 10}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			reg.Histogram(fmt.Sprintf("hist%d", b.N), buckets)
		}
	})

	b.Run("HistogramLookup", func(b *testing.B) {
		reg := NewRegistry()
		buckets := []float64{0.1, 1, 10}
		reg.Histogram("test_hist", buckets)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			reg.Histogram("test_hist", buckets)
		}
	})
}

// BenchmarkRegistry_ManyMetrics measures registry with many metrics.
func BenchmarkRegistry_ManyMetrics(b *testing.B) {
	metricCounts := []int{10, 100, 500}

	for _, count := range metricCounts {
		b.Run(fmt.Sprintf("Counters%d", count), func(b *testing.B) {
			reg := NewRegistry()
			for i := range count {
				reg.Counter(fmt.Sprintf("counter%d", i))
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				reg.Counter("counter0").Inc()
			}
		})

		b.Run(fmt.Sprintf("Gauges%d", count), func(b *testing.B) {
			reg := NewRegistry()
			for i := range count {
				reg.Gauge(fmt.Sprintf("gauge%d", i))
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				reg.Gauge("gauge0").Set(42.5)
			}
		})

		b.Run(fmt.Sprintf("Histograms%d", count), func(b *testing.B) {
			reg := NewRegistry()
			buckets := []float64{0.1, 1, 10}
			for i := range count {
				reg.Histogram(fmt.Sprintf("hist%d", i), buckets)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				reg.Histogram("hist0", buckets).Observe(0.5)
			}
		})
	}
}

// BenchmarkRegistry_Gather measures metric gathering/scraping performance.
func BenchmarkRegistry_Gather(b *testing.B) {
	b.Run("EmptyRegistry", func(b *testing.B) {
		reg := NewRegistry()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reg.Gather()
		}
	})

	b.Run("SingleCounter", func(b *testing.B) {
		reg := NewRegistry()
		counter := reg.Counter("test_counter")
		counter.Inc()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reg.Gather()
		}
	})

	b.Run("SingleCounterWithLabels", func(b *testing.B) {
		reg := NewRegistry()
		counter := reg.Counter("test_counter", "method", "status")
		counter.WithLabelValues("GET", "200").Inc()
		counter.WithLabelValues("POST", "201").Inc()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reg.Gather()
		}
	})

	b.Run("ManyMetrics", func(b *testing.B) {
		reg := NewRegistry()
		for i := range 100 {
			c := reg.Counter(fmt.Sprintf("counter%d", i), "label")
			c.WithLabelValues("value").Inc()
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reg.Gather()
		}
	})

	b.Run("ManyLabelValues", func(b *testing.B) {
		reg := NewRegistry(RegistryConfig{MaxCardinality: 1000})
		counter := reg.Counter("test_counter", "id")
		for i := range 1000 {
			counter.WithLabelValues(fmt.Sprintf("id%d", i)).Inc()
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reg.Gather()
		}
	})
}

// BenchmarkNopRegistry measures no-op registry overhead.
func BenchmarkNopRegistry(b *testing.B) {
	b.Run("Counter", func(b *testing.B) {
		reg := SafeRegistry(nil)
		counter := reg.Counter("test")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			counter.Inc()
		}
	})

	b.Run("Gauge", func(b *testing.B) {
		reg := SafeRegistry(nil)
		gauge := reg.Gauge("test")

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			gauge.Set(42.5)
		}
	})

	b.Run("Histogram", func(b *testing.B) {
		reg := SafeRegistry(nil)
		hist := reg.Histogram("test", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			hist.Observe(0.5)
		}
	})
}

// BenchmarkLabelKeyCreation measures label key creation overhead.
func BenchmarkLabelKeyCreation(b *testing.B) {
	testCases := []struct {
		name   string
		labels []string
	}{
		{"Empty", nil},
		{"One", []string{"value1"}},
		{"Two", []string{"value1", "value2"}},
		{"Three", []string{"value1", "value2", "value3"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = makeLabelKey(tc.labels)
			}
		})
	}
}

// BenchmarkLabelKeyParse measures label key parsing overhead.
func BenchmarkLabelKeyParse(b *testing.B) {
	testCases := []struct {
		name       string
		labelNames []string
		key        string
	}{
		{"Empty", nil, ""},
		{"One", []string{"label1"}, "value1"},
		{"Two", []string{"label1", "label2"}, "value1\x00value2"},
		{"Three", []string{"label1", "label2", "label3"}, "value1\x00value2\x00value3"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = parseLabelKey(tc.key, tc.labelNames)
			}
		})
	}
}
