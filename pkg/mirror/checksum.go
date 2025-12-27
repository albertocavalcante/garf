package mirror

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

// ChecksumValidatingReader wraps an io.ReadCloser and validates the SHA256 checksum
// of the read content against an expected value.
type ChecksumValidatingReader struct {
	reader    io.ReadCloser
	hash      hash.Hash
	expected  string
	validated bool
	err       error
}

// NewChecksumValidatingReader creates a new ChecksumValidatingReader.
func NewChecksumValidatingReader(reader io.ReadCloser, expectedChecksum string) *ChecksumValidatingReader {
	return &ChecksumValidatingReader{
		reader:   reader,
		hash:     sha256.New(),
		expected: expectedChecksum,
	}
}

// Read reads from the underlying reader and updates the hash.
// If EOF is reached, it validates the checksum.
func (r *ChecksumValidatingReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	n, err := r.reader.Read(p)
	if n > 0 {
		r.hash.Write(p[:n])
	}

	if err == io.EOF {
		if !r.validated {
			sum := hex.EncodeToString(r.hash.Sum(nil))
			if sum != r.expected {
				r.err = fmt.Errorf("checksum mismatch: expected %s, got %s", r.expected, sum)
				return n, r.err
			}
			r.validated = true
		}
	} else if err != nil {
		r.err = err
	}

	return n, err
}

// Close closes the underlying reader.
func (r *ChecksumValidatingReader) Close() error {
	return r.reader.Close()
}
