package config

import (
	"net/http"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

var Epoch = time.Unix(0, 0).UTC().Format(http.TimeFormat)

var DefaultNoCacheHeaders = map[string]string{
	httpx.HeaderExpires:       Epoch,
	httpx.HeaderCacheControl:  "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
	httpx.HeaderPragma:        httpx.CacheControlNoCache,
	httpx.HeaderXAccelExpires: "0",
}

var DefaultETagHeaders = []string{
	httpx.HeaderETag,
	httpx.HeaderIfModifiedSince,
	httpx.HeaderIfMatch,
	httpx.HeaderIfNoneMatch,
	httpx.HeaderIfRange,
	httpx.HeaderIfUnmodifiedSince,
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
