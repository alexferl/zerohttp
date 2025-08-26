package config

import (
	"net/http"
	"testing"
)

func TestTrailingSlashConfig_DefaultValues(t *testing.T) {
	cfg := DefaultTrailingSlashConfig
	if cfg.Action != RedirectAction {
		t.Errorf("expected default action = %s, got %s", RedirectAction, cfg.Action)
	}
	if cfg.PreferTrailingSlash != false {
		t.Errorf("expected default prefer trailing slash = false, got %t", cfg.PreferTrailingSlash)
	}
	if cfg.RedirectCode != http.StatusMovedPermanently {
		t.Errorf("expected default redirect code = %d, got %d", http.StatusMovedPermanently, cfg.RedirectCode)
	}

	// Test default redirect code specifically
	expectedCode := 301 // Moved Permanently
	if cfg.RedirectCode != expectedCode {
		t.Errorf("expected default redirect code = %d (Moved Permanently), got %d", expectedCode, cfg.RedirectCode)
	}
	if cfg.RedirectCode != http.StatusMovedPermanently {
		t.Errorf("expected redirect code to equal http.StatusMovedPermanently (%d), got %d", http.StatusMovedPermanently, cfg.RedirectCode)
	}
}

func TestTrailingSlashConfig_ActionConstants(t *testing.T) {
	tests := []struct {
		action   TrailingSlashAction
		expected string
	}{
		{RedirectAction, "redirect"},
		{StripAction, "strip"},
		{AppendAction, "append"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if string(tt.action) != tt.expected {
				t.Errorf("expected action string = %s, got %s", tt.expected, string(tt.action))
			}
		})
	}
}

func TestTrailingSlashOptions(t *testing.T) {
	t.Run("action option", func(t *testing.T) {
		cfg := DefaultTrailingSlashConfig
		WithTrailingSlashAction(StripAction)(&cfg)
		if cfg.Action != StripAction {
			t.Errorf("expected action = %s, got %s", StripAction, cfg.Action)
		}
	})

	t.Run("preference option", func(t *testing.T) {
		cfg := DefaultTrailingSlashConfig
		WithTrailingSlashPreference(true)(&cfg)
		if cfg.PreferTrailingSlash != true {
			t.Errorf("expected prefer trailing slash = true, got %t", cfg.PreferTrailingSlash)
		}
		// Test setting back to false
		WithTrailingSlashPreference(false)(&cfg)
		if cfg.PreferTrailingSlash != false {
			t.Errorf("expected prefer trailing slash = false, got %t", cfg.PreferTrailingSlash)
		}
	})

	t.Run("redirect code option", func(t *testing.T) {
		cfg := DefaultTrailingSlashConfig
		WithTrailingSlashRedirectCode(http.StatusFound)(&cfg)
		if cfg.RedirectCode != http.StatusFound {
			t.Errorf("expected redirect code = %d, got %d", http.StatusFound, cfg.RedirectCode)
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		cfg := DefaultTrailingSlashConfig
		WithTrailingSlashAction(AppendAction)(&cfg)
		WithTrailingSlashPreference(true)(&cfg)
		WithTrailingSlashRedirectCode(http.StatusFound)(&cfg)

		if cfg.Action != AppendAction {
			t.Errorf("expected action = %s, got %s", AppendAction, cfg.Action)
		}
		if cfg.PreferTrailingSlash != true {
			t.Errorf("expected prefer trailing slash = true, got %t", cfg.PreferTrailingSlash)
		}
		if cfg.RedirectCode != http.StatusFound {
			t.Errorf("expected redirect code = %d, got %d", http.StatusFound, cfg.RedirectCode)
		}
	})
}

func TestTrailingSlashConfig_AllActions(t *testing.T) {
	actions := []TrailingSlashAction{RedirectAction, StripAction, AppendAction}
	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			cfg := DefaultTrailingSlashConfig
			WithTrailingSlashAction(action)(&cfg)
			if cfg.Action != action {
				t.Errorf("expected action = %s, got %s", action, cfg.Action)
			}
		})
	}
}

