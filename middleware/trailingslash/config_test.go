package trailingslash

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestTrailingSlashConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, RedirectAction, cfg.Action)
	zhtest.AssertFalse(t, cfg.PreferTrailingSlash)
	zhtest.AssertEqual(t, http.StatusMovedPermanently, cfg.RedirectCode)

	// Test default redirect code specifically
	zhtest.AssertEqual(t, 301, cfg.RedirectCode)
	zhtest.AssertEqual(t, http.StatusMovedPermanently, cfg.RedirectCode)
}

func TestTrailingSlashConfig_ActionConstants(t *testing.T) {
	tests := []struct {
		action   Action
		expected string
	}{
		{RedirectAction, "redirect"},
		{StripAction, "strip"},
		{AppendAction, "append"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			zhtest.AssertEqual(t, tt.expected, string(tt.action))
		})
	}
}

func TestTrailingSlashConfig_StructAssignment(t *testing.T) {
	t.Run("action assignment", func(t *testing.T) {
		cfg := Config{
			Action: StripAction,
		}
		zhtest.AssertEqual(t, StripAction, cfg.Action)
	})

	t.Run("preference assignment", func(t *testing.T) {
		cfg := Config{
			PreferTrailingSlash: true,
		}
		zhtest.AssertTrue(t, cfg.PreferTrailingSlash)
		// Test setting back to false
		cfg.PreferTrailingSlash = false
		zhtest.AssertFalse(t, cfg.PreferTrailingSlash)
	})

	t.Run("redirect code assignment", func(t *testing.T) {
		cfg := Config{
			RedirectCode: http.StatusFound,
		}
		zhtest.AssertEqual(t, http.StatusFound, cfg.RedirectCode)
	})

	t.Run("multiple fields", func(t *testing.T) {
		cfg := Config{
			Action:              AppendAction,
			PreferTrailingSlash: true,
			RedirectCode:        http.StatusFound,
		}

		zhtest.AssertEqual(t, AppendAction, cfg.Action)
		zhtest.AssertTrue(t, cfg.PreferTrailingSlash)
		zhtest.AssertEqual(t, http.StatusFound, cfg.RedirectCode)
	})
}

func TestTrailingSlashConfig_AllActions(t *testing.T) {
	actions := []Action{RedirectAction, StripAction, AppendAction}
	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			cfg := Config{
				Action: action,
			}
			zhtest.AssertEqual(t, action, cfg.Action)
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
			cfg := Config{
				RedirectCode: tt.redirectCode,
			}
			zhtest.AssertEqual(t, tt.httpConst, cfg.RedirectCode)
			zhtest.AssertEqual(t, tt.redirectCode, cfg.RedirectCode)
		})
	}
}

func TestTrailingSlashConfig_UsageScenarios(t *testing.T) {
	tests := []struct {
		name                string
		preferTrailingSlash bool
		action              Action
		description         string
	}{
		{"API style - no trailing slash", false, RedirectAction, "typical API configuration"},
		{"website style - with trailing slash", true, RedirectAction, "typical website configuration"},
		{"flexible API - strip trailing slash", false, StripAction, "API that accepts both forms"},
		{"flexible website - append trailing slash", true, AppendAction, "website that accepts both forms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				PreferTrailingSlash: tt.preferTrailingSlash,
				Action:              tt.action,
			}

			zhtest.AssertEqual(t, tt.preferTrailingSlash, cfg.PreferTrailingSlash)
			zhtest.AssertEqual(t, tt.action, cfg.Action)
		})
	}
}

func TestTrailingSlashConfig_EdgeCases(t *testing.T) {
	t.Run("boolean toggling", func(t *testing.T) {
		cfg := DefaultConfig
		// Start with default (false)
		zhtest.AssertFalse(t, cfg.PreferTrailingSlash)
		// Toggle to true
		cfg.PreferTrailingSlash = true
		zhtest.AssertTrue(t, cfg.PreferTrailingSlash)
		// Toggle back to false
		cfg.PreferTrailingSlash = false
		zhtest.AssertFalse(t, cfg.PreferTrailingSlash)
	})

	t.Run("default behavior validation", func(t *testing.T) {
		cfg := DefaultConfig
		// Should redirect by default
		zhtest.AssertEqual(t, RedirectAction, cfg.Action)
		// Should prefer no trailing slash (API style)
		zhtest.AssertFalse(t, cfg.PreferTrailingSlash)
		// Should use permanent redirect
		zhtest.AssertEqual(t, http.StatusMovedPermanently, cfg.RedirectCode)
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		zhtest.AssertEqual(t, Action(""), cfg.Action)
		zhtest.AssertFalse(t, cfg.PreferTrailingSlash)
		zhtest.AssertEqual(t, 0, cfg.RedirectCode)
	})
}
