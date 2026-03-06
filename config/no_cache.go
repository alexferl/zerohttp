package config

import (
	"net/http"
	"time"
)

var Epoch = time.Unix(0, 0).UTC().Format(http.TimeFormat)

var DefaultNoCacheHeaders = map[string]string{
	"Expires":         Epoch,
	"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

var DefaultETagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// NoCacheConfig allows customization of the set/deleted headers
type NoCacheConfig struct {
	NoCacheHeaders map[string]string // Headers to set for no-cache; defaults provided
	ETagHeaders    []string          // Headers to delete; defaults provided
}

// DefaultNoCacheConfig contains the default values for no-cache configuration.
var DefaultNoCacheConfig = NoCacheConfig{
	NoCacheHeaders: DefaultNoCacheHeaders,
	ETagHeaders:    DefaultETagHeaders,
}
