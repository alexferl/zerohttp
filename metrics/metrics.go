// Package metrics provides Prometheus-compatible metrics collection.
package metrics

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	zconfig "github.com/alexferl/zerohttp/internal/config"
)

// Registry provides methods for creating and registering metrics.
type Registry interface {
	// Counter creates a new counter metric with the given name and labels.
	Counter(name string, labels ...string) Counter

	// Gauge creates a new gauge metric with the given name and labels.
	Gauge(name string, labels ...string) Gauge

	// Histogram creates a new histogram metric with the given name, buckets, and labels.
	Histogram(name string, buckets []float64, labels ...string) Histogram

	// RegisterCollector allows custom collectors to add metrics on scrape.
	RegisterCollector(collector Collector)

	// UnregisterCollector removes a collector from the registry.
	UnregisterCollector(collector Collector)

	// Gather returns all metric families for exposition.
	Gather() []MetricFamily
}

// Counter is a monotonically increasing metric.
type Counter interface {
	Inc()
	Add(val float64)
	WithLabelValues(values ...string) Counter
}

// Gauge can go up and down.
// Note: Gauge values are stored with fixed-point precision (3 decimal places).
// Values are rounded to the nearest 0.001. For example, 1.2345 becomes 1.235.
type Gauge interface {
	Inc()
	Dec()
	Set(val float64)
	Add(val float64)
	Sub(val float64)
	WithLabelValues(values ...string) Gauge
}

// Histogram samples observations (e.g., request durations).
type Histogram interface {
	Observe(val float64)
	WithLabelValues(values ...string) Histogram
}

// Collector is for computed metrics (called on each /metrics scrape).
type Collector interface {
	Collect()
}

// MetricFamily represents a group of metrics with the same name.
type MetricFamily struct {
	Name    string
	Help    string
	Type    MetricType
	Metrics []Metric
}

// MetricType represents the type of a metric.
type MetricType int

const (
	CounterType MetricType = iota
	GaugeType
	HistogramType
)

// Metric represents a single metric with labels and value.
type Metric struct {
	Labels    map[string]string
	Counter   uint64
	Gauge     float64
	Histogram *HistogramValue
}

// HistogramValue contains histogram data.
type HistogramValue struct {
	Buckets map[float64]uint64
	Sum     float64
	Count   uint64
}

// String returns the metric type as a string.
func (t MetricType) String() string {
	switch t {
	case CounterType:
		return "counter"
	case GaugeType:
		return "gauge"
	case HistogramType:
		return "histogram"
	default:
		return "unknown"
	}
}

// context key type (private to avoid collisions)
type contextKey struct{}

// WithRegistry adds the registry to the context.
func WithRegistry(ctx context.Context, reg Registry) context.Context {
	return context.WithValue(ctx, contextKey{}, reg)
}

// GetRegistry retrieves the registry from context.
// Returns nil if metrics are not enabled.
func GetRegistry(ctx context.Context) Registry {
	if reg, ok := ctx.Value(contextKey{}).(Registry); ok {
		return reg
	}
	return nil
}

// SafeRegistry returns a registry that safely handles nil.
// If reg is nil, returns a no-op registry that does nothing.
// This allows code to call registry methods without nil checks.
func SafeRegistry(reg Registry) Registry {
	if reg == nil {
		return nopRegistry{}
	}
	return reg
}

// Ensure nopRegistry implements Registry
var _ Registry = (*nopRegistry)(nil)

// nopRegistry is a no-op registry that does nothing.
type nopRegistry struct{}

func (nopRegistry) Counter(string, ...string) Counter                { return nopCounter{} }
func (nopRegistry) Gauge(string, ...string) Gauge                    { return nopGauge{} }
func (nopRegistry) Histogram(string, []float64, ...string) Histogram { return nopHistogram{} }
func (nopRegistry) RegisterCollector(Collector)                      {}
func (nopRegistry) UnregisterCollector(Collector)                    {}
func (nopRegistry) Gather() []MetricFamily                           { return nil }

// Ensure nopCounter implements Counter
var _ Counter = (*nopCounter)(nil)

// nopCounter is a no-op counter.
type nopCounter struct{}

func (nopCounter) Inc()                              {}
func (nopCounter) Add(float64)                       {}
func (nopCounter) WithLabelValues(...string) Counter { return nopCounter{} }

// Ensure nopGauge implements Gauge
var _ Gauge = (*nopGauge)(nil)

// nopGauge is a no-op gauge.
type nopGauge struct{}

