package middleware

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// HMACSigner provides client-side request signing for HMAC authentication
type HMACSigner struct {
	accessKeyID          string
	secretKey            string
	algorithm            config.HMACHashAlgorithm
	allowUnsignedPayload bool
	headersToSign        []string
}

// NewHMACSigner creates a new HMAC signer with the given credentials
// Defaults to HMAC-SHA256 algorithm
func NewHMACSigner(accessKeyID, secretKey string) *HMACSigner {
	return &HMACSigner{
		accessKeyID: accessKeyID,
		secretKey:   secretKey,
		algorithm:   config.HMACSHA256,
	}
}

// NewHMACSignerWithAlgorithm creates a signer with a specific algorithm
func NewHMACSignerWithAlgorithm(accessKeyID, secretKey string, algorithm config.HMACHashAlgorithm) *HMACSigner {
	return &HMACSigner{
		accessKeyID: accessKeyID,
		secretKey:   secretKey,
		algorithm:   algorithm,
	}
}

// AccessKeyID returns the signer's access key ID
func (s *HMACSigner) AccessKeyID() string {
	return s.accessKeyID
}

// Algorithm returns the signer's algorithm
func (s *HMACSigner) Algorithm() config.HMACHashAlgorithm {
	return s.algorithm
}

// SetAllowUnsignedPayload sets whether to use "UNSIGNED-PAYLOAD" for body hash.
// This is useful for streaming large requests where computing body hash is impractical.
func (s *HMACSigner) SetAllowUnsignedPayload(allow bool) {
	s.allowUnsignedPayload = allow
}

// SetHeadersToSign sets the list of header names to include in the signature.
// Headers are signed in the order specified. Use lowercase header names.
// Default: ["host", "x-timestamp", "content-type" (if present)]
func (s *HMACSigner) SetHeadersToSign(headers []string) {
	s.headersToSign = headers
}

// SignRequest signs an HTTP request with HMAC authentication.
// It adds the Authorization header and X-Timestamp header.
func (s *HMACSigner) SignRequest(req *http.Request) error {
	return s.SignRequestWithTime(req, time.Now().UTC())
}

