package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Handler returns an http.Handler that exposes metrics in Prometheus format.
func Handler(reg Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reg == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintln(w, "metrics not enabled")
			return
		}

		families := reg.Gather()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		for _, family := range families {
			// Write HELP and TYPE
			_, _ = fmt.Fprintf(w, "# HELP %s %s\n", family.Name, family.Help)
			_, _ = fmt.Fprintf(w, "# TYPE %s %s\n", family.Name, family.Type.String())

			// Sort metrics for consistent output
			sort.Slice(family.Metrics, func(i, j int) bool {
				return metricKey(family.Metrics[i].Labels) < metricKey(family.Metrics[j].Labels)
			})

			for _, m := range family.Metrics {
				labels := formatLabels(m.Labels)

				switch family.Type {
				case CounterType:
					if labels != "" {
						_, _ = fmt.Fprintf(w, "%s{%s} %d\n", family.Name, labels, m.Counter)
					} else {
						_, _ = fmt.Fprintf(w, "%s %d\n", family.Name, m.Counter)
					}

				case GaugeType:
					if labels != "" {
						_, _ = fmt.Fprintf(w, "%s{%s} %g\n", family.Name, labels, m.Gauge)
					} else {
						_, _ = fmt.Fprintf(w, "%s %g\n", family.Name, m.Gauge)
					}

				case HistogramType:
					if m.Histogram == nil {
						continue
					}

					// Sort buckets by upper bound
					bounds := make([]float64, 0, len(m.Histogram.Buckets))
					for b := range m.Histogram.Buckets {
						bounds = append(bounds, b)
					}
					sort.Float64s(bounds)

					// Write buckets
					for _, bound := range bounds {
						count := m.Histogram.Buckets[bound]
						boundStr := fmt.Sprintf("%g", bound)
						if labels != "" {
							_, _ = fmt.Fprintf(w, "%s_bucket{%s,le=\"%s\"} %d\n", family.Name, labels, boundStr, count)
						} else {
							_, _ = fmt.Fprintf(w, "%s_bucket{le=\"%s\"} %d\n", family.Name, boundStr, count)
						}
					}

					// Write +Inf bucket (total count)
					if labels != "" {
						_, _ = fmt.Fprintf(w, "%s_bucket{%s,le=\"+Inf\"} %d\n", family.Name, labels, m.Histogram.Count)
					} else {
						_, _ = fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", family.Name, m.Histogram.Count)
					}

					// Write sum and count
					if labels != "" {
						_, _ = fmt.Fprintf(w, "%s_sum{%s} %g\n", family.Name, labels, m.Histogram.Sum)
						_, _ = fmt.Fprintf(w, "%s_count{%s} %d\n", family.Name, labels, m.Histogram.Count)
					} else {
						_, _ = fmt.Fprintf(w, "%s_sum %g\n", family.Name, m.Histogram.Sum)
						_, _ = fmt.Fprintf(w, "%s_count %d\n", family.Name, m.Histogram.Count)
					}
				}
			}

			_, _ = fmt.Fprintln(w)
		}
	})
}

// formatLabels formats label map as Prometheus label string.
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	// Sort label names for consistent output
	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)

	var parts []string
	for _, name := range names {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, name, escapeLabel(labels[name])))
	}
	return strings.Join(parts, ",")
}

// escapeLabel escapes special characters in label values.
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// metricKey creates a sortable key from labels.
func metricKey(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
