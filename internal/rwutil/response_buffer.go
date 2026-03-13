package rwutil

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
)

// ResponseBuffer is a non-thread-safe buffered ResponseWriter helper.
// Embed it in a middleware-specific wrapper that owns the mutex.
// It manages five states: buffering, overflow, header written, body written, and flushed.
type ResponseBuffer struct {
	http.ResponseWriter
	Buf           bytes.Buffer
	Status        int
	MaxBodySize   int64
	HasWritten    bool // WriteHeader or Write was called
	HeaderWritten bool // underlying ResponseWriter.WriteHeader was called
	Buffering     bool // true while buffering, false when in pass-through mode
}

// NewResponseBuffer creates a new ResponseBuffer.
func NewResponseBuffer(w http.ResponseWriter, maxBodySize int64) *ResponseBuffer {
	return &ResponseBuffer{
		ResponseWriter: w,
		Status:         http.StatusOK,
		MaxBodySize:    maxBodySize,
		Buffering:      true,
	}
}

// Reset resets the buffer for reuse (for use with sync.Pool).
func (b *ResponseBuffer) Reset(w http.ResponseWriter) {
	b.ResponseWriter = w
	b.Buf.Reset()
	b.Status = http.StatusOK
	b.HasWritten = false
	b.HeaderWritten = false
	b.Buffering = true
}

// WriteHeader captures the status code.
// Does NOT forward to underlying writer - caller decides when to commit.
func (b *ResponseBuffer) WriteHeader(status int) {
	if b.HasWritten {
		return
	}
	b.HasWritten = true
	b.Status = status
}

// CommitHeader writes the status code to the underlying ResponseWriter.
// Caller should set response headers before calling this.
func (b *ResponseBuffer) CommitHeader() {
	if !b.HeaderWritten {
		b.ResponseWriter.WriteHeader(b.Status)
		b.HeaderWritten = true
	}
}

// Write buffers data until maxBodySize is reached, then switches to pass-through.
func (b *ResponseBuffer) Write(p []byte) (int, error) {
	if !b.HasWritten {
		b.WriteHeader(http.StatusOK)
	}
	if !b.Buffering {
		return b.ResponseWriter.Write(p)
	}
	if b.MaxBodySize > 0 && int64(b.Buf.Len()+len(p)) > b.MaxBodySize {
		// Switch to pass-through mode
		b.Buffering = false
		b.CommitHeader()
		if b.Buf.Len() > 0 {
			_, _ = b.ResponseWriter.Write(b.Buf.Bytes())
			b.Buf.Reset()
		}
		return b.ResponseWriter.Write(p)
	}
	return b.Buf.Write(p)
}

// Commit writes buffered status+body to the underlying writer.
// Caller should set response headers before calling this.
func (b *ResponseBuffer) Commit() {
	if !b.Buffering {
		return
	}
	b.Buffering = false
	b.CommitHeader()
	if b.Buf.Len() > 0 {
		_, _ = b.ResponseWriter.Write(b.Buf.Bytes())
		b.Buf.Reset()
	}
}

// FlushTo handles all three Flush cases. onFlush is called (while caller
// holds its lock) only when there is buffered content to commit, allowing
// the middleware to set final headers (ETag, Cache-Control, etc.) before
// CommitHeader fires.
func (b *ResponseBuffer) FlushTo(underlying http.Flusher, onFlush func()) {
	if b.Buffering {
		if onFlush != nil {
			onFlush()
		}
		b.Commit()
	}
	if underlying != nil {
		underlying.Flush()
	}
}

// Hijack implements http.Hijacker.
func (b *ResponseBuffer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := b.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("response writer does not support hijacking")
}

// Push implements http.Pusher.
func (b *ResponseBuffer) Push(target string, opts *http.PushOptions) error {
	if ps, ok := b.ResponseWriter.(http.Pusher); ok {
		return ps.Push(target, opts)
	}
	return errors.New("response writer does not support push")
}

// Ensure interface compliance at compile time.
var (
	_ http.ResponseWriter = (*ResponseBuffer)(nil)
	_ http.Hijacker       = (*ResponseBuffer)(nil)
	_ http.Pusher         = (*ResponseBuffer)(nil)
)
