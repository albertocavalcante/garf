package netrc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTemporaryNetrc creates a temporary .netrc file with the specified content and returns its path.
func createTemporaryNetrc(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, ".netrc")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
	return file
}

func TestGetCredentials(t *testing.T) {
	host := "art.example.com"
	netrcContent := "machine " + host + " login myuser password mypass\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)

	// Create a File instance with the test .netrc file
	f, err := NewFile(netrcPath)
	require.NoError(t, err)

	// Get credentials for the host
	creds, found, err := f.GetCredentials(host)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "myuser", creds.Login)
	require.Equal(t, "mypass", creds.Password)
}

func TestMissingHost(t *testing.T) {
	// Create .netrc file with a different host
	netrcContent := "machine othermachine.example.com login otheruser password otherpass\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)

	// Create a File instance with the test .netrc file
	f, err := NewFile(netrcPath)
	require.NoError(t, err)

	// Try to get credentials for a non-existent host
	host := "missing.example.com"
	creds, found, err := f.GetCredentials(host)

	// Should not error, but should indicate no credentials found
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, creds.Login)
	require.Empty(t, creds.Password)
}

func TestIncompleteCredentials(t *testing.T) {
	// Create .netrc file with incomplete credentials (missing password)
	host := "incomplete.example.com"
	netrcContent := "machine " + host + " login useronly\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)

	// Create a File instance with the test .netrc file
	f, err := NewFile(netrcPath)
	require.NoError(t, err)

	// Get credentials for the host
	creds, found, err := f.GetCredentials(host)

	// Should not error but indicate no credentials found since they are incomplete
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, creds.Login)
	require.Empty(t, creds.Password)
}

func TestMultipleHosts(t *testing.T) {
	// Create .netrc file with multiple hosts
	netrcContent := `machine host1.example.com login user1 password pass1
machine host2.example.com login user2 password pass2
machine host3.example.com login user3 password pass3`

	netrcPath := createTemporaryNetrc(t, netrcContent)

	// Create a File instance with the test .netrc file
	f, err := NewFile(netrcPath)
	require.NoError(t, err)

	// Get credentials for the second host
	creds, found, err := f.GetCredentials("host2.example.com")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "user2", creds.Login)
	require.Equal(t, "pass2", creds.Password)
}

func TestCommentsAndWhitespace(t *testing.T) {
	// Test that comments and whitespace are properly handled
	host := "test.example.com"
	netrcContent := `# This is a comment
machine ` + host + ` login userWithComment password passWithComment
# Another comment
`
	netrcPath := createTemporaryNetrc(t, netrcContent)

	// Create a File instance with the test .netrc file
	f, err := NewFile(netrcPath)
	require.NoError(t, err)

	// Get credentials for the host
	creds, found, err := f.GetCredentials(host)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "userWithComment", creds.Login)
	require.Equal(t, "passWithComment", creds.Password)
}

func TestGetHostCredentials(t *testing.T) {
	// Create .netrc file and set NETRC environment variable
	host := "myhost.example.com"
	netrcContent := "machine " + host + " login envuser password envpass\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)

	// Save original NETRC value and restore after test
	origNetrc := os.Getenv("NETRC")
	os.Setenv("NETRC", netrcPath)
	defer os.Setenv("NETRC", origNetrc)

	// Use the convenience function
	creds, found, err := GetHostCredentials(host)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "envuser", creds.Login)
	require.Equal(t, "envpass", creds.Password)
}

func TestNonExistentFile(t *testing.T) {
	// Create a non-existent file path
	netrcPath := filepath.Join(t.TempDir(), "non-existent-file")

	// Create a File instance with the non-existent file
	f, err := NewFile(netrcPath)
	require.NoError(t, err)

	// Attempt to get credentials
	creds, found, err := f.GetCredentials("anyhost.example.com")

	// Should not error, just indicate not found
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, creds.Login)
	require.Empty(t, creds.Password)
}

func TestDefaultPath(t *testing.T) {
	// Save original values
	origNetrc := os.Getenv("NETRC")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("NETRC", origNetrc)
		os.Setenv("HOME", origHome)
	}()

	// Test with NETRC environment variable
	customPath := "/custom/path/.netrc"
	os.Setenv("NETRC", customPath)
	os.Unsetenv("HOME") // Clear HOME to ensure we're using NETRC

	path, err := DefaultPath()
	require.NoError(t, err)
	require.Equal(t, customPath, path)

	// Test without NETRC environment variable - use a fake home directory
	os.Unsetenv("NETRC")
	fakeHome := "/fake/home"
	os.Setenv("HOME", fakeHome)

	path, err = DefaultPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(fakeHome, ".netrc"), path)
}
