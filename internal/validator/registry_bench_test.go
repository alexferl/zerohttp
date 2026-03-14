package validator

import (
	"reflect"
	"sync"
	"testing"
)

// BenchmarkGetTypeInfo measures the performance of type info retrieval
func BenchmarkGetTypeInfo(b *testing.B) {
	// Clear cache
	ValidatorRegistry.cache = sync.Map{}

	typ := reflect.TypeOf(testCollectionStruct{})

	b.Run("Cold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ValidatorRegistry.cache = sync.Map{}
			_ = ValidatorRegistry.GetTypeInfo(typ)
		}
	})

	b.Run("Warm", func(b *testing.B) {
		// Pre-populate cache
		_ = ValidatorRegistry.GetTypeInfo(typ)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ValidatorRegistry.GetTypeInfo(typ)
		}
	})
}