func (nopGauge) Inc()                            {}
func (nopGauge) Dec()                            {}
func (nopGauge) Set(float64)                     {}
func (nopGauge) Add(float64)                     {}
func (nopGauge) Sub(float64)                     {}
func (nopGauge) WithLabelValues(...string) Gauge { return nopGauge{} }

// Ensure nopHistogram implements Histogram
var _ Histogram = (*nopHistogram)(nil)

// nopHistogram is a no-op histogram.
type nopHistogram struct{}

func (nopHistogram) Observe(float64)                     {}
func (nopHistogram) WithLabelValues(...string) Histogram { return nopHistogram{} }

// NewRegistry creates a new metrics registry.
// Default max cardinality is 1000 unique label combinations per metric.
// Pass RegistryConfig{MaxCardinality: N} to customize.
func NewRegistry(cfg ...RegistryConfig) Registry {
	c := DefaultRegistryConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}
	return &registry{
		counters:       make(map[string]*counterVec),
		gauges:         make(map[string]*gaugeVec),
		histograms:     make(map[string]*histogramVec),
		collectors:     make([]Collector, 0),
		maxCardinality: c.MaxCardinality,
	}
}

// RegistryConfig holds optional configuration for the registry.
type RegistryConfig struct {
	// MaxCardinality limits the number of unique label combinations per metric.
	// When exceeded, oldest entries are evicted (FIFO).
	// Default: 1000. Set to 0 for unlimited (not recommended with user-controlled labels).
	MaxCardinality int
}

// DefaultRegistryConfig is the default registry configuration.
var DefaultRegistryConfig = RegistryConfig{
	MaxCardinality: 1000,
}

// Ensure registry implements Registry
var _ Registry = (*registry)(nil)

// registry is the internal implementation of Registry.
type registry struct {
	mu             sync.RWMutex
	counters       map[string]*counterVec
	gauges         map[string]*gaugeVec
	histograms     map[string]*histogramVec
	collectors     []Collector
	maxCardinality int // 0 = unlimited, default 1000
}

// Ensure counterVec implements Counter
var _ Counter = (*counterVec)(nil)

// counterVec holds counters with different label values.
type counterVec struct {
	name           string
	labels         []string
	mu             sync.RWMutex
	values         map[string]*counter
	insertOrder    []string // tracks insertion order for LRU eviction
	maxCardinality int      // 0 = unlimited
}

// Ensure counter implements Counter
var _ Counter = (*counter)(nil)

// counter is a single counter instance.
type counter struct {
	value uint64
}

func (c *counter) Inc() {
	atomic.AddUint64(&c.value, 1)
}

func (c *counter) Add(val float64) {
	if val > 0 {
		atomic.AddUint64(&c.value, uint64(val))
	}
}

func (c *counter) WithLabelValues(values ...string) Counter {
	return c
}

func (c *counter) Value() uint64 {
	return atomic.LoadUint64(&c.value)
}

func (cv *counterVec) Inc() {
	cv.WithLabelValues().Inc()
}

func (cv *counterVec) Add(val float64) {
	cv.WithLabelValues().Add(val)
}

func (cv *counterVec) WithLabelValues(values ...string) Counter {
	key := makeLabelKey(values)

	cv.mu.RLock()
	if c, exists := cv.values[key]; exists {
		cv.mu.RUnlock()
		return c
	}
	cv.mu.RUnlock()

	cv.mu.Lock()
	defer cv.mu.Unlock()

	if c, exists := cv.values[key]; exists {
		return c
	}

	// Check cardinality limit
	if cv.maxCardinality > 0 && len(cv.values) >= cv.maxCardinality {
		// Evict oldest entry (FIFO)
		oldestKey := cv.insertOrder[0]
		delete(cv.values, oldestKey)
		cv.insertOrder = cv.insertOrder[1:]
	}

	c := &counter{}
	cv.values[key] = c
	cv.insertOrder = append(cv.insertOrder, key)
	return c
}

// Ensure gaugeVec implements Gauge
var _ Gauge = (*gaugeVec)(nil)

// gaugeVec holds gauges with different label values.
type gaugeVec struct {
	name           string
	labels         []string
	mu             sync.RWMutex
	values         map[string]*gauge
	insertOrder    []string // tracks insertion order for LRU eviction
	maxCardinality int      // 0 = unlimited
}

// Ensure gauge implements Gauge
var _ Gauge = (*gauge)(nil)

