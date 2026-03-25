package validator

import (
	"reflect"
	"sync"
	"testing"
)

// BenchmarkGetTypeInfo measures the performance of type info retrieval
func BenchmarkGetTypeInfo(b *testing.B) {
	// Clear cache
	Registry.cache = sync.Map{}

	typ := reflect.TypeOf(testCollectionStruct{})

	b.Run("Cold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Registry.cache = sync.Map{}
			_ = Registry.GetTypeInfo(typ)
		}
	})

	b.Run("Warm", func(b *testing.B) {
		// Pre-populate cache
		_ = Registry.GetTypeInfo(typ)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Registry.GetTypeInfo(typ)
		}
	})
}
