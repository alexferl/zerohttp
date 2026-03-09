package metrics

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("expected registry to not be nil")
	}
}

func TestCounter(t *testing.T) {
	reg := NewRegistry().(*registry)

	// Create counter
	c := reg.Counter("test_counter", "label")

	// Increment with different label values
	c.WithLabelValues("value1").Inc()
	c.WithLabelValues("value1").Inc()
	c.WithLabelValues("value2").Inc()

	// Gather and check
	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "test_counter" {
			found = true
			if len(f.Metrics) != 2 {
				t.Errorf("expected 2 metrics, got %d", len(f.Metrics))
			}
		}
	}
	if !found {
		t.Error("expected to find test_counter metric family")
	}
}

func TestCounterAdd(t *testing.T) {
	reg := NewRegistry()
	c := reg.Counter("test_counter_add", "label")

	c.WithLabelValues("a").Add(5)
	c.WithLabelValues("a").Add(3)
	c.WithLabelValues("b").Add(10)

	families := reg.Gather()
	for _, f := range families {
		if f.Name == "test_counter_add" {
			for _, m := range f.Metrics {
				if m.Labels["label"] == "a" && m.Counter != 8 {
					t.Errorf("expected counter a=8, got %d", m.Counter)
				}
				if m.Labels["label"] == "b" && m.Counter != 10 {
					t.Errorf("expected counter b=10, got %d", m.Counter)
				}
			}
		}
	}
}

func TestGauge(t *testing.T) {
	reg := NewRegistry()

	// Create gauge
	g := reg.Gauge("test_gauge", "label")

	// Set and modify
	g.WithLabelValues("a").Set(10)
	g.WithLabelValues("a").Inc()
	g.WithLabelValues("a").Add(5)
	g.WithLabelValues("a").Sub(2)
	g.WithLabelValues("b").Set(20)
	g.WithLabelValues("b").Dec()

	// Gather and check
	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "test_gauge" {
			found = true
			if f.Type != GaugeType {
				t.Errorf("expected gauge type, got %v", f.Type)
			}
			// Check values: a = 10 + 1 + 5 - 2 = 14, b = 20 - 1 = 19
			for _, m := range f.Metrics {
				if m.Labels["label"] == "a" && m.Gauge != 14 {
					t.Errorf("expected gauge a=14, got %f", m.Gauge)
				}
				if m.Labels["label"] == "b" && m.Gauge != 19 {
					t.Errorf("expected gauge b=19, got %f", m.Gauge)
				}
			}
		}
	}
	if !found {
		t.Error("expected to find test_gauge metric family")
	}
}

func TestHistogram(t *testing.T) {
	reg := NewRegistry()

	// Create histogram with custom buckets
	buckets := []float64{0.1, 0.5, 1.0, 5.0}
	h := reg.Histogram("test_histogram", buckets, "method")

	// Observe values
	h.WithLabelValues("GET").Observe(0.05)
	h.WithLabelValues("GET").Observe(0.3)
	h.WithLabelValues("GET").Observe(2.0)
	h.WithLabelValues("POST").Observe(0.7)

	// Gather and check
	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "test_histogram" {
			found = true
			if f.Type != HistogramType {
				t.Errorf("expected histogram type, got %v", f.Type)
			}
			if len(f.Metrics) != 2 {
				t.Errorf("expected 2 metrics (GET and POST), got %d", len(f.Metrics))
			}
		}
	}
	if !found {
		t.Error("expected to find test_histogram metric family")
	}
}

func TestContext(t *testing.T) {
	reg := NewRegistry()

	// Test adding to context
	ctx := context.Background()
	ctx = WithRegistry(ctx, reg)

	// Test retrieving from context
	retrieved := GetRegistry(ctx)
	if retrieved != reg {
		t.Error("expected to retrieve the same registry from context")
	}

	// Test nil context
	emptyCtx := context.Background()
	if GetRegistry(emptyCtx) != nil {
		t.Error("expected nil when no registry in context")
	}
}

func TestCollector(t *testing.T) {
	reg := NewRegistry().(*registry)

	// Create a custom collector
	collector := &testCollector{
		gauge: reg.Gauge("custom_metric", "label"),
		value: 42,
	}

	reg.RegisterCollector(collector)

	// Gather should call Collect
	reg.Gather()

	// Check that the collector was called
	if !collector.collected {
		t.Error("expected collector to be called")
	}
}

type testCollector struct {
	gauge     Gauge
	value     float64
	collected bool
}