// gauge is a single gauge instance.
// Values are stored with fixed-point precision (3 decimal places) using int64.
// The value is multiplied by 1000 for storage, allowing precision to 0.001.
type gauge struct {
	value int64 // stored as fixed-point (val * 1000) for atomic operations
}

func (g *gauge) Inc() {
	g.Add(1)
}

func (g *gauge) Dec() {
	g.Add(-1)
}

func (g *gauge) Set(val float64) {
	atomic.StoreInt64(&g.value, int64(val*1000))
}

func (g *gauge) Add(val float64) {
	atomic.AddInt64(&g.value, int64(val*1000))
}

func (g *gauge) Sub(val float64) {
	atomic.AddInt64(&g.value, -int64(val*1000))
}

func (g *gauge) WithLabelValues(values ...string) Gauge {
	return g
}

func (g *gauge) Value() float64 {
	return float64(atomic.LoadInt64(&g.value)) / 1000
}

func (gv *gaugeVec) Inc() {
	gv.WithLabelValues().Inc()
}

func (gv *gaugeVec) Dec() {
	gv.WithLabelValues().Dec()
}

func (gv *gaugeVec) Set(val float64) {
	gv.WithLabelValues().Set(val)
}

func (gv *gaugeVec) Add(val float64) {
	gv.WithLabelValues().Add(val)
}

func (gv *gaugeVec) Sub(val float64) {
	gv.WithLabelValues().Sub(val)
}

func (gv *gaugeVec) WithLabelValues(values ...string) Gauge {
	key := makeLabelKey(values)

	gv.mu.RLock()
	if g, exists := gv.values[key]; exists {
		gv.mu.RUnlock()
		return g
	}
	gv.mu.RUnlock()

	gv.mu.Lock()
	defer gv.mu.Unlock()

	if g, exists := gv.values[key]; exists {
		return g
	}

	// Check cardinality limit
	if gv.maxCardinality > 0 && len(gv.values) >= gv.maxCardinality {
		// Evict oldest entry (FIFO)
		oldestKey := gv.insertOrder[0]
		delete(gv.values, oldestKey)
		gv.insertOrder = gv.insertOrder[1:]
	}

	g := &gauge{}
	gv.values[key] = g
	gv.insertOrder = append(gv.insertOrder, key)
	return g
}

// Ensure histogramVec implements Histogram
var _ Histogram = (*histogramVec)(nil)

// histogramVec holds histograms with different label values.
type histogramVec struct {
	name           string
	labels         []string
	buckets        []float64
	mu             sync.RWMutex
	values         map[string]*histogram
	insertOrder    []string // tracks insertion order for LRU eviction
	maxCardinality int      // 0 = unlimited
}

// Ensure histogram implements Histogram
var _ Histogram = (*histogram)(nil)

// histogram is a single histogram instance.
type histogram struct {
	buckets []float64
	counts  []uint64
	sum     uint64
	count   uint64
}

// Observe records a value in the histogram.
// Values are stored with fixed-point precision (3 decimal places) for atomic operations.
// This provides sufficient precision for typical latency measurements (milliseconds).
func (h *histogram) Observe(val float64) {
	atomic.AddUint64(&h.count, 1)
	atomic.AddUint64(&h.sum, uint64(val*1000))

	for i, bucket := range h.buckets {
		if val <= bucket {
			atomic.AddUint64(&h.counts[i], 1)
		}
	}
}

func (h *histogram) WithLabelValues(values ...string) Histogram {
	return h
}

func (hv *histogramVec) Observe(val float64) {
	hv.WithLabelValues().Observe(val)
}

func (hv *histogramVec) WithLabelValues(values ...string) Histogram {
	key := makeLabelKey(values)

	hv.mu.RLock()
	if h, exists := hv.values[key]; exists {
		hv.mu.RUnlock()
		return h
	}
	hv.mu.RUnlock()

	hv.mu.Lock()
	defer hv.mu.Unlock()

	if h, exists := hv.values[key]; exists {
		return h
	}

	// Check cardinality limit
	if hv.maxCardinality > 0 && len(hv.values) >= hv.maxCardinality {
		// Evict oldest entry (FIFO)
		oldestKey := hv.insertOrder[0]
		delete(hv.values, oldestKey)
		hv.insertOrder = hv.insertOrder[1:]
	}

	h := &histogram{
		buckets: hv.buckets,
		counts:  make([]uint64, len(hv.buckets)),
	}
	hv.values[key] = h
	hv.insertOrder = append(hv.insertOrder, key)
	return h
}

