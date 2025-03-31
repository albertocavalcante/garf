package progress

import (
	"fmt"
	"io"
)

// SetupProgressReader creates a progress reader if needed.
// SetupProgressReader creates a progress-aware reader that wraps the provided io.Reader.
// It returns an error if the input reader is nil, and if the progress function is nil,
// the original reader is returned unmodified.
// If the reader supports seeking (io.Seeker), it determines the total size of the content
// by seeking to the end and then restoring the original position before wrapping it with NewReader.
func SetupProgressReader(content io.Reader, progressFunc ProgressFunc) (io.Reader, error) {
	if content == nil {
		return nil, fmt.Errorf("content reader is nil")
	}

	if progressFunc == nil {
		return content, nil
	}

	// Get the total size if available
	var total int64

	if seeker, ok := content.(io.Seeker); ok {
		// Save current position
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err == nil {
			// Seek to end to get size
			total, err = seeker.Seek(0, io.SeekEnd)
			if err == nil {
				// Restore position
				_, _ = seeker.Seek(pos, io.SeekStart)
			}
		}
	}

	return NewReader(content, total, progressFunc), nil
}
