package mirror

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultMirror_buildPreservedZipName(t *testing.T) {
	mirror := NewDefaultMirror(nil)

	tests := []struct {
		name          string
		zipName       string
		extractedName string
		expected      string
	}{
		{
			name:          "bazel zip with exe",
			zipName:       "bazel_nojdk-8.2.1-windows-x86_64.zip",
			extractedName: "bazel.exe",
			expected:      "bazel_nojdk-8.2.1-windows-x86_64.exe",
		},
		{
			name:          "uppercase ZIP extension",
			zipName:       "tool-1.0.0-linux.ZIP",
			extractedName: "tool",
			expected:      "tool-1.0.0-linux",
		},
		{
			name:          "extracted file with no extension",
			zipName:       "binary-v2.3.4.zip",
			extractedName: "binary",
			expected:      "binary-v2.3.4",
		},
		{
			name:          "extracted file with multiple extensions",
			zipName:       "archive-1.0.zip",
			extractedName: "file.tar.gz",
			expected:      "archive-1.0.tar.gz",
		},
		{
			name:          "simple case",
			zipName:       "test.zip",
			extractedName: "test.txt",
			expected:      "test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mirror.buildPreservedZipName(tt.zipName, tt.extractedName)
			require.Equal(t, tt.expected, result)
		})
	}
}
