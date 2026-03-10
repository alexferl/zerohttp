package rwutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	if rw.StatusCode() != http.StatusOK {
		t.Errorf("expected default status code %d, got %d", http.StatusOK, rw.StatusCode())
	}

	if rw.HeaderWritten() {
		t.Error("expected HeaderWritten to be false initially")
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)

	if rw.StatusCode() != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, rw.StatusCode())
	}

	if !rw.HeaderWritten() {
		t.Error("expected HeaderWritten to be true after WriteHeader")
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected recorder code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestResponseWriter_WriteHeader_MultipleCalls(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)
	rw.WriteHeader(http.StatusInternalServerError) // Should be ignored

	if rw.StatusCode() != http.StatusNotFound {
		t.Errorf("expected status code to remain %d, got %d", http.StatusNotFound, rw.StatusCode())
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected recorder code to remain %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	data := []byte("hello world")
	n, err := rw.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	if rw.StatusCode() != http.StatusOK {
		t.Errorf("expected status code %d after Write, got %d", http.StatusOK, rw.StatusCode())
	}

	if !rw.HeaderWritten() {
		t.Error("expected HeaderWritten to be true after Write")
	}

	if rec.Body.String() != "hello world" {
		t.Errorf("expected body %q, got %q", "hello world", rec.Body.String())
	}
}

func TestResponseWriter_Write_WithHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusCreated)
	n, err := rw.Write([]byte("created"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != 7 {
		t.Errorf("expected to write 7 bytes, wrote %d", n)
	}

	if rw.StatusCode() != http.StatusCreated {
		t.Errorf("expected status code %d, got %d", http.StatusCreated, rw.StatusCode())
	}
}

func TestResponseWriter_Header(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.Header().Set("X-Custom-Header", "value")

	if rec.Header().Get("X-Custom-Header") != "value" {
		t.Error("expected header to be set on underlying recorder")
	}
}

func TestNewFlusherResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	frw := NewFlusherResponseWriter(rec)

	if frw.StatusCode() != http.StatusOK {
		t.Errorf("expected default status code %d, got %d", http.StatusOK, frw.StatusCode())
	}

	// Should not panic
	frw.Flush()
}

func TestFlusherResponseWriter_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	frw := NewFlusherResponseWriter(rec)

	// Write some data and flush
	_, _ = frw.Write([]byte("data"))
	frw.Flush()

	// The recorder should have the data
	if rec.Body.String() != "data" {
		t.Errorf("expected body %q, got %q", "data", rec.Body.String())
	}
}
