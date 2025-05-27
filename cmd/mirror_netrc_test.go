package cmd_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/albertocavalcante/garf/cmd"
	"github.com/albertocavalcante/garf/pkg/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestGetJFrogCredentialsPrecedenceAndErrors(t *testing.T) {
	netrcContent := "machine art.example.com login netrcuser password netrcpass\n"
	netrcPath := testutil.CreateTemporaryNetrc(t, netrcContent)

	// Create a .netrc file with invalid permissions for the error test (Unix only)
	invalidNetrcPath := testutil.CreateTemporaryNetrc(t, "machine art.example.com login user password pass")
	if runtime.GOOS != "windows" {
		// Make it readable by others to trigger permission error (on Unix systems)
		require.NoError(t, os.Chmod(invalidNetrcPath, 0o644))
	}

	testCases := []struct {
		name     string
		flags    cmd.MirrorFlags
		env      map[string]string
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
			flags:    cmd.MirrorFlags{JFrogUser: "flaguser", JFrogPassword: "flagpass"},
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
			flags:    cmd.MirrorFlags{JFrogUser: "flaguser", JFrogPassword: "flagpass"},
			env:      map[string]string{"NETRC": filepath.Join(t.TempDir(), "non-existent-netrc")},
			wantUser: "flaguser",
			wantPass: "flagpass",
		},
		{
			name:     "partial env credentials with working netrc",
			flags:    cmd.MirrorFlags{},
			env:      map[string]string{"NETRC": netrcPath, "JFROG_USER": "envuser"},
			wantUser: "envuser",
			wantPass: "netrcpass", // Password comes from netrc
		},
		{
			name:     "partial env password with working netrc",
			flags:    cmd.MirrorFlags{},
			env:      map[string]string{"NETRC": netrcPath, "JFROG_PASSWORD": "envpass"},
			wantUser: "netrcuser", // User comes from netrc
			wantPass: "envpass",
		},
	}

	// Add permission-based error test only for Unix systems
	if runtime.GOOS != "windows" {
		testCases = append(testCases, struct {
			name     string
			flags    cmd.MirrorFlags
			env      map[string]string
			wantUser string
			wantPass string
			wantErr  string
		}{
			name:    "partial env credentials with netrc permission error",
			flags:   cmd.MirrorFlags{},
			env:     map[string]string{"NETRC": invalidNetrcPath, "JFROG_USER": "envuser"},
			wantErr: "JFrog password is required and .netrc lookup failed",
		})
	}

	jfrogURL := "https://art.example.com/artifactory"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.WithEnv(t, tc.env, func() {
				flags := tc.flags // copy
				v := viper.New()
				v.SetEnvPrefix("JFROG")
				v.AutomaticEnv()

				user, pass, err := cmd.GetJFrogCredentials(jfrogURL, &flags, v)
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

func TestExtractHostFromURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr string
	}{
		{
			name: "valid https URL",
			url:  "https://example.com/artifactory",
			want: "example.com",
		},
		{
			name: "valid http URL with port",
			url:  "http://example.com:8080/artifactory",
			want: "example.com:8080",
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: "URL cannot be empty",
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: "failed to parse JFrog URL",
		},
		{
			name:    "URL without host",
			url:     "file:///path/to/file",
			wantErr: "does not contain a valid host",
		},
		{
			name:    "malformed URL",
			url:     "not-a-url",
			want:    "not-a-url", // url.Parse treats this as a path, but Host will be empty
			wantErr: "does not contain a valid host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cmd.ExtractHostFromURL(tt.url)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
