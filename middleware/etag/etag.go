package etag

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/metrics"
)

// New creates an ETag middleware with the provided configuration
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "ETag")

	if c.Algorithm != FNV && c.Algorithm != MD5 {
		c.Algorithm = DefaultConfig.Algorithm
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			if c.ExcludedFunc != nil && c.ExcludedFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch)
			ifMatch := r.Header.Get(httpx.HeaderIfMatch)
			ifRange := r.Header.Get(httpx.HeaderIfRange)
			rangeHeader := r.Header.Get(httpx.HeaderRange)

			// Only generate ETags for GET and HEAD requests (for caching)
			// But still check If-Match for state-changing methods
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				// For PUT/PATCH/DELETE, check If-Match but don't generate/cache ETags
				if ifMatch != "" {
					// Pass through with a wrapper that checks If-Match
					ew := newETagResponseWriter(w, c, "", ifMatch, "", "", reg)
					defer ew.release()
					next.ServeHTTP(ew, r)
					ew.finalize()
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			ew := newETagResponseWriter(w, c, ifNoneMatch, ifMatch, ifRange, rangeHeader, reg)
			defer ew.release()

			next.ServeHTTP(ew, r)

			ew.finalize()
		})
	}
}

// GenerateFromFile generates an etag for a file based on its modification time and size.
// This is much more efficient than hashing the file content, especially for large files.
// The format is: W/"mtime-size" (weak etag) or "mtime-size" (strong etag)
// Example: W/"1709999999-1024"
//
// Usage:
//
//	file, _ := os.Open("largefile.zip")
//	stat, _ := file.Stat()
//	etag := etag.GenerateFromFile(stat.ModTime().Unix(), stat.Size(), true) // weak ETag
//	w.Header().Set(httpx.HeaderETag, etag)
func GenerateFromFile(modTime int64, size int64, weak bool) string {
	tag := strconv.FormatInt(modTime, 10) + "-" + strconv.FormatInt(size, 10)
	if weak {
		return `W/"` + tag + `"`
	}
	return `"` + tag + `"`
}

// GenerateFromFileInfo generates an etag from fs.FileInfo or os.FileInfo.
// This helper handles both interface types properly.
//
// Usage:
//
//	file, _ := os.Open("largefile.zip")
//	stat, _ := file.Stat()
//	etag := etag.GenerateFromFileInfo(stat, true)
//	w.Header().Set(httpx.HeaderETag, etag)
func GenerateFromFileInfo(info interface {
	ModTime() time.Time
	Size() int64
}, weak bool,
) string {
	return GenerateFromFile(info.ModTime().Unix(), info.Size(), weak)
}

// Parse extracts the hash value from an etag, handling weak ETags
// Returns the hash value and a boolean indicating if it was a weak etag
// Example: Parse(`W/"abc123"`) returns ("abc123", true)
// Example: Parse(`"abc123"`) returns ("abc123", false)
func Parse(etag string) (string, bool) {
	if strings.HasPrefix(etag, `W/"`) && strings.HasSuffix(etag, `"`) {
		return etag[3 : len(etag)-1], true
	}
	if strings.HasPrefix(etag, `"`) && strings.HasSuffix(etag, `"`) {
		return etag[1 : len(etag)-1], false
	}
	return etag, false
}

// ServeContentWithETag serves content with automatic etag support.
// It handles If-None-Match and If-Range headers properly.
// Similar to http.ServeContent but with our etag generation logic.
func ServeContentWithETag(w http.ResponseWriter, r *http.Request, modTime int64, content io.ReadSeeker) {
	if content == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var size int64
	if seeker, ok := content.(io.Seeker); ok {
		end, err := seeker.Seek(0, io.SeekEnd)
		if err == nil {
			size = end
			_, _ = seeker.Seek(0, io.SeekStart)
		}
	}

	etag := GenerateFromFile(modTime, size, true)

	if ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch); ifNoneMatch != "" {
		if Matches(ifNoneMatch, etag) {
			w.Header().Set(httpx.HeaderETag, etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Header().Set(httpx.HeaderETag, etag)

	// Serve content using http.ServeContent which handles Range requests
	// We need to convert int64 modTime to time.Time
	http.ServeContent(w, r, "", time.Unix(modTime, 0), content)
}

// Matches checks if the provided etag matches any in the If-None-Match header
func Matches(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "*" {
		return true
	}

	for _, et := range strings.Split(ifNoneMatch, ",") {
		et = strings.TrimSpace(et)
		// Compare weak ETags ignoring the W/ prefix
		if strings.HasPrefix(et, "W/") && strings.HasPrefix(etag, "W/") {
			if et == etag {
				return true
			}
		} else if strings.HasPrefix(et, "W/") {
			// Client has weak, we have strong - compare values
			if et[2:] == etag {
				return true
			}
		} else if strings.HasPrefix(etag, "W/") {
			// Client has strong, we have weak - compare values
			if et == etag[2:] {
				return true
			}
		} else {
			if et == etag {
				return true
			}
		}
	}
	return false
}
