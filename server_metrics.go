// Package zerohttp provides metrics server support. See [Server.Metrics] and [Server.MetricsAddr].
package zerohttp

import (
	"net"

	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
)

// Metrics returns the metrics registry for collecting custom metrics.
// Returns nil if metrics are not enabled.
//
// Use this to create custom metrics in your handlers or middleware:
//
//	requests := app.Metrics().Counter("my_requests_total", "status")
//	requests.WithLabelValues("200").Inc()
func (s *Server) Metrics() metrics.Registry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metricsRegistry
}

// MetricsAddr returns the network address that the metrics server is listening on.
// If a listener is configured, it returns the listener's actual address.
// If no listener is configured but a metrics server is configured, it returns the server's configured address.
// If no metrics server is configured, it returns an empty string.
//
// This method is thread-safe and can be called concurrently.
func (s *Server) MetricsAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.metricsListener != nil {
		return s.metricsListener.Addr().String()
	}

	if s.metricsServer != nil {
		return s.metricsServer.Addr
	}

	return ""
}

// startMetricsServer starts the dedicated metrics server.
// It creates a listener if one doesn't exist and serves metrics.
func (s *Server) startMetricsServer() error {
	s.mu.Lock()

	var err error
	if s.metricsListener == nil {
		s.logger.Debug("Creating metrics listener", log.F("addr", s.metricsServer.Addr))
		s.metricsListener, err = net.Listen("tcp", s.metricsServer.Addr)
		if err != nil {
			s.mu.Unlock()
			return err
		}
	}

	s.mu.Unlock()

	return s.metricsServer.Serve(s.metricsListener)
}
