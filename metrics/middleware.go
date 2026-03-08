package metrics

import (
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// responseWriter wraps http.ResponseWriter to capture status and size.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (w *responseWriter) WriteHeader(code int) {
	if w.statusCode == 0 {
		w.statusCode = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	return n, err
}

// labelSet holds pre-allocated label slices to avoid allocations per request.
type labelSet struct {
	inFlight  []string
	request   []string
	requestSz []string
}

// Middleware collects HTTP request metrics.
type Middleware struct {
	reg          Registry
	Requests     Counter
	RequestDur   Histogram
	RequestSize  Histogram
	ResponseSize Histogram
	InFlight     Gauge

	DurationBuckets []float64
	SizeBuckets     []float64
	ExcludePaths    map[string]struct{}
	PathLabelFunc   func(string) string
	CustomLabels    func(r *http.Request) map[string]string
	customLabelKeys []string
	mu              sync.Mutex
	initialized     bool
}

// NewMiddleware creates a new metrics middleware.
func NewMiddleware(reg Registry, cfg config.MetricsConfig) func(http.Handler) http.Handler {
	if !cfg.Enabled || reg == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	excludePaths := make(map[string]struct{})
	for _, p := range cfg.ExcludePaths {
		excludePaths[p] = struct{}{}
	}

	mm := &Middleware{
		reg:             reg,
		DurationBuckets: cfg.DurationBuckets,
		SizeBuckets:     cfg.SizeBuckets,
		ExcludePaths:    excludePaths,
		PathLabelFunc:   cfg.PathLabelFunc,
		CustomLabels:    cfg.CustomLabels,
	}

	// Only initialize metrics immediately if CustomLabels is not set
	// If CustomLabels is set, we'll initialize on first request when we know the label keys
	if cfg.CustomLabels == nil {
		mm.initMetrics(reg, nil)
		mm.initialized = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip excluded paths
			if _, excluded := mm.ExcludePaths[r.URL.Path]; excluded {
				next.ServeHTTP(w, r)
				return
			}

			path := mm.PathLabelFunc(r.URL.Path)
			method := r.Method

			// Get custom labels if configured
			var customLabelValues []string
			if mm.CustomLabels != nil {
				customLabels := mm.CustomLabels(r)
				mm.ensureInitialized(reg, customLabels)
				customLabelValues = mm.getCustomLabelValues(customLabels)
			}

			// Build label sets (pre-allocated capacity to avoid reallocations)
			labels := mm.buildLabels(method, path, customLabelValues)

			// Track in-flight
			mm.InFlight.WithLabelValues(labels.inFlight...).Inc()
			defer mm.InFlight.WithLabelValues(labels.inFlight...).Dec()

			// Wrap writer to capture status and size
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     0,
			}

			start := time.Now()
			next.ServeHTTP(wrapped, r)
			duration := time.Since(start).Seconds()

			status := strconv.Itoa(wrapped.statusCode)
			if wrapped.statusCode == 0 {
				status = "200"
			}

			// Update status in request labels
			labels.request[1] = status

			// Record metrics
			mm.Requests.WithLabelValues(labels.request...).Inc()
			mm.RequestDur.WithLabelValues(labels.request...).Observe(duration)

			if r.ContentLength > 0 {
				mm.RequestSize.WithLabelValues(labels.requestSz...).Observe(float64(r.ContentLength))
			}
			if wrapped.size > 0 {
				mm.ResponseSize.WithLabelValues(labels.request...).Observe(float64(wrapped.size))
			}
		})
	}
}

// buildLabels creates label slices with pre-allocated capacity.
func (mm *Middleware) buildLabels(method, path string, customValues []string) labelSet {
	numCustom := len(customValues)

	// inFlight: method, path + custom
	inFlight := make([]string, 0, 2+numCustom)
	inFlight = append(inFlight, method, path)
	inFlight = append(inFlight, customValues...)

	// request: method, status, path + custom
	// status is placeholder, will be set after request
	request := make([]string, 0, 3+numCustom)
	request = append(request, method, "", path)
	request = append(request, customValues...)

	// requestSz: method, path + custom
	requestSz := make([]string, 0, 2+numCustom)
	requestSz = append(requestSz, method, path)
	requestSz = append(requestSz, customValues...)

	return labelSet{
		inFlight:  inFlight,
		request:   request,
		requestSz: requestSz,
	}
}

// initMetrics initializes metrics with the given custom label keys.
func (mm *Middleware) initMetrics(reg Registry, customLabelKeys []string) {
	mm.customLabelKeys = customLabelKeys

	// Standard labels
	requestLabels := []string{"method", "status", "path"}
	sizeLabels := []string{"method", "path"}
	inFlightLabels := []string{"method", "path"}

	// Add custom labels if provided
	if len(customLabelKeys) > 0 {
		requestLabels = append(requestLabels, customLabelKeys...)
		sizeLabels = append(sizeLabels, customLabelKeys...)
		inFlightLabels = append(inFlightLabels, customLabelKeys...)
	}

	mm.Requests = reg.Counter("http_requests_total", requestLabels...)
	mm.RequestDur = reg.Histogram("http_request_duration_seconds", mm.DurationBuckets, requestLabels...)
	mm.RequestSize = reg.Histogram("http_request_size_bytes", mm.SizeBuckets, sizeLabels...)
	mm.ResponseSize = reg.Histogram("http_response_size_bytes", mm.SizeBuckets, requestLabels...)
	mm.InFlight = reg.Gauge("http_requests_in_flight", inFlightLabels...)
}

// ensureInitialized ensures metrics are initialized with custom label keys.
func (mm *Middleware) ensureInitialized(reg Registry, customLabels map[string]string) {
	if mm.initialized {
		return
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.initialized {
		return
	}

	// Extract and sort custom label keys for consistent ordering
	keys := make([]string, 0, len(customLabels))
	for k := range customLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mm.initMetrics(reg, keys)
	mm.initialized = true
}

// getCustomLabelValues returns custom label values in the same order as the keys.
func (mm *Middleware) getCustomLabelValues(customLabels map[string]string) []string {
	values := make([]string, len(mm.customLabelKeys))
	for i, key := range mm.customLabelKeys {
		if v, ok := customLabels[key]; ok {
			values[i] = v
		} else {
			values[i] = ""
		}
	}
	return values
}