func (r *registry) Counter(name string, labels ...string) Counter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if vec, exists := r.counters[name]; exists {
		return vec
	}

	vec := &counterVec{
		name:           name,
		labels:         labels,
		values:         make(map[string]*counter),
		insertOrder:    make([]string, 0),
		maxCardinality: r.maxCardinality,
	}
	r.counters[name] = vec
	return vec
}

func (r *registry) Gauge(name string, labels ...string) Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()

	if vec, exists := r.gauges[name]; exists {
		return vec
	}

	vec := &gaugeVec{
		name:           name,
		labels:         labels,
		values:         make(map[string]*gauge),
		insertOrder:    make([]string, 0),
		maxCardinality: r.maxCardinality,
	}
	r.gauges[name] = vec
	return vec
}

func (r *registry) Histogram(name string, buckets []float64, labels ...string) Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()

	if buckets == nil {
		buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	}

	if vec, exists := r.histograms[name]; exists {
		return vec
	}

	vec := &histogramVec{
		name:           name,
		labels:         labels,
		buckets:        buckets,
		values:         make(map[string]*histogram),
		insertOrder:    make([]string, 0),
		maxCardinality: r.maxCardinality,
	}
	r.histograms[name] = vec
	return vec
}

func (r *registry) RegisterCollector(collector Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors = append(r.collectors, collector)
}

func (r *registry) UnregisterCollector(collector Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, c := range r.collectors {
		if c == collector {
			r.collectors = append(r.collectors[:i], r.collectors[i+1:]...)
			return
		}
	}
}

func (r *registry) Gather() []MetricFamily {
	// Run collectors first
	r.mu.RLock()
	collectors := make([]Collector, len(r.collectors))
	copy(collectors, r.collectors)
	r.mu.RUnlock()

	for _, c := range collectors {
		c.Collect()
	}

	var families []MetricFamily

	// Gather counters
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, vec := range r.counters {
		vec.mu.RLock()
		metrics := make([]Metric, 0, len(vec.values))
		for labelKey, c := range vec.values {
			labels := parseLabelKey(labelKey, vec.labels)
			metrics = append(metrics, Metric{
				Labels:  labels,
				Counter: c.Value(),
			})
		}
		vec.mu.RUnlock()

		families = append(families, MetricFamily{
			Name:    name,
			Help:    fmt.Sprintf("Total %s", name),
			Type:    CounterType,
			Metrics: metrics,
		})
	}

	// Gather gauges
	for name, vec := range r.gauges {
		vec.mu.RLock()
		metrics := make([]Metric, 0, len(vec.values))
		for labelKey, g := range vec.values {
			labels := parseLabelKey(labelKey, vec.labels)
			metrics = append(metrics, Metric{
				Labels: labels,
				Gauge:  g.Value(),
			})
		}
		vec.mu.RUnlock()

		families = append(families, MetricFamily{
			Name:    name,
			Help:    fmt.Sprintf("Current %s", name),
			Type:    GaugeType,
			Metrics: metrics,
		})
	}

	// Gather histograms
	for name, vec := range r.histograms {
		vec.mu.RLock()
		metrics := make([]Metric, 0, len(vec.values))
		for labelKey, h := range vec.values {
			labels := parseLabelKey(labelKey, vec.labels)
			buckets := make(map[float64]uint64)
			for i, b := range vec.buckets {
				buckets[b] = atomic.LoadUint64(&h.counts[i])
			}
			metrics = append(metrics, Metric{
				Labels: labels,
				Histogram: &HistogramValue{
					Buckets: buckets,
					Sum:     float64(atomic.LoadUint64(&h.sum)) / 1000,
					Count:   atomic.LoadUint64(&h.count),
				},
			})
		}
		vec.mu.RUnlock()

		families = append(families, MetricFamily{
			Name:    name,
			Help:    fmt.Sprintf("Distribution of %s", name),
			Type:    HistogramType,
			Metrics: metrics,
		})
	}

	// Sort families by name for consistent output
	sort.Slice(families, func(i, j int) bool {
		return families[i].Name < families[j].Name
	})

	return families
}

func makeLabelKey(values []string) string {
	return strings.Join(values, "\x00")
}

func parseLabelKey(key string, labelNames []string) map[string]string {
	labels := make(map[string]string)
	values := strings.Split(key, "\x00")
	for i, name := range labelNames {
		if i < len(values) {
			labels[name] = values[i]
		}
	}
	return labels
}
