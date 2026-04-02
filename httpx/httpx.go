// Package httpx provides HTTP header names, content types, and other
// constants for use throughout the framework.
//
// Content-Type constants include (among others):
//
//	[MIMETextHTML], [MIMETextPlain], [MIMEApplicationJSON]
//	[MIMEApplicationProblemJSON], [MIMEApplicationFormURLEncoded]
//
// Header name constants include (among others):
//
//	[HeaderContentType], [HeaderAuthorization], [HeaderXRequestId]
//	[HeaderCacheControl], [HeaderETag], [HeaderLocation]
//
// CORS and Security headers include (among others):
//
//	[HeaderAccessControlAllowOrigin]
//	[HeaderContentSecurityPolicy]
//	[HeaderStrictTransportSecurity]
//
// See the full list of constants below for complete coverage of headers,
// content types, authentication schemes, and common header values.
package httpx

const (
	MIMETextHTML                  = "text/html"
	MIMETextHTMLCharset           = "text/html; charset=utf-8"
	MIMETextPlain                 = "text/plain"
	MIMETextPlainCharset          = "text/plain; charset=utf-8"
	MIMETextEventStream           = "text/event-stream"
	MIMETextCSS                   = "text/css"
	MIMETextJavaScript            = "text/javascript"
	MIMEApplicationJSON           = "application/json"
	MIMEApplicationJSONCharset    = "application/json; charset=utf-8"
	MIMEApplicationJavaScript     = "application/javascript"
	MIMEApplicationXML            = "application/xml"
	MIMEApplicationProblemJSON    = "application/problem+json"
	MIMEApplicationFormURLEncoded = "application/x-www-form-urlencoded"
	MIMEApplicationRSSXML         = "application/rss+xml"
	MIMEApplicationAtomXML        = "application/atom+xml"
	MIMEMultipartFormData         = "multipart/form-data"
	MIMETextXML                   = "text/xml"
	MIMEImageSVGXML               = "image/svg+xml"
)

// Request Headers
const (
	HeaderAccept            = "Accept"
	HeaderAcceptCharset     = "Accept-Charset"
	HeaderAcceptEncoding    = "Accept-Encoding"
	HeaderAcceptLanguage    = "Accept-Language"
	HeaderAcceptRanges      = "Accept-Ranges"
	HeaderAuthorization     = "Authorization"
	HeaderCacheControl      = "Cache-Control"
	HeaderConnection        = "Connection"
	HeaderContentLength     = "Content-Length"
	HeaderContentType       = "Content-Type"
	HeaderCookie            = "Cookie"
	HeaderDate              = "Date"
	HeaderExpect            = "Expect"
	HeaderForwarded         = "Forwarded"
	HeaderFrom              = "From"
	HeaderHost              = "Host"
	HeaderIfMatch           = "If-Match"
	HeaderIfModifiedSince   = "If-Modified-Since"
	HeaderIfNoneMatch       = "If-None-Match"
	HeaderIfRange           = "If-Range"
	HeaderIfUnmodifiedSince = "If-Unmodified-Since"
	HeaderLastEventId       = "Last-Event-Id"
	HeaderMaxForwards       = "Max-Forwards"
	HeaderOrigin            = "Origin"
	HeaderPragma            = "Pragma"
	HeaderProxyAuthorization = "Proxy-Authorization"
	HeaderRange             = "Range"
	HeaderReferer           = "Referer"
	HeaderTE                = "TE"
	HeaderUserAgent         = "User-Agent"
	HeaderUpgrade           = "Upgrade"
	HeaderVia               = "Via"
	HeaderWarning           = "Warning"
)

// Response Headers
const (
	HeaderAcceptPatch        = "Accept-Patch"
	HeaderAcceptPost         = "Accept-Post"
	HeaderAge                = "Age"
	HeaderAllow              = "Allow"
	HeaderAltSvc             = "Alt-Svc"
	HeaderContentDisposition = "Content-Disposition"
	HeaderContentEncoding    = "Content-Encoding"
	HeaderContentLanguage    = "Content-Language"
	HeaderContentLocation    = "Content-Location"
	HeaderContentRange       = "Content-Range"
	HeaderETag               = "ETag"
	HeaderExpires            = "Expires"
	HeaderIdempotencyKey     = "Idempotency-Key"
	HeaderKeepAlive          = "Keep-Alive"
	HeaderLastModified       = "Last-Modified"
	HeaderLink               = "Link"
	HeaderLocation           = "Location"
	HeaderProxyAuthenticate = "Proxy-Authenticate"
	HeaderRetryAfter        = "Retry-After"
	HeaderServer             = "Server"
	HeaderSetCookie          = "Set-Cookie"
	HeaderTrailer            = "Trailer"
	HeaderTransferEncoding   = "Transfer-Encoding"
	HeaderVary               = "Vary"
	HeaderWWWAuthenticate    = "WWW-Authenticate"
)

