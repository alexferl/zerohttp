package mediatype

// Config allows customization of allowed media types
type Config struct {
	// AllowedTypes is a list of allowed media type patterns.
	// Supports wildcards (*) and suffix matching (+json, +xml, etc.).
	// Examples:
	//   - "application/vnd.api+json" - exact vendor type
	//   - "application/*+json" - any JSON-based media type
	//   - "application/vnd.company.*+json" - vendor types with wildcards
	// Default: [] (no validation, allows any)
	AllowedTypes []string

	// DefaultType is the media type to use when the client accepts any type (*/*)
	// or when no Accept header is provided. This value is set as the Accept header
	// on the request, allowing handlers to perform content negotiation.
	// If empty, the Accept header is left as-is.
	// Default: ""
	DefaultType string

	// ValidateContentType also validates the request Content-Type header.
	// When true, both Accept and Content-Type must match AllowedTypes.
	// Default: false
	ValidateContentType bool

	// ResponseTypeHeader is the response header name to set with the effective
	// media type. e.g. "X-App-Media-Type"
	// Default: ""
	ResponseTypeHeader string

	// ResponseTypeFunc transforms the negotiated media type into the response header value.
	// The negotiated type is either the client's requested type (if matched) or DefaultType.
	// If nil, the negotiated type is used as-is.
	//
	// Use VendorShortType to extract short identifiers from vendor types:
	//
	//    "application/vnd.app.v1+json" -> "app.v1"
	//
	// Or provide a custom function for different formatting:
	//
	//    ResponseTypeFunc: func(t string) string {
	//        return "version=" + t
	//    }
	//    // "application/vnd.app.v1+json" -> "version=application/vnd.app.v1+json"
	ResponseTypeFunc func(string) string

	// ExcludedPaths contains paths that skip media type validation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where media type validation is explicitly applied.
	// If set, validation will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, validation applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains the default values for media type configuration.
var DefaultConfig = Config{
	AllowedTypes:        []string{},
	DefaultType:         "",
	ValidateContentType: false,
	ResponseTypeHeader:  "",
	ResponseTypeFunc:    nil,
	ExcludedPaths:       []string{},
	IncludedPaths:       []string{},
}
