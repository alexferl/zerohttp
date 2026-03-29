package zerohttp

import (
	"testing"

	"github.com/alexferl/zerohttp/sse"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestServer_SSEProvider(t *testing.T) {
	t.Run("SSEProvider returns nil by default", func(t *testing.T) {
		s := New()
		zhtest.AssertNil(t, s.SSEProvider())
	})

	t.Run("SetSSEProvider stores provider", func(t *testing.T) {
		s := New()
		provider := sse.NewDefaultProvider()
		s.SetSSEProvider(provider)

		zhtest.AssertEqual(t, provider, s.SSEProvider())
	})

	t.Run("SSEProvider works with config option", func(t *testing.T) {
		provider := sse.NewDefaultProvider()
		s := New(Config{Extensions: ExtensionsConfig{SSEProvider: provider}})

		zhtest.AssertEqual(t, provider, s.SSEProvider())
	})
}
