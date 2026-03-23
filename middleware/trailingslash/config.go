package trailingslash

import "net/http"

// Action defines the action to take for trailing slash mismatches
type Action string

const (
	// RedirectAction redirects to the canonical URL (default)
	RedirectAction Action = "redirect"
	// StripAction removes trailing slash and continues processing
	StripAction Action = "strip"
	// AppendAction adds trailing slash and continues processing
	AppendAction Action = "append"
)

// Config allows customization of trailing slash handling
type Config struct {
	// Action to take when trailing slash doesn't match preference (defaults to redirect)
	Action Action

	// PreferTrailingSlash determines if URLs should have trailing slashes (defaults to false)
	PreferTrailingSlash bool

	// RedirectCode for redirects (defaults to 301 Moved Permanently)
	RedirectCode int
}

// DefaultConfig contains the default values for trailing slash configuration.
var DefaultConfig = Config{
	Action:              RedirectAction,
	PreferTrailingSlash: false, // Most APIs prefer no trailing slash
	RedirectCode:        http.StatusMovedPermanently,
}
