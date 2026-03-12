package rwutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkResponseWriter_WriteHeader measures status code capture overhead
func BenchmarkResponseWriter_WriteHeader(b *testing.B) {
	b.Run("RawResponseWriter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			w.WriteHeader(http.StatusOK)
		}
	})

	b.Run("WrappedResponseWriter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			rw.WriteHeader(http.StatusOK)
		}
	})
}

// BenchmarkResponseWriter_WriteHeader_DifferentStatuses measures different status codes
func BenchmarkResponseWriter_WriteHeader_DifferentStatuses(b *testing.B) {
	statuses := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, status := range statuses {
		b.Run(fmt.Sprintf("Status%d", status), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				rw := NewResponseWriter(w)
				rw.WriteHeader(status)
			}
		})
	}
}

// BenchmarkResponseWriter_Write measures Write performance
func BenchmarkResponseWriter_Write(b *testing.B) {
	data := []byte("Hello, World!")

	b.Run("RawResponseWriter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_, _ = w.Write(data)
		}
	})

	b.Run("WrappedResponseWriter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			_, _ = rw.Write(data)
		}
	})

	b.Run("WrappedWithHeaderWritten", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write(data)
		}
	})
}

// BenchmarkResponseWriter_Write_DifferentSizes measures Write with different payload sizes
func BenchmarkResponseWriter_Write_DifferentSizes(b *testing.B) {
	sizes := []int{100, 1024, 10240, 102400}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := make([]byte, size)

			b.Run("Raw", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					w := httptest.NewRecorder()
					_, _ = w.Write(data)
				}
			})

			b.Run("Wrapped", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					w := httptest.NewRecorder()
					rw := NewResponseWriter(w)
					_, _ = rw.Write(data)
				}
			})
		})
	}
}

// BenchmarkResponseWriter_StatusCode measures StatusCode() call overhead
func BenchmarkResponseWriter_StatusCode(b *testing.B) {
	b.Run("AfterWriteHeader", func(b *testing.B) {
		w := httptest.NewRecorder()
		rw := NewResponseWriter(w)
		rw.WriteHeader(http.StatusCreated)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = rw.StatusCode()
		}
	})

	b.Run("Default", func(b *testing.B) {
		w := httptest.NewRecorder()
		rw := NewResponseWriter(w)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = rw.StatusCode()
		}
	})
}

// BenchmarkResponseWriter_HeaderWritten measures HeaderWritten() call overhead
func BenchmarkResponseWriter_HeaderWritten(b *testing.B) {
	b.Run("AfterWriteHeader", func(b *testing.B) {
		w := httptest.NewRecorder()
		rw := NewResponseWriter(w)
		rw.WriteHeader(http.StatusOK)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = rw.HeaderWritten()
		}
	})

	b.Run("BeforeWriteHeader", func(b *testing.B) {
		w := httptest.NewRecorder()
		rw := NewResponseWriter(w)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = rw.HeaderWritten()
		}
	})
}

// BenchmarkResponseWriter_MultipleWrites measures multiple Write calls
func BenchmarkResponseWriter_MultipleWrites(b *testing.B) {
	data := []byte("chunk")

	b.Run("SingleWrite", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			_, _ = rw.Write(data)
		}
	})

	b.Run("MultipleWrites_10", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			for range 10 {
				_, _ = rw.Write(data)
			}
		}
	})
}

// BenchmarkResponseWriter_NestedLayers measures nested wrapper overhead
func BenchmarkResponseWriter_NestedLayers(b *testing.B) {
	data := []byte("test")

	layerCounts := []int{1, 3, 5}

	for _, count := range layerCounts {
		b.Run(fmt.Sprintf("Layers%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				w := httptest.NewRecorder()
				var rw http.ResponseWriter = w
				for range count {
					rw = NewResponseWriter(rw)
				}
				_, _ = rw.Write(data)
			}
		})
	}
}

// BenchmarkFlusherResponseWriter measures FlusherResponseWriter performance
func BenchmarkFlusherResponseWriter(b *testing.B) {
	b.Run("Create", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_ = NewFlusherResponseWriter(w)
		}
	})

	b.Run("Flush", func(b *testing.B) {
		w := httptest.NewRecorder()
		frw := NewFlusherResponseWriter(w)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			frw.Flush()
		}
	})

	b.Run("WriteAndFlush", func(b *testing.B) {
		data := []byte("streaming data")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			frw := NewFlusherResponseWriter(w)
			_, _ = frw.Write(data)
			frw.Flush()
		}
	})
}

// BenchmarkResponseWriter_Header measures Header() access overhead
func BenchmarkResponseWriter_Header(b *testing.B) {
	b.Run("RawResponseWriter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			_ = w.Header()
		}
	})

	b.Run("WrappedResponseWriter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			_ = rw.Header()
		}
	})
}

// BenchmarkResponseWriter_Concurrent measures concurrent access
func BenchmarkResponseWriter_Concurrent(b *testing.B) {
	data := []byte("concurrent write")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			_, _ = rw.Write(data)
		}
	})
}

// BenchmarkResponseWriter_DoubleWriteHeader tests idempotent WriteHeader
func BenchmarkResponseWriter_DoubleWriteHeader(b *testing.B) {
	b.Run("SingleWriteHeader", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			rw.WriteHeader(http.StatusOK)
		}
	})

	b.Run("DoubleWriteHeader", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			rw.WriteHeader(http.StatusOK)
			rw.WriteHeader(http.StatusInternalServerError)
		}
	})
}
