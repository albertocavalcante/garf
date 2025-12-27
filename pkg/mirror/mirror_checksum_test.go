package mirror

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChecksumValidatingReader(t *testing.T) {
	content := "hello world"
	// sha256("hello world") = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	validChecksum := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	invalidChecksum := "invalid"

	t.Run("Valid checksum", func(t *testing.T) {
		reader := io.NopCloser(strings.NewReader(content))
		validatingReader := NewChecksumValidatingReader(reader, validChecksum)

		readContent, err := io.ReadAll(validatingReader)
		require.NoError(t, err)
		assert.Equal(t, content, string(readContent))
	})

	t.Run("Invalid checksum", func(t *testing.T) {
		reader := io.NopCloser(strings.NewReader(content))
		validatingReader := NewChecksumValidatingReader(reader, invalidChecksum)

		_, err := io.ReadAll(validatingReader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("Invalid checksum partial read", func(t *testing.T) {
		// Only reading partially shouldn't trigger error yet if we haven't hit EOF
		reader := io.NopCloser(strings.NewReader(content))
		validatingReader := NewChecksumValidatingReader(reader, invalidChecksum)

		buf := make([]byte, 5)
		n, err := validatingReader.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
	})

	t.Run("Underlying reader error", func(t *testing.T) {
		expectedErr := assert.AnError
		reader := io.NopCloser(iotest.ErrReader(expectedErr))
		validatingReader := NewChecksumValidatingReader(reader, validChecksum)

		_, err := io.ReadAll(validatingReader)
		require.ErrorIs(t, err, expectedErr)
	})
}
