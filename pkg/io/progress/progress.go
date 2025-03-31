// Package progress provides utilities for tracking progress of operations.
package progress

import "io"

// - message: A descriptive message about the current operation.
type ProgressFunc func(current, total int64, message string)

// Reader wraps an io.Reader to track progress.
type Reader struct {
	reader       io.Reader
	total        int64
	current      int64
	progressFunc ProgressFunc
}

// NewReader creates a new progress-tracking Reader that wraps the provided io.Reader.
// It uses the given total byte count to monitor progress and, if provided,
// calls the progressFunc with updates that include the current progress and a descriptive message.
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
