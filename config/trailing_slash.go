package config

import "net/http"

// TrailingSlashAction defines the action to take for trailing slash mismatches
type TrailingSlashAction string

const (
	// RedirectAction redirects to the canonical URL (default)
	RedirectAction TrailingSlashAction = "redirect"
	// StripAction removes trailing slash and continues processing
	StripAction TrailingSlashAction = "strip"
	// AppendAction adds trailing slash and continues processing
	AppendAction TrailingSlashAction = "append"
)

// TrailingSlashConfig allows customization of trailing slash handling
type TrailingSlashConfig struct {
	// Action to take when trailing slash doesn't match preference (defaults to redirect)
	Action TrailingSlashAction
	// PreferTrailingSlash determines if URLs should have trailing slashes (defaults to false)
	PreferTrailingSlash bool
	// RedirectCode for redirects (defaults to 301 Moved Permanently)
	RedirectCode int
}

// DefaultTrailingSlashConfig contains the default values for trailing slash configuration.
var DefaultTrailingSlashConfig = TrailingSlashConfig{
	Action:              RedirectAction,
	PreferTrailingSlash: false, // Most APIs prefer no trailing slash
	RedirectCode:        http.StatusMovedPermanently,
}

// TrailingSlashOption configures trailing slash middleware.
type TrailingSlashOption func(*TrailingSlashConfig)

// WithTrailingSlashAction sets the action to take for trailing slash mismatches.
func WithTrailingSlashAction(action TrailingSlashAction) TrailingSlashOption {
	return func(c *TrailingSlashConfig) {
		c.Action = action
	}
}

// WithTrailingSlashPreference sets whether URLs should have trailing slashes.
func WithTrailingSlashPreference(prefer bool) TrailingSlashOption {
	return func(c *TrailingSlashConfig) {
		c.PreferTrailingSlash = prefer
	}
}

// WithTrailingSlashRedirectCode sets the redirect status code for redirects.
func WithTrailingSlashRedirectCode(code int) TrailingSlashOption {
	return func(c *TrailingSlashConfig) {
		c.RedirectCode = code
	}
}
