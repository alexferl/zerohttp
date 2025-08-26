package zerohttp

const (
	MIMETextHTML           = "text/html; charset=utf-8"
	MIMETextPlain          = "text/plain; charset=utf-8"
	MIMEApplicationJSON    = "application/json; charset=utf-8"
	MIMEApplicationProblem = "application/problem+json"
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
	HeaderMaxForwards       = "Max-Forwards"
	HeaderOrigin            = "Origin"
	HeaderPragma            = "Pragma"
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
	HeaderAge                = "Age"
	HeaderAllow              = "Allow"
	HeaderContentDisposition = "Content-Disposition"
	HeaderContentEncoding    = "Content-Encoding"
	HeaderContentLanguage    = "Content-Language"
	HeaderContentLocation    = "Content-Location"
	HeaderContentRange       = "Content-Range"
	HeaderETag               = "ETag"
	HeaderExpires            = "Expires"
	HeaderLastModified       = "Last-Modified"
	HeaderLink               = "Link"
	HeaderLocation           = "Location"
	HeaderProxyAuthenticate  = "Proxy-Authenticate"
	HeaderRetryAfter         = "Retry-After"
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
	HeaderXAPIKey             = "X-API-Key"
	HeaderXCSRFToken          = "X-CSRF-Token"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXForwardedHost      = "X-Forwarded-Host"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedProtocol  = "X-Forwarded-Protocol"
	HeaderXForwardedSsl       = "X-Forwarded-Ssl"
	HeaderXRealIP             = "X-Real-IP"
	HeaderXRequestID          = "X-Request-ID"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderXPoweredBy          = "X-Powered-By"
	HeaderXRateLimitLimit     = "X-RateLimit-Limit"
	HeaderXRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderXRateLimitReset     = "X-RateLimit-Reset"
)

// WebSocket Headers
const (
	HeaderUpgradeWebSocket       = "websocket"
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
	CacheControlMustRevalidate = "must-revalidate"
	CacheControlPublic         = "public"
	CacheControlPrivate        = "private"
	CacheControlMaxAge         = "max-age"

	ContentEncodingGzip    = "gzip"
	ContentEncodingDeflate = "deflate"
	ContentEncodingBrotli  = "br"

	TransferEncodingChunked = "chunked"
)
