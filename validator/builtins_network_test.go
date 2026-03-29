package validator

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestIPValidator(t *testing.T) {
	type TestIP struct {
		IP string `validate:"ip"`
	}

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"empty", "", false},
		{"ipv4", "192.168.1.1", false},
		{"ipv4 broadcast", "255.255.255.255", false},
		{"ipv4 loopback", "127.0.0.1", false},
		{"ipv6 loopback", "::1", false},
		{"ipv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"ipv6 compressed", "2001:db8::1", false},
		{"invalid", "not-an-ip", true},
		{"invalid octet", "256.1.1.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestIP{IP: tt.ip})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestIPOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"ip"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestIPv4Validator(t *testing.T) {
	type TestIPv4 struct {
		IPv4 string `validate:"ipv4"`
	}

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"empty", "", false},
		{"standard", "192.168.1.1", false},
		{"broadcast", "255.255.255.255", false},
		{"zero", "0.0.0.0", false},
		{"loopback", "127.0.0.1", false},
		{"invalid octet", "256.1.1.1", true},
		{"negative octet", "-1.1.1.1", true},
		{"too few octets", "192.168.1", true},
		{"too many octets", "192.168.1.1.1", true},
		{"leading zero", "192.168.01.1", true},
		{"non-numeric", "a.b.c.d", true},
		{"spaces", "192. 168.1.1", true},
		{"ipv6 in ipv4 field", "::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestIPv4{IPv4: tt.ip})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestIPv4OnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"ipv4"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestIPv6Validator(t *testing.T) {
	type TestIPv6 struct {
		IPv6 string `validate:"ipv6"`
	}

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"empty", "", false},
		{"loopback compressed", "::1", false},
		{"full loopback", "0:0:0:0:0:0:0:1", false},
		{"unspecified", "::", false},
		{"full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"compressed", "2001:db8:85a3::8a2e:370:7334", false},
		{"uppercase", "2001:DB8::1", false},
		{"with zone", "fe80::1%eth0", true},
		{"ipv4 mapped", "::ffff:192.0.2.1", true},
		{"invalid char", "2001:db8::gggg", true},
		{"too many groups", "2001:db8::1:2:3:4:5:6:7", true},
		{"double :: twice", "2001::db8::1", true},
		{"truncated", "2001:db8:", true},
		{"spaces", "2001: db8::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestIPv6{IPv6: tt.ip})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestIPv6OnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"ipv6"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestCIDRValidator(t *testing.T) {
	type TestCIDR struct {
		CIDR string `validate:"cidr"`
	}

	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{"empty", "", false},
		{"ipv4 /24", "192.168.0.0/24", false},
		{"ipv4 /32", "192.168.1.1/32", false},
		{"ipv4 /0", "0.0.0.0/0", false},
		{"ipv6 /64", "2001:db8::/64", false},
		{"ipv6 /128", "2001:db8::1/128", false},
		{"invalid prefix", "192.168.0.0/33", true},
		{"missing slash", "192.168.0.0", true},
		{"slash only", "/24", true},
		{"non-numeric prefix", "192.168.0.0/abc", true},
		{"negative prefix", "192.168.0.0/-1", true},
		{"ipv6 invalid prefix", "2001:db8::/129", true},
		{"invalid ip", "invalid/24", true},
		{"spaces", "192.168.0.0 /24", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestCIDR{CIDR: tt.cidr})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestCIDROnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"cidr"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestHostnameValidator(t *testing.T) {
	type TestHostname struct {
		Hostname string `validate:"hostname"`
	}

	tests := []struct {
		name     string
		hostname string
		wantErr  bool
	}{
		{"empty", "", false},
		{"localhost", "localhost", false},
		{"single label", "example", false},
		{"two labels", "example.com", false},
		{"multiple labels", "deep.sub.domain.example.com", false},
		{"max label length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com", true},
		{"numeric TLD", "example.123", false},
		{"underscore", "_example.com", true},
		{"space", "example .com", true},
		{"dot at end", "example.com.", true},
		{"dot at start", ".example.com", true},
		{"consecutive dots", "example..com", true},
		{"hyphen at start", "-example.com", true},
		{"hyphen at end", "example-.com", true},
		{"valid hyphen", "example-host.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestHostname{Hostname: tt.hostname})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestHostnameOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"hostname"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestURIValidator(t *testing.T) {
	type TestURI struct {
		URI string `validate:"uri"`
	}

	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{"empty", "", false},
		{"http", "http://example.com", false},
		{"https with path", "https://example.com/path/to/resource", false},
		{"ftp", "ftp://ftp.example.com/file", false},
		{"relative path", "/path/to/resource", true},
		{"relative dot", "./relative/path", true},
		{"urn", "urn:isbn:0451450523", true},
		{"file", "file:///etc/passwd", true},
		{"mailto", "mailto:user@example.com", true},
		{"query only", "?key=value", true},
		{"fragment only", "#section", true},
		{"spaces", "/path with spaces", true},
		{"unicode", "/日本語", true},
		{"double slash without scheme", "//example.com/path", true},
		{"invalid", "://invalid-uri", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestURI{URI: tt.uri})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestURIOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"uri"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestURLValidator(t *testing.T) {
	type TestURL struct {
		Website string `validate:"url"`
	}

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty", "", false},
		{"http", "http://example.com", false},
		{"https", "https://example.com/path", false},
		{"http with port", "http://example.com:8080", false},
		{"https with port", "https://example.com:443/path", false},
		{"with query params", "https://example.com?foo=bar&baz=qux", false},
		{"with fragment", "https://example.com#section", false},
		{"with user info", "https://user:pass@example.com", false},
		{"subdomain", "https://sub.example.com", false},
		{"deep path", "https://example.com/a/b/c/d", false},
		{"ip address", "http://192.168.1.1", false},
		{"ip with port", "http://192.168.1.1:8080/path", false},
		{"scheme only", "https://", false},
		{"spaces", "https://example .com", false},
		{"unicode domain", "https://例え.jp", false},
		{"ftp", "ftp://example.com", true},
		{"file protocol", "file:///etc/passwd", true},
		{"mailto", "mailto:test@example.com", true},
		{"javascript", "javascript:alert(1)", true},
		{"relative path", "/path/to/resource", true},
		{"no scheme", "example.com", true},
		{"invalid", "://bad-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestURL{Website: tt.url}
			err := New().Struct(&input)
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestURLOnNonString(t *testing.T) {
	type TestURLInt struct {
		Value int `validate:"url"`
	}
	input := TestURLInt{Value: 123}
	err := New().Struct(&input)
	zhtest.AssertError(t, err)
}
