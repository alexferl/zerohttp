package config

import (
	"testing"
	"time"
)

func TestDefaultReverseProxyConfig(t *testing.T) {
	cfg := DefaultReverseProxyConfig

	if cfg.LoadBalancer != RoundRobin {
		t.Errorf("expected LoadBalancer to be RoundRobin, got %s", cfg.LoadBalancer)
	}
	if cfg.HealthCheckPath != "/" {
		t.Errorf("expected HealthCheckPath to be /, got %s", cfg.HealthCheckPath)
	}
	if cfg.StripPrefix != "" {
		t.Errorf("expected StripPrefix to be empty, got %s", cfg.StripPrefix)
	}
	if cfg.AddPrefix != "" {
		t.Errorf("expected AddPrefix to be empty, got %s", cfg.AddPrefix)
	}
	if len(cfg.Rewrites) != 0 {
		t.Errorf("expected Rewrites to be empty, got %d items", len(cfg.Rewrites))
	}
	if len(cfg.SetHeaders) != 0 {
		t.Errorf("expected SetHeaders to be empty, got %d items", len(cfg.SetHeaders))
	}
	if len(cfg.RemoveHeaders) != 0 {
		t.Errorf("expected RemoveHeaders to be empty, got %d items", len(cfg.RemoveHeaders))
	}
	if !cfg.ForwardHeaders {
		t.Error("expected ForwardHeaders to be true")
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected ExemptPaths to be empty, got %d items", len(cfg.ExemptPaths))
	}
}

func TestLoadBalancerAlgorithm(t *testing.T) {
	algorithms := []LoadBalancerAlgorithm{
		RoundRobin,
		Random,
		LeastConnections,
	}

	for _, algo := range algorithms {
		if algo == "" {
			t.Error("algorithm should not be empty")
		}
	}
}

func TestBackend(t *testing.T) {
	b := Backend{
		Target:  "http://localhost:8081",
		Weight:  3,
		Healthy: true,
	}

	if b.Target != "http://localhost:8081" {
		t.Errorf("expected Target to be http://localhost:8081, got %s", b.Target)
	}
	if b.Weight != 3 {
		t.Errorf("expected Weight to be 3, got %d", b.Weight)
	}
	if !b.Healthy {
		t.Error("expected Healthy to be true")
	}
}

func TestRewriteRule(t *testing.T) {
	rule := RewriteRule{
		Pattern:     "/api/v1/*",
		Replacement: "/api/v2/$1",
	}

	if rule.Pattern != "/api/v1/*" {
		t.Errorf("expected Pattern to be /api/v1/*, got %s", rule.Pattern)
	}
	if rule.Replacement != "/api/v2/$1" {
		t.Errorf("expected Replacement to be /api/v2/$1, got %s", rule.Replacement)
	}
}

func TestReverseProxyConfig(t *testing.T) {
	cfg := ReverseProxyConfig{
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
		ForwardHeaders: true,
		ExemptPaths:    []string{"/health", "/metrics"},
	}

	if cfg.Target != "http://localhost:8081" {
		t.Errorf("expected Target to be http://localhost:8081, got %s", cfg.Target)
	}
	if cfg.HealthCheckInterval != 10*time.Second {
		t.Errorf("expected HealthCheckInterval to be 10s, got %v", cfg.HealthCheckInterval)
	}
	if cfg.HealthCheckTimeout != 5*time.Second {
		t.Errorf("expected HealthCheckTimeout to be 5s, got %v", cfg.HealthCheckTimeout)
	}
	if cfg.HealthCheckPath != "/health" {
		t.Errorf("expected HealthCheckPath to be /health, got %s", cfg.HealthCheckPath)
	}
	if cfg.StripPrefix != "/api" {
		t.Errorf("expected StripPrefix to be /api, got %s", cfg.StripPrefix)
	}
	if cfg.AddPrefix != "/v2" {
		t.Errorf("expected AddPrefix to be /v2, got %s", cfg.AddPrefix)
	}
	if len(cfg.Rewrites) != 1 {
		t.Errorf("expected 1 RewriteRule, got %d", len(cfg.Rewrites))
	} else {
		if cfg.Rewrites[0].Pattern != "/old/*" {
			t.Errorf("expected Pattern to be /old/*, got %s", cfg.Rewrites[0].Pattern)
		}
		if cfg.Rewrites[0].Replacement != "/new/$1" {
			t.Errorf("expected Replacement to be /new/$1, got %s", cfg.Rewrites[0].Replacement)
		}
	}
	if cfg.SetHeaders["X-Custom"] != "value" {
		t.Errorf("expected X-Custom header to be value, got %s", cfg.SetHeaders["X-Custom"])
	}
	if len(cfg.RemoveHeaders) != 1 {
		t.Errorf("expected 1 RemoveHeader, got %d", len(cfg.RemoveHeaders))
	} else if cfg.RemoveHeaders[0] != "X-Internal" {
		t.Errorf("expected RemoveHeaders[0] to be X-Internal, got %s", cfg.RemoveHeaders[0])
	}
	if !cfg.ForwardHeaders {
		t.Error("expected ForwardHeaders to be true")
	}
	if len(cfg.ExemptPaths) != 2 {
		t.Errorf("expected 2 ExemptPaths, got %d", len(cfg.ExemptPaths))
	}
}

func TestReverseProxyConfigWithTargets(t *testing.T) {
	cfg := ReverseProxyConfig{
		Targets: []Backend{
			{Target: "http://backend1:8081", Weight: 1, Healthy: true},
			{Target: "http://backend2:8081", Weight: 2, Healthy: true},
			{Target: "http://backend3:8081", Weight: 1, Healthy: false},
		},
		LoadBalancer: LeastConnections,
	}

	if len(cfg.Targets) != 3 {
		t.Errorf("expected 3 Targets, got %d", len(cfg.Targets))
	}
	if cfg.Targets[0].Target != "http://backend1:8081" {
		t.Errorf("expected Targets[0].Target to be http://backend1:8081, got %s", cfg.Targets[0].Target)
	}
	if cfg.Targets[0].Weight != 1 {
		t.Errorf("expected Targets[0].Weight to be 1, got %d", cfg.Targets[0].Weight)
	}
	if !cfg.Targets[0].Healthy {
		t.Error("expected Targets[0].Healthy to be true")
	}
	if cfg.LoadBalancer != LeastConnections {
		t.Errorf("expected LoadBalancer to be LeastConnections, got %s", cfg.LoadBalancer)
	}
}
