package validator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// registerBuiltins registers all built-in validation functions.
func (v *defaultValidator) registerBuiltins() {
	// Create a local map to build the validators atomically
	validators := make(map[string]ValidationFunc)

	// required - field must not be zero value
	validators["required"] = func(value reflect.Value, tag string) error {
		if isZeroValue(value) {
			return fmt.Errorf("required")
		}
		return nil
	}

	// omitempty - skip validation if zero value (handled in validateField)
	validators["omitempty"] = func(value reflect.Value, tag string) error {
		return nil // Handled in validateField before calling validators
	}

	// min - minimum value for numbers, minimum length for strings/slices/arrays/maps
	validators["min"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if utf8.RuneCountInString(value.String()) < length {
				return fmt.Errorf("must be at least %d characters", length)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Int() < target {
				return fmt.Errorf("must be at least %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Uint() < target {
				return fmt.Errorf("must be at least %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Float() < target {
				return fmt.Errorf("must be at least %v", target)
			}
		case reflect.Slice, reflect.Array, reflect.Map:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid min parameter: %s", tag)
			}
			if value.Len() < length {
				return fmt.Errorf("must have at least %d items", length)
			}
		default:
			return fmt.Errorf("min not supported for type %s", value.Kind())
		}
		return nil
	}

	// max - maximum value for numbers, maximum length for strings/slices/arrays/maps
	validators["max"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if utf8.RuneCountInString(value.String()) > length {
				return fmt.Errorf("must be at most %d characters", length)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Int() > target {
				return fmt.Errorf("must be at most %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Uint() > target {
				return fmt.Errorf("must be at most %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Float() > target {
				return fmt.Errorf("must be at most %v", target)
			}
		case reflect.Slice, reflect.Array, reflect.Map:
			length, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid max parameter: %s", tag)
			}
			if value.Len() > length {
				return fmt.Errorf("must have at most %d items", length)
			}
		default:
			return fmt.Errorf("max not supported for type %s", value.Kind())
		}
		return nil
	}

	// len - exact length for strings/slices/arrays
	validators["len"] = func(value reflect.Value, tag string) error {
		length, err := strconv.Atoi(tag)
		if err != nil {
			return fmt.Errorf("invalid len parameter: %s", tag)
		}

		switch value.Kind() {
		case reflect.String:
			if utf8.RuneCountInString(value.String()) != length {
				return fmt.Errorf("must be exactly %d characters", length)
			}
		case reflect.Slice, reflect.Array, reflect.Map:
			if value.Len() != length {
				return fmt.Errorf("must have exactly %d items", length)
			}
		default:
			return fmt.Errorf("len not supported for type %s", value.Kind())
		}
		return nil
	}

	// email - validates email address format
	validators["email"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("email validator only supports strings")
		}
		email := value.String()
		if email == "" {
			return nil // Let "required" handle empty strings
		}
		_, err := mail.ParseAddress(email)
		if err != nil {
			return fmt.Errorf("invalid email format")
		}
		return nil
	}

	// url - validates URL format
	validators["url"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("url validator only supports strings")
		}
		urlStr := value.String()
		if urlStr == "" {
			return nil // Let "required" handle empty strings
		}
		// Simple URL validation
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			return fmt.Errorf("invalid URL format")
		}
		return nil
	}

	// alpha - only alphabetic characters
	validators["alpha"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("alpha validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return fmt.Errorf("must contain only letters")
			}
		}
		return nil
	}

	// alphanum - only alphanumeric characters
	validators["alphanum"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("alphanum validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
				return fmt.Errorf("must contain only letters and numbers")
			}
		}
		return nil
	}

	// numeric - only numeric characters
	validators["numeric"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("numeric validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsNumber(r) {
				return fmt.Errorf("must contain only numbers")
			}
		}
		return nil
	}

	// oneof - value must be one of the specified options
	validators["oneof"] = func(value reflect.Value, tag string) error {
		options := strings.Fields(tag)
		if len(options) == 0 {
			return nil
		}

		switch value.Kind() {
		case reflect.String:
			s := value.String()
			if s == "" {
				return nil
			}
			for _, opt := range options {
				if s == opt {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", strings.Join(options, ", "))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := value.Int()
			for _, opt := range options {
				if i, err := strconv.ParseInt(opt, 10, 64); err == nil && n == i {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", tag)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := value.Uint()
			for _, opt := range options {
				if i, err := strconv.ParseUint(opt, 10, 64); err == nil && n == i {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", tag)
		default:
			return fmt.Errorf("oneof not supported for type %s", value.Kind())
		}
	}

	// eq - equal to value
	validators["eq"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			if value.String() != tag {
				return fmt.Errorf("must be equal to %s", tag)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid eq parameter: %s", tag)
			}
			if value.Int() != n {
				return fmt.Errorf("must be equal to %d", n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid eq parameter: %s", tag)
			}
			if value.Uint() != n {
				return fmt.Errorf("must be equal to %d", n)
			}
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid eq parameter: %s", tag)
			}
			if value.Float() != n {
				return fmt.Errorf("must be equal to %v", n)
			}
		default:
			return fmt.Errorf("eq not supported for type %s", value.Kind())
		}
		return nil
	}

	// ne - not equal to value
	validators["ne"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.String:
			if value.String() == tag {
				return fmt.Errorf("must not be equal to %s", tag)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid ne parameter: %s", tag)
			}
			if value.Int() == n {
				return fmt.Errorf("must not be equal to %d", n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid ne parameter: %s", tag)
			}
			if value.Uint() == n {
				return fmt.Errorf("must not be equal to %d", n)
			}
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid ne parameter: %s", tag)
			}
			if value.Float() == n {
				return fmt.Errorf("must not be equal to %v", n)
			}
		default:
			return fmt.Errorf("ne not supported for type %s", value.Kind())
		}
		return nil
	}

	// gt - greater than
	validators["gt"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gt parameter: %s", tag)
			}
			if value.Int() <= target {
				return fmt.Errorf("must be greater than %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gt parameter: %s", tag)
			}
			if value.Uint() <= target {
				return fmt.Errorf("must be greater than %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid gt parameter: %s", tag)
			}
			if value.Float() <= target {
				return fmt.Errorf("must be greater than %v", target)
			}
		default:
			return fmt.Errorf("gt not supported for type %s", value.Kind())
		}
		return nil
	}

	// lt - less than
	validators["lt"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lt parameter: %s", tag)
			}
			if value.Int() >= target {
				return fmt.Errorf("must be less than %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lt parameter: %s", tag)
			}
			if value.Uint() >= target {
				return fmt.Errorf("must be less than %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid lt parameter: %s", tag)
			}
			if value.Float() >= target {
				return fmt.Errorf("must be less than %v", target)
			}
		default:
			return fmt.Errorf("lt not supported for type %s", value.Kind())
		}
		return nil
	}

	// gte - greater than or equal
	validators["gte"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gte parameter: %s", tag)
			}
			if value.Int() < target {
				return fmt.Errorf("must be greater than or equal to %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gte parameter: %s", tag)
			}
			if value.Uint() < target {
				return fmt.Errorf("must be greater than or equal to %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid gte parameter: %s", tag)
			}
			if value.Float() < target {
				return fmt.Errorf("must be greater than or equal to %v", target)
			}
		default:
			return fmt.Errorf("gte not supported for type %s", value.Kind())
		}
		return nil
	}

	// lte - less than or equal
	validators["lte"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			target, err := strconv.ParseInt(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lte parameter: %s", tag)
			}
			if value.Int() > target {
				return fmt.Errorf("must be less than or equal to %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid lte parameter: %s", tag)
			}
			if value.Uint() > target {
				return fmt.Errorf("must be less than or equal to %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := strconv.ParseFloat(tag, 64)
			if err != nil {
				return fmt.Errorf("invalid lte parameter: %s", tag)
			}
			if value.Float() > target {
				return fmt.Errorf("must be less than or equal to %v", target)
			}
		default:
			return fmt.Errorf("lte not supported for type %s", value.Kind())
		}
		return nil
	}

	// uuid - validates UUID format
	validators["uuid"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("uuid validator only supports strings")
		}
		uuid := value.String()
		if uuid == "" {
			return nil
		}
		if !uuidRegex.MatchString(uuid) {
			return fmt.Errorf("invalid UUID format")
		}
		return nil
	}

	// datetime - validates datetime format
	validators["datetime"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("datetime validator only supports strings")
		}
		dt := value.String()
		if dt == "" {
			return nil
		}
		// Default to RFC3339 if no format specified
		format := tag
		if format == "" {
			format = time.RFC3339
		}
		_, err := time.Parse(format, dt)
		if err != nil {
			return fmt.Errorf("invalid datetime format, expected %s", format)
		}
		return nil
	}

	// ===== String Content Validators =====

	// contains - must contain substring
	validators["contains"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("contains validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !strings.Contains(s, tag) {
			return fmt.Errorf("must contain %s", tag)
		}
		return nil
	}

	// startswith - must start with prefix
	validators["startswith"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("startswith validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !strings.HasPrefix(s, tag) {
			return fmt.Errorf("must start with %s", tag)
		}
		return nil
	}

	// endswith - must end with suffix
	validators["endswith"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("endswith validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !strings.HasSuffix(s, tag) {
			return fmt.Errorf("must end with %s", tag)
		}
		return nil
	}

	// excludes - must not contain substring
	validators["excludes"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("excludes validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if strings.Contains(s, tag) {
			return fmt.Errorf("must not contain %s", tag)
		}
		return nil
	}

	// lowercase - must be all lowercase
	validators["lowercase"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("lowercase validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if s != strings.ToLower(s) {
			return fmt.Errorf("must be lowercase")
		}
		return nil
	}

	// uppercase - must be all uppercase
	validators["uppercase"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("uppercase validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if s != strings.ToUpper(s) {
			return fmt.Errorf("must be uppercase")
		}
		return nil
	}

	// ascii - ASCII characters only
	validators["ascii"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ascii validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if r > 127 {
				return fmt.Errorf("must contain only ASCII characters")
			}
		}
		return nil
	}

	// printascii - printable ASCII only
	validators["printascii"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("printascii validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		for _, r := range s {
			if r < 32 || r > 126 {
				return fmt.Errorf("must contain only printable ASCII characters")
			}
		}
		return nil
	}

	// boolean - parseable boolean string
	validators["boolean"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("boolean validator only supports strings")
		}
		s := strings.ToLower(value.String())
		if s == "" {
			return nil
		}
		valid := []string{"true", "false", "1", "0", "yes", "no", "on", "off"}
		for _, v := range valid {
			if s == v {
				return nil
			}
		}
		return fmt.Errorf("must be a valid boolean value")
	}

	// json - valid JSON string
	validators["json"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("json validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		var js any
		if err := json.Unmarshal([]byte(s), &js); err != nil {
			return fmt.Errorf("must be valid JSON")
		}
		return nil
	}

	// ===== Format / Network Validators =====

	// ip - valid IP address (v4 or v6)
	validators["ip"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ip validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if net.ParseIP(s) == nil {
			return fmt.Errorf("must be a valid IP address")
		}
		return nil
	}

	// ipv4 - valid IPv4 address
	validators["ipv4"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ipv4 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("must be a valid IPv4 address")
		}
		return nil
	}

	// ipv6 - valid IPv6 address
	validators["ipv6"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("ipv6 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("must be a valid IPv6 address")
		}
		return nil
	}

	// cidr - valid CIDR notation
	validators["cidr"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("cidr validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		_, _, err := net.ParseCIDR(s)
		if err != nil {
			return fmt.Errorf("must be valid CIDR notation")
		}
		return nil
	}

	// hostname - valid hostname (RFC 1123)
	validators["hostname"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("hostname validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !hostnameRegex.MatchString(s) || len(s) > 253 {
			return fmt.Errorf("must be a valid hostname")
		}
		return nil
	}

	// uri - valid URI (any scheme)
	validators["uri"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("uri validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("must be a valid URI")
		}
		return nil
	}

	// base64 - valid base64 string
	validators["base64"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("base64 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if _, err := base64.StdEncoding.DecodeString(s); err != nil {
			return fmt.Errorf("must be valid base64")
		}
		return nil
	}

	// hexadecimal - valid hex string
	validators["hexadecimal"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("hexadecimal validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}

		if !hexRegex.MatchString(s) {
			return fmt.Errorf("must be valid hexadecimal")
		}
		return nil
	}

	// hexcolor - valid hex color code
	validators["hexcolor"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("hexcolor validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !hexColorRegex.MatchString(s) {
			return fmt.Errorf("must be valid hex color code")
		}
		return nil
	}

	// e164 - E.164 phone number format
	validators["e164"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("e164 validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !e164Regex.MatchString(s) {
			return fmt.Errorf("must be valid E.164 phone number")
		}
		return nil
	}

	// semver - semantic version string
	validators["semver"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("semver validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		if !semverRegex.MatchString(s) {
			return fmt.Errorf("must be valid semantic version")
		}
		return nil
	}

	// jwt - valid JWT format (header.payload.signature)
	validators["jwt"] = func(value reflect.Value, tag string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("jwt validator only supports strings")
		}
		s := value.String()
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ".")
		if len(parts) != 3 {
			return fmt.Errorf("must be valid JWT format")
		}
		// Check each part is valid base64
		for _, part := range parts {
			if _, err := base64.RawURLEncoding.DecodeString(part); err != nil {
				return fmt.Errorf("must be valid JWT format")
			}
		}
		return nil
	}

	// ===== Collection Validators =====

	// unique - all elements must be unique
	validators["unique"] = func(value reflect.Value, tag string) error {
		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			seen := make(map[any]struct{})
			for i := 0; i < value.Len(); i++ {
				elem := value.Index(i).Interface()
				if _, exists := seen[elem]; exists {
					return fmt.Errorf("must have unique elements")
				}
				seen[elem] = struct{}{}
			}
			return nil
		case reflect.Map:
			// Maps always have unique keys by definition
			return nil
		default:
			return fmt.Errorf("unique not supported for type %s", value.Kind())
		}
	}

	// each - validate each element in collection (handled in validateField)
	validators["each"] = func(value reflect.Value, tag string) error {
		// each is a special marker that tells validateField to recurse into elements
		// The actual validation is handled in validateField
		return nil
	}

	// Store the populated map atomically
	v.validators.Store(validators)
}
