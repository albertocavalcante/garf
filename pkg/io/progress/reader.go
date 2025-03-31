package progress

import (
	"fmt"
	"io"
)

// SetupProgressReader creates a progress reader if needed.
// It attempts to determine the total size if the reader supports seeking.
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
