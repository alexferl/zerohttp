package config

import (
	"reflect"
	"testing"
)

func TestContentCharsetConfig_DefaultValues(t *testing.T) {
	cfg := DefaultContentCharsetConfig
	if len(cfg.Charsets) != 2 {
		t.Errorf("expected 2 default charsets, got %d", len(cfg.Charsets))
	}
	expectedCharsets := []string{"utf-8", ""}
	if !reflect.DeepEqual(cfg.Charsets, expectedCharsets) {
		t.Errorf("expected default charsets = %v, got %v", expectedCharsets, cfg.Charsets)
	}
	if cfg.Charsets[0] != "utf-8" {
		t.Errorf("expected first charset = 'utf-8', got %s", cfg.Charsets[0])
	}
	if cfg.Charsets[1] != "" {
		t.Errorf("expected second charset = '', got %s", cfg.Charsets[1])
	}
}

func TestWithContentCharsetCharsetsOption(t *testing.T) {
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
			cfg := DefaultContentCharsetConfig
			WithContentCharsetCharsets(tt.input)(&cfg)
			if len(cfg.Charsets) != len(tt.expected) {
				t.Errorf("expected %d charsets, got %d", len(tt.expected), len(cfg.Charsets))
			}
			if !reflect.DeepEqual(cfg.Charsets, tt.expected) {
				t.Errorf("expected charsets = %v, got %v", tt.expected, cfg.Charsets)
			}
		})
	}
}

func TestContentCharsetConfig_EdgeCases(t *testing.T) {
	t.Run("empty charsets", func(t *testing.T) {
		cfg := DefaultContentCharsetConfig
		WithContentCharsetCharsets([]string{})(&cfg)
		if cfg.Charsets == nil {
			t.Error("expected charsets slice to be initialized, not nil")
		}
		if len(cfg.Charsets) != 0 {
			t.Errorf("expected empty charsets slice, got %d entries", len(cfg.Charsets))
		}
	})

	t.Run("nil charsets", func(t *testing.T) {
		cfg := DefaultContentCharsetConfig
		WithContentCharsetCharsets(nil)(&cfg)
		if cfg.Charsets != nil {
			t.Error("expected charsets to remain nil when nil is passed")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		charsets := []string{"UTF-8", "utf-8", "Utf-8"}
		cfg := DefaultContentCharsetConfig
		WithContentCharsetCharsets(charsets)(&cfg)
		if len(cfg.Charsets) != 3 {
			t.Errorf("expected 3 charsets, got %d", len(cfg.Charsets))
		}
		for i, expectedCharset := range charsets {
			if cfg.Charsets[i] != expectedCharset {
				t.Errorf("expected charset[%d] = %s, got %s", i, expectedCharset, cfg.Charsets[i])
			}
		}
	})

	t.Run("duplicate charsets", func(t *testing.T) {
		charsets := []string{"utf-8", "utf-8", "iso-8859-1", "utf-8"}
		cfg := DefaultContentCharsetConfig
		WithContentCharsetCharsets(charsets)(&cfg)
		if len(cfg.Charsets) != 4 {
			t.Errorf("expected 4 charsets (including duplicates), got %d", len(cfg.Charsets))
		}
		for i, expectedCharset := range charsets {
			if cfg.Charsets[i] != expectedCharset {
				t.Errorf("expected charset[%d] = %s, got %s", i, expectedCharset, cfg.Charsets[i])
			}
		}
	})

	t.Run("mixed empty and non-empty", func(t *testing.T) {
		charsets := []string{"", "utf-8", "", "iso-8859-1", ""}
		cfg := DefaultContentCharsetConfig
		WithContentCharsetCharsets(charsets)(&cfg)
		if len(cfg.Charsets) != 5 {
			t.Errorf("expected 5 charsets, got %d", len(cfg.Charsets))
		}
		for i, expectedCharset := range charsets {
			if cfg.Charsets[i] != expectedCharset {
				t.Errorf("expected charset[%d] = %q, got %q", i, expectedCharset, cfg.Charsets[i])
			}
		}
	})

	t.Run("whitespace charsets", func(t *testing.T) {
		charsets := []string{"utf-8", " ", "\t", "\n", "iso-8859-1"}
		cfg := DefaultContentCharsetConfig
		WithContentCharsetCharsets(charsets)(&cfg)
		if len(cfg.Charsets) != 5 {
			t.Errorf("expected 5 charsets, got %d", len(cfg.Charsets))
		}
		for i, expectedCharset := range charsets {
			if cfg.Charsets[i] != expectedCharset {
				t.Errorf("expected charset[%d] = %q, got %q", i, expectedCharset, cfg.Charsets[i])
			}
		}
	})
}
