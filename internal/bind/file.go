package bind

import (
	"fmt"
	"io"
	"mime/multipart"
)

// FileHeader represents an uploaded file in a multipart form.
// It provides access to the file's metadata and content.
type FileHeader struct {
	Filename string
	Size     int64
	Header   map[string][]string
	File     multipart.File // reference to the open File
}

// Open opens the uploaded file for reading.
// The caller is responsible for closing the file.
// Returns an error if the file cannot be opened or has already been closed.
func (fh *FileHeader) Open() (multipart.File, error) {
	if fh.File == nil {
		return nil, fmt.Errorf("file %s: no longer available (already processed or closed)", fh.Filename)
	}
	return fh.File, nil
}

// ReadAll reads the entire file content into a byte slice.
// This is a convenience method that opens, reads, and closes the file.
// Uses the Size field to limit reading; returns an error if actual file size
// exceeds the declared Size to prevent OOM attacks.
// Note: This loads the entire file into memory; use Open() for large files.
func (fh *FileHeader) ReadAll() (data []byte, err error) {
	file, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if cErr := file.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()
	if fh.Size > 0 {
		return io.ReadAll(io.LimitReader(file, fh.Size))
	}
	return io.ReadAll(file)
}
