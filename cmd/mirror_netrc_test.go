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

func TestReadNetrcCredentials(t *testing.T) {
	host := "art.example.com"
	netrcContent := "machine " + host + " login myuser password mypass\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)

	// Override NETRC env var for the duration of the test.
	origNetrc := os.Getenv("NETRC")
	os.Setenv("NETRC", netrcPath)
	defer os.Setenv("NETRC", origNetrc)

	user, pass, ok, err := readNetrcCredentials(host)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "myuser", user)
	require.Equal(t, "mypass", pass)
}

func TestGetJFrogCredentialsPrecedence(t *testing.T) {
	// Save original environment variables
	origUser := os.Getenv("JFROG_USER")
	origPass := os.Getenv("JFROG_PASSWORD")

	// Clean up environment variables to avoid interference
	os.Unsetenv("JFROG_USER")
	os.Unsetenv("JFROG_PASSWORD")
	defer func() {
		// Restore original values
		if origUser != "" {
			os.Setenv("JFROG_USER", origUser)
		}
		if origPass != "" {
			os.Setenv("JFROG_PASSWORD", origPass)
		}
	}()

	flags := &MirrorFlags{}
	jfrogURL := "https://art.example.com/artifactory"

	// Prepare .netrc credentials.
	netrcContent := "machine art.example.com login netrcuser password netrcpass\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)
	origNetrc := os.Getenv("NETRC")
	os.Setenv("NETRC", netrcPath)
	defer os.Setenv("NETRC", origNetrc)

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

	user, pass, err = func() (string, string, error) {
		u, p, err := getJFrogCredentials(jfrogURL, flags, v)
		return u, p, err
	}()

	require.NoError(t, err)
	require.Equal(t, "envuser", user)
	require.Equal(t, "envpass", pass)

	// Test precedence: flags should override environment variables
	flags.JFrogUser = "flaguser"
	flags.JFrogPassword = "flagpass"

	user, pass, err = func() (string, string, error) {
		u, p, err := getJFrogCredentials(jfrogURL, flags, v)
		return u, p, err
	}()

	require.NoError(t, err)
	require.Equal(t, "flaguser", user)
	require.Equal(t, "flagpass", pass)
}
