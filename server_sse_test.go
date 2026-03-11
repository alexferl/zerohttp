package zerohttp

import (
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestServer_SSEProvider(t *testing.T) {
	t.Run("SSEProvider returns nil by default", func(t *testing.T) {
		s := New()
		if s.SSEProvider() != nil {
			t.Error("expected nil SSEProvider by default")
		}
	})

	t.Run("SetSSEProvider stores provider", func(t *testing.T) {
		s := New()
		provider := NewDefaultProvider()
		s.SetSSEProvider(provider)

		if s.SSEProvider() != provider {
			t.Error("expected SSEProvider to be set")
		}
	})

	t.Run("SSEProvider works with config option", func(t *testing.T) {
		provider := NewDefaultProvider()
		s := New(config.Config{SSEProvider: provider})

		if s.SSEProvider() != provider {
			t.Error("expected SSEProvider from config option")
		}
	})
}
