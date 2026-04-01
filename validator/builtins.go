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

var booleanValues = map[string]struct{}{
	"true": {}, "false": {}, "1": {}, "0": {},
	"yes": {}, "no": {}, "on": {}, "off": {},
}

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
			target, err := parseIntParam(tag, "min")
			if err != nil {
				return err
			}
			if value.Int() < target {
				return fmt.Errorf("must be at least %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := parseUintParam(tag, "min")
			if err != nil {
				return err
			}
			if value.Uint() < target {
				return fmt.Errorf("must be at least %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := parseFloatParam(tag, "min")
			if err != nil {
				return err
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
			target, err := parseIntParam(tag, "max")
			if err != nil {
				return err
			}
			if value.Int() > target {
				return fmt.Errorf("must be at most %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := parseUintParam(tag, "max")
			if err != nil {
				return err
			}
			if value.Uint() > target {
				return fmt.Errorf("must be at most %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := parseFloatParam(tag, "max")
			if err != nil {
				return err
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
		email, empty, err := stringVal(value, "email")
		if err != nil || empty {
			return err
		}
		_, err = mail.ParseAddress(email)
		if err != nil {
			return fmt.Errorf("invalid email format")
		}
		return nil
	}

	// url - validates URL format
	validators["url"] = func(value reflect.Value, tag string) error {
		urlStr, empty, err := stringVal(value, "url")
		if err != nil || empty {
			return err
		}
		// Simple URL validation
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			return fmt.Errorf("invalid URL format")
		}
		return nil
	}

	// alpha - only alphabetic characters
	validators["alpha"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "alpha")
		if err != nil || empty {
			return err
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
		s, empty, err := stringVal(value, "alphanum")
		if err != nil || empty {
			return err
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
		s, empty, err := stringVal(value, "numeric")
		if err != nil || empty {
			return err
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

	// anyof - for slices/arrays, at least one element must match one of the specified options
	validators["anyof"] = func(value reflect.Value, tag string) error {
		options := strings.Fields(tag)
		if len(options) == 0 {
			return nil
		}

		if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
			return fmt.Errorf("anyof only supports slices and arrays")
		}

		if value.Len() == 0 {
			return fmt.Errorf("must have at least one element matching: %s", tag)
		}

		// Determine the element kind
		elemKind := value.Type().Elem().Kind()

		switch elemKind {
		case reflect.String:
			for i := 0; i < value.Len(); i++ {
				s := value.Index(i).String()
				for _, opt := range options {
					if s == opt {
						return nil
					}
				}
			}
			return fmt.Errorf("must have at least one element matching: %s", tag)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			for i := 0; i < value.Len(); i++ {
				n := value.Index(i).Int()
				for _, opt := range options {
					if i, err := strconv.ParseInt(opt, 10, 64); err == nil && n == i {
						return nil
					}
				}
			}
			return fmt.Errorf("must have at least one element matching: %s", tag)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			for i := 0; i < value.Len(); i++ {
				n := value.Index(i).Uint()
				for _, opt := range options {
					if i, err := strconv.ParseUint(opt, 10, 64); err == nil && n == i {
						return nil
					}
				}
			}
			return fmt.Errorf("must have at least one element matching: %s", tag)
		default:
			return fmt.Errorf("anyof not supported for element type %s", elemKind)
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
			n, err := parseIntParam(tag, "eq")
			if err != nil {
				return err
			}
			if value.Int() != n {
				return fmt.Errorf("must be equal to %d", n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := parseUintParam(tag, "eq")
			if err != nil {
				return err
			}
			if value.Uint() != n {
				return fmt.Errorf("must be equal to %d", n)
			}
		case reflect.Float32, reflect.Float64:
			n, err := parseFloatParam(tag, "eq")
			if err != nil {
				return err
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
			n, err := parseIntParam(tag, "ne")
			if err != nil {
				return err
			}
			if value.Int() == n {
				return fmt.Errorf("must not be equal to %d", n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := parseUintParam(tag, "ne")
			if err != nil {
				return err
			}
			if value.Uint() == n {
				return fmt.Errorf("must not be equal to %d", n)
			}
		case reflect.Float32, reflect.Float64:
			n, err := parseFloatParam(tag, "ne")
			if err != nil {
				return err
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
			target, err := parseIntParam(tag, "gt")
			if err != nil {
				return err
			}
			if value.Int() <= target {
				return fmt.Errorf("must be greater than %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := parseUintParam(tag, "gt")
			if err != nil {
				return err
			}
			if value.Uint() <= target {
				return fmt.Errorf("must be greater than %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := parseFloatParam(tag, "gt")
			if err != nil {
				return err
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
			target, err := parseIntParam(tag, "lt")
			if err != nil {
				return err
			}
			if value.Int() >= target {
				return fmt.Errorf("must be less than %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := parseUintParam(tag, "lt")
			if err != nil {
				return err
			}
			if value.Uint() >= target {
				return fmt.Errorf("must be less than %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := parseFloatParam(tag, "lt")
			if err != nil {
				return err
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
			target, err := parseIntParam(tag, "gte")
			if err != nil {
				return err
			}
			if value.Int() < target {
				return fmt.Errorf("must be greater than or equal to %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := parseUintParam(tag, "gte")
			if err != nil {
				return err
			}
			if value.Uint() < target {
				return fmt.Errorf("must be greater than or equal to %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := parseFloatParam(tag, "gte")
			if err != nil {
				return err
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
			target, err := parseIntParam(tag, "lte")
			if err != nil {
				return err
			}
			if value.Int() > target {
				return fmt.Errorf("must be less than or equal to %d", target)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			target, err := parseUintParam(tag, "lte")
			if err != nil {
				return err
			}
			if value.Uint() > target {
				return fmt.Errorf("must be less than or equal to %d", target)
			}
		case reflect.Float32, reflect.Float64:
			target, err := parseFloatParam(tag, "lte")
			if err != nil {
				return err
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
		uuid, empty, err := stringVal(value, "uuid")
		if err != nil || empty {
			return err
		}
		if !uuidRegex.MatchString(uuid) {
			return fmt.Errorf("invalid UUID format")
		}
		return nil
	}

	// datetime - validates datetime format
	validators["datetime"] = func(value reflect.Value, tag string) error {
		dt, empty, err := stringVal(value, "datetime")
		if err != nil || empty {
			return err
		}
		// Default to RFC3339 if no format specified
		format := tag
		if format == "" {
			format = time.RFC3339
		}
		_, err = time.Parse(format, dt)
		if err != nil {
			return fmt.Errorf("invalid datetime format, expected %s", format)
		}
		return nil
	}

	// ===== String Content Validators =====

	// contains - must contain substring
	validators["contains"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "contains")
		if err != nil || empty {
			return err
		}
		if !strings.Contains(s, tag) {
			return fmt.Errorf("must contain %s", tag)
		}
		return nil
	}

	// startswith - must start with prefix
	validators["startswith"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "startswith")
		if err != nil || empty {
			return err
		}
		if !strings.HasPrefix(s, tag) {
			return fmt.Errorf("must start with %s", tag)
		}
		return nil
	}

	// endswith - must end with suffix
	validators["endswith"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "endswith")
		if err != nil || empty {
			return err
		}
		if !strings.HasSuffix(s, tag) {
			return fmt.Errorf("must end with %s", tag)
		}
		return nil
	}

	// excludes - must not contain substring
	validators["excludes"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "excludes")
		if err != nil || empty {
			return err
		}
		if strings.Contains(s, tag) {
			return fmt.Errorf("must not contain %s", tag)
		}
		return nil
	}

	// lowercase - must be all lowercase
	validators["lowercase"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "lowercase")
		if err != nil || empty {
			return err
		}
		if s != strings.ToLower(s) {
			return fmt.Errorf("must be lowercase")
		}
		return nil
	}

	// uppercase - must be all uppercase
	validators["uppercase"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "uppercase")
		if err != nil || empty {
			return err
		}
		if s != strings.ToUpper(s) {
			return fmt.Errorf("must be uppercase")
		}
		return nil
	}

	// ascii - ASCII characters only
	validators["ascii"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "ascii")
		if err != nil || empty {
			return err
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
		s, empty, err := stringVal(value, "printascii")
		if err != nil || empty {
			return err
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
		s, empty, err := stringVal(value, "boolean")
		if err != nil || empty {
			return err
		}
		if _, ok := booleanValues[strings.ToLower(s)]; !ok {
			return fmt.Errorf("must be a valid boolean value")
		}
		return nil
	}

	// json - valid JSON string
	validators["json"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "json")
		if err != nil || empty {
			return err
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
		s, empty, err := stringVal(value, "ip")
		if err != nil || empty {
			return err
		}
		if net.ParseIP(s) == nil {
			return fmt.Errorf("must be a valid IP address")
		}
		return nil
	}

	// ipv4 - valid IPv4 address
	validators["ipv4"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "ipv4")
		if err != nil || empty {
			return err
		}
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("must be a valid IPv4 address")
		}
		return nil
	}

	// ipv6 - valid IPv6 address
	validators["ipv6"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "ipv6")
		if err != nil || empty {
			return err
		}
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("must be a valid IPv6 address")
		}
		return nil
	}

	// cidr - valid CIDR notation
	validators["cidr"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "cidr")
		if err != nil || empty {
			return err
		}
		_, _, err = net.ParseCIDR(s)
		if err != nil {
			return fmt.Errorf("must be valid CIDR notation")
		}
		return nil
	}

	// hostname - valid hostname (RFC 1123)
	validators["hostname"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "hostname")
		if err != nil || empty {
			return err
		}
		if !hostnameRegex.MatchString(s) || len(s) > 253 {
			return fmt.Errorf("must be a valid hostname")
		}
		return nil
	}

	// uri - valid URI (any scheme)
	validators["uri"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "uri")
		if err != nil || empty {
			return err
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("must be a valid URI")
		}
		return nil
	}

	// base64 - valid base64 string
	validators["base64"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "base64")
		if err != nil || empty {
			return err
		}
		if _, err := base64.StdEncoding.DecodeString(s); err != nil {
			return fmt.Errorf("must be valid base64")
		}
		return nil
	}

	// hexadecimal - valid hex string
	validators["hexadecimal"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "hexadecimal")
		if err != nil || empty {
			return err
		}
		if !hexRegex.MatchString(s) {
			return fmt.Errorf("must be valid hexadecimal")
		}
		return nil
	}

	// hexcolor - valid hex color code
	validators["hexcolor"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "hexcolor")
		if err != nil || empty {
			return err
		}
		if !hexColorRegex.MatchString(s) {
			return fmt.Errorf("must be valid hex color code")
		}
		return nil
	}

	// e164 - E.164 phone number format
	validators["e164"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "e164")
		if err != nil || empty {
			return err
		}
		if !e164Regex.MatchString(s) {
			return fmt.Errorf("must be valid E.164 phone number")
		}
		return nil
	}

	// semver - semantic version string
	validators["semver"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "semver")
		if err != nil || empty {
			return err
		}
		if !semverRegex.MatchString(s) {
			return fmt.Errorf("must be valid semantic version")
		}
		return nil
	}

	// jwt - valid JWT format (header.payload.signature)
	validators["jwt"] = func(value reflect.Value, tag string) error {
		s, empty, err := stringVal(value, "jwt")
		if err != nil || empty {
			return err
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

	// Store the populated map atomically
	v.validators.Store(validators)
}

func stringVal(value reflect.Value, name string) (string, bool, error) {
	if value.Kind() != reflect.String {
		return "", false, fmt.Errorf("%s validator only supports strings", name)
	}
	s := value.String()
	return s, s == "", nil
}

func parseIntParam(tag, name string) (int64, error) {
	n, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter: %s", name, tag)
	}
	return n, nil
}

func parseUintParam(tag, name string) (uint64, error) {
	n, err := strconv.ParseUint(tag, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter: %s", name, tag)
	}
	return n, nil
}

func parseFloatParam(tag, name string) (float64, error) {
	n, err := strconv.ParseFloat(tag, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter: %s", name, tag)
	}
	return n, nil
}
