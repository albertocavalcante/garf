package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
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

func TestGetJFrogCredentialsPrecedence(t *testing.T) {
	// Save original environment variables
	origUser := os.Getenv("JFROG_USER")
	origPass := os.Getenv("JFROG_PASSWORD")
	origNetrc := os.Getenv("NETRC")
	origHome := os.Getenv("HOME")

	// Clean up environment variables to avoid interference
	os.Unsetenv("JFROG_USER")
	os.Unsetenv("JFROG_PASSWORD")
	os.Setenv("HOME", t.TempDir()) // Set HOME to a writable directory for tests

	defer func() {
		// Restore original values
		if origUser != "" {
			os.Setenv("JFROG_USER", origUser)
		}
		if origPass != "" {
			os.Setenv("JFROG_PASSWORD", origPass)
		}
		if origNetrc != "" {
			os.Setenv("NETRC", origNetrc)
		}
		if origHome != "" {
			os.Setenv("HOME", origHome)
		}
	}()

	flags := &MirrorFlags{}
	jfrogURL := "https://art.example.com/artifactory"

	// Prepare .netrc credentials
	netrcContent := "machine art.example.com login netrcuser password netrcpass\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)
	os.Setenv("NETRC", netrcPath)

	v := viper.New()
	v.SetEnvPrefix("JFROG")
	v.AutomaticEnv()

	user, pass, err := getJFrogCredentials(jfrogURL, flags, v)

	require.NoError(t, err)
	require.Equal(t, "netrcuser", user)
	require.Equal(t, "netrcpass", pass)

	// Test precedence: environment variables should override netrc
	os.Setenv("JFROG_USER", "envuser")
	os.Setenv("JFROG_PASSWORD", "envpass")

	user, pass, err = getJFrogCredentials(jfrogURL, flags, v)
	require.NoError(t, err)
	require.Equal(t, "envuser", user)
	require.Equal(t, "envpass", pass)

	// Test precedence: flags should override environment variables
	flags.JFrogUser = "flaguser"
	flags.JFrogPassword = "flagpass"

	user, pass, err = getJFrogCredentials(jfrogURL, flags, v)
	require.NoError(t, err)
	require.Equal(t, "flaguser", user)
	require.Equal(t, "flagpass", pass)
}

func TestGetJFrogCredentialsErrorHandling(t *testing.T) {
	// Save original environment variables
	origUser := os.Getenv("JFROG_USER")
	origPass := os.Getenv("JFROG_PASSWORD")
	origNetrc := os.Getenv("NETRC")
	origHome := os.Getenv("HOME")

	// Clean up environment variables to avoid interference
	os.Unsetenv("JFROG_USER")
	os.Unsetenv("JFROG_PASSWORD")
	os.Setenv("HOME", t.TempDir()) // Set HOME to a writable directory for tests

	defer func() {
		// Restore original values
		if origUser != "" {
			os.Setenv("JFROG_USER", origUser)
		}
		if origPass != "" {
			os.Setenv("JFROG_PASSWORD", origPass)
		}
		if origNetrc != "" {
			os.Setenv("NETRC", origNetrc)
		}
		if origHome != "" {
			os.Setenv("HOME", origHome)
		}
	}()

	flags := &MirrorFlags{}
	jfrogURL := "https://art.example.com/artifactory"

	// Create a non-existent netrc file path to ensure no credentials are found
	nonExistentNetrc := filepath.Join(t.TempDir(), "non-existent-netrc")
	os.Setenv("NETRC", nonExistentNetrc)

	v := viper.New()
	v.SetEnvPrefix("JFROG")
	v.AutomaticEnv()

	// Case 1: No credentials anywhere - should get user required error
	_, _, err := getJFrogCredentials(jfrogURL, flags, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "JFrog user is required")

	// Case 2: With username from env, should still error with password required
	os.Setenv("JFROG_USER", "envuser")
	_, _, err = getJFrogCredentials(jfrogURL, flags, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "JFrog password is required")

	// Case 3: With complete credentials, should not error
	flags.JFrogUser = "flaguser"
	flags.JFrogPassword = "flagpass"
	user, pass, err := getJFrogCredentials(jfrogURL, flags, v)
	require.NoError(t, err)
	require.Equal(t, "flaguser", user)
	require.Equal(t, "flagpass", pass)
}