func (c *testCollector) Collect() {
	c.collected = true
	c.gauge.WithLabelValues("test").Set(c.value)
}

func TestGatherSorting(t *testing.T) {
	reg := NewRegistry()

	// Create metrics in non-alphabetical order
	reg.Counter("zebra", "x")
	reg.Counter("alpha", "y")
	reg.Gauge("middle", "z")

	families := reg.Gather()

	// Check families are sorted by name
	if len(families) != 3 {
		t.Fatalf("expected 3 families, got %d", len(families))
	}

	if families[0].Name != "alpha" {
		t.Errorf("expected first family to be 'alpha', got %s", families[0].Name)
	}
	if families[1].Name != "middle" {
		t.Errorf("expected second family to be 'middle', got %s", families[1].Name)
	}
	if families[2].Name != "zebra" {
		t.Errorf("expected third family to be 'zebra', got %s", families[2].Name)
	}
}

func TestCounterVecMethods(t *testing.T) {
	reg := NewRegistry().(*registry)

	// Test counterVec methods that delegate to default counter
	cv := reg.Counter("counter_vec_test").(*counterVec)

	// Inc without labels
	cv.Inc()

	// Add without labels
	cv.Add(5)

	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "counter_vec_test" {
			found = true
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Counter != 6 {
				t.Errorf("expected counter value 6, got %d", f.Metrics[0].Counter)
			}
		}
	}
	if !found {
		t.Error("expected to find counter_vec_test metric family")
	}
}

func TestGaugeVecMethods(t *testing.T) {
	reg := NewRegistry().(*registry)

	// Test gaugeVec methods that delegate to default gauge
	gv := reg.Gauge("gauge_vec_test").(*gaugeVec)

	// Inc without labels
	gv.Inc()

	// Dec without labels
	gv.Dec()

	// Set without labels
	gv.Set(42)

	// Add without labels
	gv.Add(8)

	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "gauge_vec_test" {
			found = true
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Gauge != 50 {
				t.Errorf("expected gauge value 50, got %f", f.Metrics[0].Gauge)
			}
		}
	}
	if !found {
		t.Error("expected to find gauge_vec_test metric family")
	}
}

func TestHistogramVecMethods(t *testing.T) {
	reg := NewRegistry().(*registry)

	// Test histogramVec methods that delegate to default histogram
	hv := reg.Histogram("histogram_vec_test", []float64{0.1, 0.5, 1.0}).(*histogramVec)

	// Observe without labels
	hv.Observe(0.3)
	hv.Observe(0.8)

	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "histogram_vec_test" {
			found = true
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Histogram.Count != 2 {
				t.Errorf("expected histogram count 2, got %d", f.Metrics[0].Histogram.Count)
			}
		}
	}
	if !found {
		t.Error("expected to find histogram_vec_test metric family")
	}
}

func TestCounterWithLabelValuesNoLabels(t *testing.T) {
	reg := NewRegistry()

	// Create counter with no labels
	c := reg.Counter("no_label_counter")

	// WithLabelValues with no args should return the same counter
	c1 := c.WithLabelValues()
	c1.Inc()

	c2 := c.WithLabelValues()
	c2.Inc()

	families := reg.Gather()
	for _, f := range families {
		if f.Name == "no_label_counter" {
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Counter != 2 {
				t.Errorf("expected counter value 2, got %d", f.Metrics[0].Counter)
			}
		}
	}
}

func TestGaugeWithLabelValuesNoLabels(t *testing.T) {
	reg := NewRegistry()

	// Create gauge with no labels
	g := reg.Gauge("no_label_gauge")

	// WithLabelValues with no args should return the same gauge
	g1 := g.WithLabelValues()
	g1.Set(10)

	g2 := g.WithLabelValues()
	g2.Add(5)

	families := reg.Gather()
	for _, f := range families {
		if f.Name == "no_label_gauge" {
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Gauge != 15 {
				t.Errorf("expected gauge value 15, got %f", f.Metrics[0].Gauge)
			}
		}
	}
}

func TestHistogramWithLabelValuesNoLabels(t *testing.T) {
	reg := NewRegistry()

	// Create histogram with no labels
	h := reg.Histogram("no_label_histogram", []float64{0.1, 0.5})

	// WithLabelValues with no args should return the same histogram
	h1 := h.WithLabelValues()
	h1.Observe(0.05)

	h2 := h.WithLabelValues()
	h2.Observe(0.3)

	families := reg.Gather()
	for _, f := range families {
		if f.Name == "no_label_histogram" {
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Histogram.Count != 2 {
				t.Errorf("expected histogram count 2, got %d", f.Metrics[0].Histogram.Count)
			}
		}
	}
}

