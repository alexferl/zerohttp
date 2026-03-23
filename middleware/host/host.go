package host

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/internal/problem"
)

// New creates a host validation middleware with the provided configuration that validates
// the Host header to prevent DNS rebinding attacks and ensures requests target valid domains.
//
// Example:
//
//	host.New(host.Config{
//	    AllowedHosts:    []string{"api.example.com", "example.com"},
//	    AllowSubdomains: true,
//	})
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	if c.StrictPort {
		if c.Port == 0 {
			panic("zerohttp: HostValidation StrictPort requires Port to be set")
		}
		if c.Port == 80 || c.Port == 443 {
			panic("zerohttp: HostValidation StrictPort has no effect on standard ports 80/443")
		}
	}

	allowedHosts := make([]string, len(c.AllowedHosts))
	for i, h := range c.AllowedHosts {
		if hostWithoutPort, _, err := net.SplitHostPort(h); err == nil {
			allowedHosts[i] = hostWithoutPort
		} else {
			if strings.HasPrefix(h, "[") && strings.HasSuffix(h, "]") {
				allowedHosts[i] = h[1 : len(h)-1]
			} else {
				allowedHosts[i] = h
			}
		}
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "HostValidation")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(allowedHosts) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			if r.Host == "" {
				detail := problem.NewDetail(c.StatusCode, c.Message)
				_ = detail.RenderAuto(w, r)
				return
			}

			hostWithoutPort, port, err := net.SplitHostPort(r.Host)
			if err != nil {
				hostWithoutPort = r.Host
				port = ""

				if strings.HasPrefix(hostWithoutPort, "[") && strings.HasSuffix(hostWithoutPort, "]") {
					hostWithoutPort = hostWithoutPort[1 : len(hostWithoutPort)-1]
				}
			}

			if c.StrictPort {
				expectedPort := strconv.Itoa(c.Port)
				if port != expectedPort {
					detail := problem.NewDetail(c.StatusCode, c.Message)
					_ = detail.RenderAuto(w, r)
					return
				}
			}

			if !isValidHost(hostWithoutPort, allowedHosts, c.AllowSubdomains) {
				detail := problem.NewDetail(c.StatusCode, c.Message)
				_ = detail.RenderAuto(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isValidHost checks if the given host matches any of the allowed hosts.
// If allowSubdomains is true, subdomains of allowed hosts are also valid.
func isValidHost(host string, allowedHosts []string, allowSubdomains bool) bool {
	// Normalize FQDN trailing dot (example.com. -> example.com)
	host = strings.TrimSuffix(host, ".")

	for _, allowed := range allowedHosts {
		allowed = strings.TrimSuffix(allowed, ".")

		if strings.EqualFold(host, allowed) {
			return true
		}

		if allowSubdomains {
			// Host is exactly "example.com" and allowed is "example.com" (already handled above)
			// Or host is "sub.example.com" and allowed is "example.com"
			if strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(allowed)) {
				return true
			}
		}
	}
	return false
}