// SignRequestWithTime signs a request with a specific timestamp.
// Useful for testing or creating pre-signed URLs with specific expiration.
func (s *HMACSigner) SignRequestWithTime(req *http.Request, timestamp time.Time) error {
	if req.Host == "" && req.URL != nil {
		req.Host = req.URL.Host
	}

	req.Header.Set("X-Timestamp", timestamp.Format(time.RFC3339))

	bodyHash := s.computeBodyHash(req)

	signedHeaders := s.buildSignedHeadersList(req)

	canonicalRequest := s.buildCanonicalRequest(req, signedHeaders, bodyHash)

	signature := s.computeSignature(canonicalRequest)

	authHeader := s.buildAuthorizationHeader(timestamp, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

// GenerateSignature computes the signature without modifying the request.
// Useful for pre-signed URLs or manual header management.
func (s *HMACSigner) GenerateSignature(req *http.Request, timestamp time.Time) (string, error) {
	host := req.Host
	if host == "" && req.URL != nil {
		host = req.URL.Host
	}

	// Set the X-Timestamp header for signing
	req.Header.Set("X-Timestamp", timestamp.Format(time.RFC3339))

	signedHeaders := []string{"host", "x-timestamp"}
	if req.Header.Get("Content-Type") != "" {
		signedHeaders = append(signedHeaders, "content-type")
	}

	bodyHash := s.computeBodyHash(req)

	var b strings.Builder
	b.WriteString(strings.ToUpper(req.Method))
	b.WriteByte('\n')
	b.WriteString(url.PathEscape(req.URL.Path))
	b.WriteByte('\n')
	b.WriteString(s.buildCanonicalQueryString(req.URL.Query()))
	b.WriteByte('\n')

	for _, h := range signedHeaders {
		value := ""
		if h == "host" {
			value = host
		} else {
			value = req.Header.Get(h)
		}
		b.WriteString(h + ":" + strings.TrimSpace(value) + "\n")
	}
	b.WriteByte('\n')
	b.WriteString(bodyHash)

	sig := s.computeSignature(b.String())
	return base64.StdEncoding.EncodeToString(sig), nil
}

// PresignURL generates a pre-signed URL with HMAC parameters in query string
func (s *HMACSigner) PresignURL(req *http.Request, expiry time.Duration) (string, error) {
	return s.PresignURLWithTime(req, time.Now().UTC().Add(expiry))
}

// PresignURLWithTime generates a pre-signed URL with specific expiration time
func (s *HMACSigner) PresignURLWithTime(req *http.Request, expiresAt time.Time) (string, error) {
	sig, err := s.GenerateSignature(req, expiresAt)
	if err != nil {
		return "", err
	}

	u := req.URL
	q := u.Query()
	q.Set("X-HMAC-Algorithm", "HMAC-"+string(s.algorithm))
	q.Set("X-HMAC-Credential", s.accessKeyID+"/"+expiresAt.Format(time.RFC3339))
	q.Set("X-HMAC-SignedHeaders", "host;x-timestamp")
	q.Set("X-HMAC-Signature", sig)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// computeBodyHash computes the hash of the request body
// Returns "UNSIGNED-PAYLOAD" if allowUnsignedPayload is set
func (s *HMACSigner) computeBodyHash(req *http.Request) string {
	if s.allowUnsignedPayload {
		return "UNSIGNED-PAYLOAD"
	}

	var h hash.Hash
	switch s.algorithm {
	case config.HMACSHA256:
		h = sha256.New()
	case config.HMACSHA384:
		h = sha512.New384()
	case config.HMACSHA512:
		h = sha512.New()
	default:
		h = sha256.New()
	}

	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		h.Write(body)
		// Restore body
		req.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// buildSignedHeadersList builds the list of headers to sign
// If headersToSign is set, uses that list; otherwise uses defaults
func (s *HMACSigner) buildSignedHeadersList(req *http.Request) []string {
	if len(s.headersToSign) > 0 {
		var headers []string
		for _, h := range s.headersToSign {
			h = strings.ToLower(strings.TrimSpace(h))
			if h == "host" {
				headers = append(headers, h)
				continue
			}
			if req.Header.Get(h) != "" {
				headers = append(headers, h)
			}
		}
		return headers
	}

	headers := []string{"host", "x-timestamp"}

	if req.Header.Get("Content-Type") != "" {
		headers = append(headers, "content-type")
	}

	return headers
}

// buildCanonicalRequest creates the canonical request string
func (s *HMACSigner) buildCanonicalRequest(req *http.Request, signedHeaders []string, bodyHash string) string {
	var b strings.Builder

	b.WriteString(strings.ToUpper(req.Method))
	b.WriteByte('\n')
	b.WriteString(url.PathEscape(req.URL.Path))
	b.WriteByte('\n')
	b.WriteString(s.buildCanonicalQueryString(req.URL.Query()))
	b.WriteByte('\n')

	for _, h := range signedHeaders {
		value := ""
		if h == "host" {
			value = req.Host
		} else {
			value = req.Header.Get(h)
		}
		b.WriteString(h + ":" + strings.TrimSpace(value) + "\n")
	}
	b.WriteByte('\n')
	b.WriteString(bodyHash)

	return b.String()
}

// buildCanonicalQueryString builds the canonical query string
func (s *HMACSigner) buildCanonicalQueryString(values url.Values) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		vals := values[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	return strings.Join(parts, "&")
}

// computeSignature computes the HMAC signature
func (s *HMACSigner) computeSignature(canonicalRequest string) []byte {
	return computeHMACSignature(s.secretKey, canonicalRequest, s.algorithm)
}

// buildAuthorizationHeader builds the Authorization header value
func (s *HMACSigner) buildAuthorizationHeader(timestamp time.Time, signedHeaders []string, signature []byte) string {
	algo := "HMAC-" + string(s.algorithm)
	credential := s.accessKeyID + "/" + timestamp.Format(time.RFC3339)
	signedHeadersStr := strings.Join(signedHeaders, ";")
	sigB64 := base64.StdEncoding.EncodeToString(signature)

	return algo + " " +
		"Credential=" + credential + ", " +
		"SignedHeaders=" + signedHeadersStr + ", " +
		"Signature=" + sigB64
}
