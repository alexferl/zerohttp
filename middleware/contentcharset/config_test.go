package contentcharset

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestContentCharsetConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 2, len(cfg.Charsets))
	expectedCharsets := []string{"utf-8", ""}
	zhtest.AssertDeepEqual(t, expectedCharsets, cfg.Charsets)
	zhtest.AssertEqual(t, "utf-8", cfg.Charsets[0])
	zhtest.AssertEqual(t, "", cfg.Charsets[1])
}

func TestContentCharsetConfig_StructAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"multiple charsets", []string{"utf-8", "iso-8859-1", "windows-1252"}, []string{"utf-8", "iso-8859-1", "windows-1252"}},
		{"UTF-8 only", []string{"utf-8"}, []string{"utf-8"}},
		{"empty charset only", []string{""}, []string{""}},
		{"multiple with special chars", []string{"utf-8", "iso-8859-1", "windows-1252", "shift_jis", "euc-kr", "gb2312"}, []string{"utf-8", "iso-8859-1", "windows-1252", "shift_jis", "euc-kr", "gb2312"}},
		{"common charsets", []string{"utf-8", "iso-8859-1", "windows-1252", "ascii", "gb2312", "big5", "shift_jis", "euc-jp", "koi8-r", ""}, []string{"utf-8", "iso-8859-1", "windows-1252", "ascii", "gb2312", "big5", "shift_jis", "euc-jp", "koi8-r", ""}},
		{"long charset names", []string{"very-long-charset-name-that-might-not-exist", "another_extremely_long_charset_encoding_name", "utf-8"}, []string{"very-long-charset-name-that-might-not-exist", "another_extremely_long_charset_encoding_name", "utf-8"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Charsets: tt.input,
			}
			zhtest.AssertEqual(t, len(tt.expected), len(cfg.Charsets))
			zhtest.AssertDeepEqual(t, tt.expected, cfg.Charsets)
		})
	}
}

func TestContentCharsetConfig_EdgeCases(t *testing.T) {
	t.Run("empty charsets", func(t *testing.T) {
		cfg := Config{
			Charsets: []string{},
		}
		zhtest.AssertNotNil(t, cfg.Charsets)
		zhtest.AssertEqual(t, 0, len(cfg.Charsets))
	})

	t.Run("nil charsets", func(t *testing.T) {
		cfg := Config{
			Charsets: nil,
		}
		zhtest.AssertNil(t, cfg.Charsets)
	})

	t.Run("case sensitivity", func(t *testing.T) {
		charsets := []string{"UTF-8", "utf-8", "Utf-8"}
		cfg := Config{
			Charsets: charsets,
		}
		zhtest.AssertEqual(t, 3, len(cfg.Charsets))
		for i, expectedCharset := range charsets {
			zhtest.AssertEqual(t, expectedCharset, cfg.Charsets[i])
		}
	})

	t.Run("duplicate charsets", func(t *testing.T) {
		charsets := []string{"utf-8", "utf-8", "iso-8859-1", "utf-8"}
		cfg := Config{
			Charsets: charsets,
		}
		zhtest.AssertEqual(t, 4, len(cfg.Charsets))
		for i, expectedCharset := range charsets {
			zhtest.AssertEqual(t, expectedCharset, cfg.Charsets[i])
		}
	})

	t.Run("mixed empty and non-empty", func(t *testing.T) {
		charsets := []string{"", "utf-8", "", "iso-8859-1", ""}
		cfg := Config{
			Charsets: charsets,
		}
		zhtest.AssertEqual(t, 5, len(cfg.Charsets))
		for i, expectedCharset := range charsets {
			zhtest.AssertEqual(t, expectedCharset, cfg.Charsets[i])
		}
	})

	t.Run("whitespace charsets", func(t *testing.T) {
		charsets := []string{"utf-8", " ", "\t", "\n", "iso-8859-1"}
		cfg := Config{
			Charsets: charsets,
		}
		zhtest.AssertEqual(t, 5, len(cfg.Charsets))
		for i, expectedCharset := range charsets {
			zhtest.AssertEqual(t, expectedCharset, cfg.Charsets[i])
		}
	})
}