// CORS Headers
const (
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
)

// Security Headers
const (
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderCrossOriginEmbedderPolicy       = "Cross-Origin-Embedder-Policy"
	HeaderCrossOriginOpenerPolicy         = "Cross-Origin-Opener-Policy"
	HeaderCrossOriginResourcePolicy       = "Cross-Origin-Resource-Policy"
	HeaderFeaturePolicy                   = "Feature-Policy"
	HeaderPermissionsPolicy               = "Permissions-Policy"
	HeaderReferrerPolicy                  = "Referrer-Policy"
	HeaderSecFetchSite                    = "Sec-Fetch-Site"
	HeaderStrictTransportSecurity         = "Strict-Transport-Security"
	HeaderXContentTypeOptions             = "X-Content-Type-Options"
	HeaderXFrameOptions                   = "X-Frame-Options"
	HeaderXXSSProtection                  = "X-XSS-Protection"
)

// Custom/Extension Headers
const (
	HeaderXAccelExpires       = "X-Accel-Expires"
	HeaderXAPIKey             = "X-API-Key"
	HeaderXCache              = "X-Cache"
	HeaderXCSRFToken          = "X-CSRF-Token"
	HeaderXForwarded          = "X-Forwarded"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXForwardedHost      = "X-Forwarded-Host"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedProtocol  = "X-Forwarded-Protocol"
	HeaderXForwardedSsl       = "X-Forwarded-Ssl"
	HeaderXIdempotencyReplay  = "X-Idempotency-Replay"
	HeaderXPoweredBy          = "X-Powered-By"
	HeaderXRateLimitLimit     = "X-RateLimit-Limit"
	HeaderXRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderXRateLimitReset     = "X-RateLimit-Reset"
	HeaderXRateLimitWindow    = "X-RateLimit-Window"
	HeaderXRealIP             = "X-Real-IP"
	HeaderXRequestId          = "X-Request-Id"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderXTimestamp          = "X-Timestamp"
	HeaderXTotal              = "X-Total"
	HeaderXTotalPages         = "X-Total-Pages"
	HeaderXPage               = "X-Page"
	HeaderXPerPage            = "X-Per-Page"
	HeaderXPrevPage           = "X-Prev-Page"
	HeaderXNextPage           = "X-Next-Page"
)

// WebSocket Headers
const (
	UpgradeWebSocket             = "websocket"
	HeaderSecWebSocketKey        = "Sec-WebSocket-Key"
	HeaderSecWebSocketAccept     = "Sec-WebSocket-Accept"
	HeaderSecWebSocketVersion    = "Sec-WebSocket-Version"
	HeaderSecWebSocketProtocol   = "Sec-WebSocket-Protocol"
	HeaderSecWebSocketExtensions = "Sec-WebSocket-Extensions"
)

// Authentication Schemes
const (
	AuthSchemeBasic  = "Basic"
	AuthSchemeBearer = "Bearer"
	AuthSchemeDigest = "Digest"
	AuthSchemeOAuth  = "OAuth"
)

// Common Header Values
const (
	ConnectionClose     = "close"
	ConnectionKeepAlive = "keep-alive"
	ConnectionUpgrade   = "Upgrade"

	CacheControlNoCache        = "no-cache"
	CacheControlNoStore        = "no-store"
	CacheControlNoTransform    = "no-transform"
	CacheControlMaxAge         = "max-age"
	CacheControlMustRevalidate = "must-revalidate"
	CacheControlPrivate        = "private"
	CacheControlPublic         = "public"

	ContentEncodingGzip    = "gzip"
	ContentEncodingDeflate = "deflate"
	ContentEncodingBrotli  = "br"
	ContentEncodingZstd    = "zstd"

	TransferEncodingChunked = "chunked"

	XCacheHit  = "HIT"
	XCacheMiss = "MISS"
)

const (
	QueryXHMACAlgorithm     = "X-HMAC-Algorithm"
	QueryXHMACCredential    = "X-HMAC-Credential"
	QueryXHMACSignedHeaders = "X-HMAC-SignedHeaders"
	QueryXHMACSignature     = "X-HMAC-Signature"
)
