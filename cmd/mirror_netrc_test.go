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

// Helper to save and restore environment variables
func withEnv(t *testing.T, env map[string]string, fn func()) {
	t.Helper()
	// Save originals
	orig := map[string]string{}
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

func TestGetJFrogCredentialsPrecedenceAndErrors(t *testing.T) {
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)

	netrcContent := "machine art.example.com login netrcuser password netrcpass\n"
	netrcPath := createTemporaryNetrc(t, netrcContent)

	testCases := []struct {
		name     string
		flags    MirrorFlags
		env      map[string]string
		netrc    string // path to .netrc file, empty means no .netrc
		wantUser string
		wantPass string
		wantErr  string // substring match, empty means expect no error
	}{
		{
			name:     "netrc only",
			env:      map[string]string{"NETRC": netrcPath},
			wantUser: "netrcuser",
			wantPass: "netrcpass",
		},
		{
			name:     "env overrides netrc",
			env:      map[string]string{"NETRC": netrcPath, "JFROG_USER": "envuser", "JFROG_PASSWORD": "envpass"},
			wantUser: "envuser",
			wantPass: "envpass",
		},
		{
			name:     "flags override env",
			flags:    MirrorFlags{JFrogUser: "flaguser", JFrogPassword: "flagpass"},
			env:      map[string]string{"NETRC": netrcPath, "JFROG_USER": "envuser", "JFROG_PASSWORD": "envpass"},
			wantUser: "flaguser",
			wantPass: "flagpass",
		},
		{
			name:    "no credentials anywhere",
			env:     map[string]string{"NETRC": filepath.Join(t.TempDir(), "non-existent-netrc")},
			wantErr: "JFrog user is required",
		},
		{
			name:    "env user only, missing password",
			env:     map[string]string{"NETRC": filepath.Join(t.TempDir(), "non-existent-netrc"), "JFROG_USER": "envuser"},
			wantErr: "JFrog password is required",
		},
		{
			name:     "flags provide all",
			flags:    MirrorFlags{JFrogUser: "flaguser", JFrogPassword: "flagpass"},
			env:      map[string]string{"NETRC": filepath.Join(t.TempDir(), "non-existent-netrc")},
			wantUser: "flaguser",
			wantPass: "flagpass",
		},
	}

	jfrogURL := "https://art.example.com/artifactory"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withEnv(t, tc.env, func() {
				flags := tc.flags // copy
				v := viper.New()
				v.SetEnvPrefix("JFROG")
				v.AutomaticEnv()

				user, pass, err := getJFrogCredentials(jfrogURL, &flags, v)
				if tc.wantErr != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.wantErr)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tc.wantUser, user)
				require.Equal(t, tc.wantPass, pass)
			})
		})
	}
}
