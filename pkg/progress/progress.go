// Package progress provides utilities for tracking progress of operations.
package progress

import "io"

// ProgressFunc is a function that is called to report progress.
type ProgressFunc func(current, total int64, message string)

// Reader wraps an io.Reader to track progress.
type Reader struct {
	reader       io.Reader
	total        int64
	current      int64
	progressFunc ProgressFunc
}

// NewReader returns a new Reader that wraps the provided io.Reader with progress tracking.
// It sets the total number of bytes expected to be read and assigns the provided progress function
// (if non-nil) to report progress during read operations.
func NewReader(reader io.Reader, total int64, progressFunc ProgressFunc) *Reader {
	return &Reader{
		reader:       reader,
		total:        total,
		progressFunc: progressFunc,
	}
}

// Read implements the io.Reader interface.
func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		r.current += int64(n)
		if r.progressFunc != nil {
			r.progressFunc(r.current, r.total, "Processing")
		}
	}

	return
}