func TestUnregisterCollector(t *testing.T) {
	reg := NewRegistry().(*registry)

	collector1 := &testCollector{
		gauge: reg.Gauge("unregister_test1"),
		value: 1,
	}
	collector2 := &testCollector{
		gauge: reg.Gauge("unregister_test2"),
		value: 2,
	}

	reg.RegisterCollector(collector1)
	reg.RegisterCollector(collector2)

	// Unregister first collector
	reg.UnregisterCollector(collector1)

	// Gather should only call collector2
	reg.Gather()

	if collector1.collected {
		t.Error("expected collector1 to not be called after unregister")
	}
	if !collector2.collected {
		t.Error("expected collector2 to be called")
	}
}

func TestUnregisterCollector_NotFound(t *testing.T) {
	reg := NewRegistry().(*registry)

	collector := &testCollector{
		gauge: reg.Gauge("not_found_test"),
		value: 1,
	}

	// Unregister a collector that was never registered - should not panic
	reg.UnregisterCollector(collector)
}

func BenchmarkCounterInc(b *testing.B) {
	reg := NewRegistry()
	counter := reg.Counter("bench_counter", "label")
	c := counter.WithLabelValues("value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkHistogramObserve(b *testing.B) {
	reg := NewRegistry()
	hist := reg.Histogram("bench_histogram", nil, "label")
	h := hist.WithLabelValues("value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			h.Observe(float64(i%100) / 100.0)
			i++
		}
	})
}

func TestCounterVec_Concurrent(t *testing.T) {
	reg := NewRegistry()
	c := reg.Counter("concurrent_counter", "label")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.WithLabelValues("same").Inc()
		}()
	}
	wg.Wait()

	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "concurrent_counter" {
			found = true
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Counter != 100 {
				t.Errorf("expected counter value 100, got %d", f.Metrics[0].Counter)
			}
		}
	}
	if !found {
		t.Error("expected to find concurrent_counter metric family")
	}
}

func TestGaugeVec_Concurrent(t *testing.T) {
	reg := NewRegistry()
	g := reg.Gauge("concurrent_gauge", "label")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.WithLabelValues("same").Inc()
		}()
	}
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.WithLabelValues("same").Dec()
		}()
	}
	wg.Wait()

	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "concurrent_gauge" {
			found = true
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			// 50 increments - 25 decrements = 25
			if f.Metrics[0].Gauge != 25 {
				t.Errorf("expected gauge value 25, got %f", f.Metrics[0].Gauge)
			}
		}
	}
	if !found {
		t.Error("expected to find concurrent_gauge metric family")
	}
}

func TestHistogramVec_Concurrent(t *testing.T) {
	reg := NewRegistry()
	h := reg.Histogram("concurrent_histogram", []float64{0.1, 0.5, 1.0}, "label")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			h.WithLabelValues("same").Observe(val)
		}(float64(i%100) / 100.0)
	}
	wg.Wait()

	families := reg.Gather()
	var found bool
	for _, f := range families {
		if f.Name == "concurrent_histogram" {
			found = true
			if len(f.Metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(f.Metrics))
			}
			if f.Metrics[0].Histogram.Count != 100 {
				t.Errorf("expected histogram count 100, got %d", f.Metrics[0].Histogram.Count)
			}
		}
	}
	if !found {
		t.Error("expected to find concurrent_histogram metric family")
	}
}

func TestCounter_CardinalityLimit(t *testing.T) {
	reg := NewRegistry(RegistryConfig{MaxCardinality: 3}).(*registry)
	c := reg.Counter("limited_counter", "label")

	// Add 3 unique values (at limit)
	c.WithLabelValues("a").Inc()
	c.WithLabelValues("b").Inc()
	c.WithLabelValues("c").Inc()

	families := reg.Gather()
	var initialCount int
	for _, f := range families {
		if f.Name == "limited_counter" {
			initialCount = len(f.Metrics)
		}
	}
	if initialCount != 3 {
		t.Errorf("expected 3 metrics, got %d", initialCount)
	}

	// Add 4th value - should evict oldest ("a")
	c.WithLabelValues("d").Inc()

	families = reg.Gather()
	var finalCount int
	var hasA, hasD bool
	for _, f := range families {
		if f.Name == "limited_counter" {
			finalCount = len(f.Metrics)
			for _, m := range f.Metrics {
				if m.Labels["label"] == "a" {
					hasA = true
				}
				if m.Labels["label"] == "d" {
					hasD = true
				}
			}
		}
	}
	if finalCount != 3 {
		t.Errorf("expected 3 metrics after eviction, got %d", finalCount)
	}
	if hasA {
		t.Error("expected 'a' to be evicted (oldest)")
	}
	if !hasD {
		t.Error("expected 'd' to exist")
	}
}