func TestTrailingSlashConfig_RedirectCodes(t *testing.T) {
	tests := []struct {
		name         string
		redirectCode int
		httpConst    int
		description  string
	}{
		{"Moved Permanently", http.StatusMovedPermanently, 301, "permanent redirect"},
		{"Found", http.StatusFound, 302, "temporary redirect"},
		{"See Other", http.StatusSeeOther, 303, "see other resource"},
		{"Temporary Redirect", http.StatusTemporaryRedirect, 307, "temporary redirect preserving method"},
		{"Permanent Redirect", http.StatusPermanentRedirect, 308, "permanent redirect preserving method"},
		{"Custom code", 999, 999, "custom code"},
		{"Zero code", 0, 0, "no redirect code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultTrailingSlashConfig
			WithTrailingSlashRedirectCode(tt.redirectCode)(&cfg)
			if cfg.RedirectCode != tt.httpConst {
				t.Errorf("expected %s redirect code = %d, got %d", tt.description, tt.httpConst, cfg.RedirectCode)
			}
			if cfg.RedirectCode != tt.redirectCode {
				t.Errorf("expected redirect code to match constant %d, got %d", tt.redirectCode, cfg.RedirectCode)
			}
		})
	}
}

func TestTrailingSlashConfig_UsageScenarios(t *testing.T) {
	tests := []struct {
		name                string
		preferTrailingSlash bool
		action              TrailingSlashAction
		description         string
	}{
		{"API style - no trailing slash", false, RedirectAction, "typical API configuration"},
		{"website style - with trailing slash", true, RedirectAction, "typical website configuration"},
		{"flexible API - strip trailing slash", false, StripAction, "API that accepts both forms"},
		{"flexible website - append trailing slash", true, AppendAction, "website that accepts both forms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultTrailingSlashConfig
			WithTrailingSlashPreference(tt.preferTrailingSlash)(&cfg)
			WithTrailingSlashAction(tt.action)(&cfg)

			if cfg.PreferTrailingSlash != tt.preferTrailingSlash {
				t.Errorf("%s: expected prefer trailing slash = %t, got %t", tt.description, tt.preferTrailingSlash, cfg.PreferTrailingSlash)
			}
			if cfg.Action != tt.action {
				t.Errorf("%s: expected action = %s, got %s", tt.description, tt.action, cfg.Action)
			}
		})
	}
}

func TestTrailingSlashConfig_EdgeCases(t *testing.T) {
	t.Run("boolean toggling", func(t *testing.T) {
		cfg := DefaultTrailingSlashConfig
		// Start with default (false)
		if cfg.PreferTrailingSlash != false {
			t.Error("expected initial PreferTrailingSlash = false")
		}
		// Toggle to true
		WithTrailingSlashPreference(true)(&cfg)
		if cfg.PreferTrailingSlash != true {
			t.Error("expected PreferTrailingSlash = true after toggle")
		}
		// Toggle back to false
		WithTrailingSlashPreference(false)(&cfg)
		if cfg.PreferTrailingSlash != false {
			t.Error("expected PreferTrailingSlash = false after toggle back")
		}
	})

	t.Run("default behavior validation", func(t *testing.T) {
		cfg := DefaultTrailingSlashConfig
		// Should redirect by default
		if cfg.Action != RedirectAction {
			t.Error("expected default to redirect")
		}
		// Should prefer no trailing slash (API style)
		if cfg.PreferTrailingSlash != false {
			t.Error("expected default to prefer no trailing slash")
		}
		// Should use permanent redirect
		if cfg.RedirectCode != http.StatusMovedPermanently {
			t.Error("expected default to use permanent redirect")
		}
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := TrailingSlashConfig{} // Zero values
		if cfg.Action != "" {
			t.Errorf("expected zero action = '', got %s", cfg.Action)
		}
		if cfg.PreferTrailingSlash != false {
			t.Errorf("expected zero prefer trailing slash = false, got %t", cfg.PreferTrailingSlash)
		}
		if cfg.RedirectCode != 0 {
			t.Errorf("expected zero redirect code = 0, got %d", cfg.RedirectCode)
		}
	})
}
