// Package testutil provides shared testing utilities for the garf project.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	// NetrcFilePermissions defines the secure permissions for .netrc files (0600).
	NetrcFilePermissions = 0o600
)

// CreateTemporaryNetrc creates a temporary .netrc file with the specified content.
// The file is created with 0600 permissions for security.
func CreateTemporaryNetrc(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, ".netrc")
	require.NoError(t, os.WriteFile(file, []byte(content), NetrcFilePermissions))

	return file
}

// WithEnv temporarily sets environment variables for the duration of a test function.
// It automatically restores the original values when the test completes.
func WithEnv(t *testing.T, env map[string]string, fn func()) {
	t.Helper()
	// Save originals
	orig := make(map[string]string, len(env))
	for k := range env {
		orig[k] = os.Getenv(k)

		if env[k] == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, env[k])
		}
	}

	t.Cleanup(func() {
		for k, v := range orig {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})
	fn()
}