func TestGauge_CardinalityLimit(t *testing.T) {
	reg := NewRegistry(RegistryConfig{MaxCardinality: 2}).(*registry)
	g := reg.Gauge("limited_gauge", "label")

	// Add 2 unique values (at limit)
	g.WithLabelValues("x").Set(1)
	g.WithLabelValues("y").Set(2)

	// Add 3rd value - should evict oldest ("x")
	g.WithLabelValues("z").Set(3)

	families := reg.Gather()
	var hasX, hasZ bool
	for _, f := range families {
		if f.Name == "limited_gauge" {
			for _, m := range f.Metrics {
				if m.Labels["label"] == "x" {
					hasX = true
				}
				if m.Labels["label"] == "z" {
					hasZ = true
				}
			}
		}
	}
	if hasX {
		t.Error("expected 'x' to be evicted (oldest)")
	}
	if !hasZ {
		t.Error("expected 'z' to exist")
	}
}

func TestHistogram_CardinalityLimit(t *testing.T) {
	reg := NewRegistry(RegistryConfig{MaxCardinality: 2}).(*registry)
	h := reg.Histogram("limited_histogram", []float64{0.1, 0.5}, "method")

	// Add 2 unique values (at limit)
	h.WithLabelValues("GET").Observe(0.05)
	h.WithLabelValues("POST").Observe(0.3)

	// Add 3rd value - should evict oldest ("GET")
	h.WithLabelValues("PUT").Observe(0.2)

	families := reg.Gather()
	var hasGet, hasPut bool
	for _, f := range families {
		if f.Name == "limited_histogram" {
			for _, m := range f.Metrics {
				if m.Labels["method"] == "GET" {
					hasGet = true
				}
				if m.Labels["method"] == "PUT" {
					hasPut = true
				}
			}
		}
	}
	if hasGet {
		t.Error("expected 'GET' to be evicted (oldest)")
	}
	if !hasPut {
		t.Error("expected 'PUT' to exist")
	}
}

func TestRegistry_DifferentMetricsDifferentLimits(t *testing.T) {
	// Create two registries with different cardinality limits
	reg1 := NewRegistry(RegistryConfig{MaxCardinality: 5}).(*registry)
	reg2 := NewRegistry(RegistryConfig{MaxCardinality: 2}).(*registry)

	c1 := reg1.Counter("test_counter", "label")
	c2 := reg2.Counter("test_counter", "label")

	c1.WithLabelValues("a").Inc()
	c1.WithLabelValues("b").Inc()
	c1.WithLabelValues("c").Inc()

	c2.WithLabelValues("x").Inc()
	c2.WithLabelValues("y").Inc()
	c2.WithLabelValues("z").Inc()

	// Both should respect their respective limits
	families1 := reg1.Gather()
	families2 := reg2.Gather()

	for _, f := range families1 {
		if f.Name == "test_counter" {
			// reg1 has limit 5, should have 3 metrics
			if len(f.Metrics) != 3 {
				t.Errorf("expected 3 metrics for reg1 test_counter, got %d", len(f.Metrics))
			}
		}
	}
	for _, f := range families2 {
		if f.Name == "test_counter" {
			// reg2 has limit 2, should have 2 metrics (evicted oldest)
			if len(f.Metrics) != 2 {
				t.Errorf("expected 2 metrics for reg2 test_counter, got %d", len(f.Metrics))
			}
		}
	}
}

func TestCounter_CardinalityUnlimited(t *testing.T) {
	// maxCardinality = 0 means unlimited
	reg := NewRegistry(RegistryConfig{MaxCardinality: 0}).(*registry)
	c := reg.Counter("unlimited_counter", "label")

	// Add many values
	for i := 0; i < 100; i++ {
		c.WithLabelValues(fmt.Sprintf("value%d", i)).Inc()
	}

	families := reg.Gather()
	for _, f := range families {
		if f.Name == "unlimited_counter" {
			if len(f.Metrics) != 100 {
				t.Errorf("expected 100 metrics with unlimited cardinality, got %d", len(f.Metrics))
			}
		}
	}
}
