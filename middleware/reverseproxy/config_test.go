package reverseproxy

import (
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultReverseProxyConfig(t *testing.T) {
	cfg := DefaultConfig

	zhtest.AssertEqual(t, RoundRobin, cfg.LoadBalancer)
	zhtest.AssertEqual(t, "/", cfg.HealthCheckPath)
	zhtest.AssertEqual(t, "", cfg.StripPrefix)
	zhtest.AssertEqual(t, "", cfg.AddPrefix)
	zhtest.AssertEqual(t, 0, len(cfg.Rewrites))
	zhtest.AssertEqual(t, 0, len(cfg.SetHeaders))
	zhtest.AssertEqual(t, 0, len(cfg.RemoveHeaders))
	zhtest.AssertTrue(t, *cfg.ForwardHeaders)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
}

func TestLoadBalancerAlgorithm(t *testing.T) {
	algorithms := []LoadBalancerAlgorithm{
		RoundRobin,
		Random,
		LeastConnections,
	}

	for _, algo := range algorithms {
		zhtest.AssertNotEmpty(t, string(algo))
	}
}

func TestBackend(t *testing.T) {
	b := Backend{
		Target:  "http://localhost:8081",
		Weight:  3,
		Healthy: config.Bool(true),
	}

	zhtest.AssertEqual(t, "http://localhost:8081", b.Target)
	zhtest.AssertEqual(t, 3, b.Weight)
	zhtest.AssertTrue(t, *b.Healthy)
}

func TestRewriteRule(t *testing.T) {
	rule := RewriteRule{
		Pattern:     "/api/v1/*",
		Replacement: "/api/v2/$1",
	}

	zhtest.AssertEqual(t, "/api/v1/*", rule.Pattern)
	zhtest.AssertEqual(t, "/api/v2/$1", rule.Replacement)
}

func TestReverseProxyConfig(t *testing.T) {
	cfg := Config{
		Target:              "http://localhost:8081",
		LoadBalancer:        RoundRobin,
		HealthCheckInterval: 10 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		HealthCheckPath:     "/health",
		StripPrefix:         "/api",
		AddPrefix:           "/v2",
		Rewrites: []RewriteRule{
			{Pattern: "/old/*", Replacement: "/new/$1"},
		},
		SetHeaders: map[string]string{
			"X-Custom": "value",
		},
		RemoveHeaders:  []string{"X-Internal"},
		ForwardHeaders: config.Bool(true),
		ExcludedPaths:  []string{"/health", "/metrics"},
		IncludedPaths:  []string{"/api/public"},
	}

	zhtest.AssertEqual(t, "http://localhost:8081", cfg.Target)
	zhtest.AssertEqual(t, 10*time.Second, cfg.HealthCheckInterval)
	zhtest.AssertEqual(t, 5*time.Second, cfg.HealthCheckTimeout)
	zhtest.AssertEqual(t, "/health", cfg.HealthCheckPath)
	zhtest.AssertEqual(t, "/api", cfg.StripPrefix)
	zhtest.AssertEqual(t, "/v2", cfg.AddPrefix)
	zhtest.AssertEqual(t, 1, len(cfg.Rewrites))
	zhtest.AssertEqual(t, "/old/*", cfg.Rewrites[0].Pattern)
	zhtest.AssertEqual(t, "/new/$1", cfg.Rewrites[0].Replacement)
	zhtest.AssertEqual(t, "value", cfg.SetHeaders["X-Custom"])
	zhtest.AssertEqual(t, 1, len(cfg.RemoveHeaders))
	zhtest.AssertEqual(t, "X-Internal", cfg.RemoveHeaders[0])
	zhtest.AssertTrue(t, *cfg.ForwardHeaders)
	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
}

func TestReverseProxyConfig_IncludedPaths(t *testing.T) {
	t.Run("custom included paths", func(t *testing.T) {
		cfg := Config{
			Target:        "http://localhost:8081",
			IncludedPaths: []string{"/api/public", "/health"},
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, "/api/public", cfg.IncludedPaths[0])
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})
}

func TestReverseProxyConfigWithTargets(t *testing.T) {
	cfg := Config{
		Targets: []Backend{
			{Target: "http://backend1:8081", Weight: 1, Healthy: config.Bool(true)},
			{Target: "http://backend2:8081", Weight: 2, Healthy: config.Bool(true)},
			{Target: "http://backend3:8081", Weight: 1, Healthy: config.Bool(false)},
		},
		LoadBalancer: LeastConnections,
	}

	zhtest.AssertEqual(t, 3, len(cfg.Targets))
	zhtest.AssertEqual(t, "http://backend1:8081", cfg.Targets[0].Target)
	zhtest.AssertEqual(t, 1, cfg.Targets[0].Weight)
	zhtest.AssertTrue(t, *cfg.Targets[0].Healthy)
	zhtest.AssertEqual(t, LeastConnections, cfg.LoadBalancer)
}
