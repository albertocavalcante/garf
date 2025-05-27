package netrc_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/albertocavalcante/garf/pkg/netrc"
	"github.com/albertocavalcante/garf/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseNetrcScenarios(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		host        string
		expectLogin string
		expectPass  string
		expectFound bool
	}{
		{
			name:        "basic machine entry",
			content:     "machine example.com login user password pass",
			host:        "example.com",
			expectLogin: "user",
			expectPass:  "pass",
			expectFound: true,
		},
		{
			name:        "host not found",
			content:     "machine other.com login user password pass",
			host:        "example.com",
			expectFound: false,
		},
		{
			name:        "incomplete credentials",
			content:     "machine example.com login user",
			host:        "example.com",
			expectFound: false,
		},
		{
			name: "multiple machines",
			content: "machine host1.com login u1 password p1\n" +
				"machine host2.com login u2 password p2",
			host:        "host2.com",
			expectLogin: "u2",
			expectPass:  "p2",
			expectFound: true,
		},
		{
			name:        "default fallback",
			content:     "machine other.com login u1 password p1\ndefault login defuser password defpass",
			host:        "example.com",
			expectLogin: "defuser",
			expectPass:  "defpass",
			expectFound: true,
		},
		{
			name:        "machine overrides default",
			content:     "default login defuser password defpass\nmachine example.com login user password pass",
			host:        "example.com",
			expectLogin: "user",
			expectPass:  "pass",
			expectFound: true,
		},
		{
			name: "comments and whitespace",
			content: "# Comment\n" +
				"  machine example.com login user password pass  \n" +
				"# Another comment",
			host:        "example.com",
			expectLogin: "user",
			expectPass:  "pass",
			expectFound: true,
		},
		{
			name:        "quoted credentials with spaces",
			content:     `machine example.com login "user name" password "pass phrase"`,
			host:        "example.com",
			expectLogin: "user name",
			expectPass:  "pass phrase",
			expectFound: true,
		},
		{
			name:        "multiline format",
			content:     "machine example.com\nlogin user\npassword pass",
			host:        "example.com",
			expectLogin: "user",
			expectPass:  "pass",
			expectFound: true,
		},
		{
			name:        "empty file",
			content:     "",
			host:        "example.com",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login, pass, found, err := netrc.ParseNetrcFile(tt.content, tt.host)
			require.NoError(t, err)
			require.Equal(t, tt.expectFound, found)

			if tt.expectFound {
				require.Equal(t, tt.expectLogin, login)
				require.Equal(t, tt.expectPass, pass)
			}
		})
	}
}

func TestGetHostCredentials(t *testing.T) {
	content := "machine example.com login testuser password testpass"
	netrcPath := testutil.CreateTemporaryNetrc(t, content)

	testutil.WithEnv(t, map[string]string{"NETRC": netrcPath}, func() {
		creds, found, err := netrc.GetHostCredentials("example.com")
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "testuser", creds.Login)
		require.Equal(t, "testpass", creds.Password)
	})
}

func TestNonExistentFile(t *testing.T) {
	f, err := netrc.NewFile(filepath.Join(t.TempDir(), "nonexistent"))
	require.NoError(t, err)

	creds, found, err := f.GetCredentials("example.com")
	require.NoError(t, err)
	require.False(t, found)
	require.True(t, creds.IsEmpty())
}

func TestFilePermissionValidation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	content := "machine example.com login user password pass"
	dir := t.TempDir()
	netrcPath := filepath.Join(dir, ".netrc")
	require.NoError(t, os.WriteFile(netrcPath, []byte(content), 0o644))

	f, err := netrc.NewFile(netrcPath)
	require.NoError(t, err)

	_, _, err = f.GetCredentials("example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "insecure permissions")
}

func TestInputValidation(t *testing.T) {
	content := "machine example.com login user password pass"
	netrcPath := testutil.CreateTemporaryNetrc(t, content)

	f, err := netrc.NewFile(netrcPath)
	require.NoError(t, err)

	_, _, err = f.GetCredentials("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "host cannot be empty")
}

func TestDefaultPath(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "NETRC environment variable",
			env:  map[string]string{"NETRC": "/custom/.netrc", "HOME": "", "USERPROFILE": ""},
			want: "/custom/.netrc",
		},
		{
			name: "HOME fallback",
			env:  map[string]string{"NETRC": "", "HOME": "/home/user", "USERPROFILE": ""},
			want: filepath.Join("/home/user", ".netrc"),
		},
		{
			name: "USERPROFILE fallback",
			env:  map[string]string{"NETRC": "", "HOME": "", "USERPROFILE": "/Users/user"},
			want: filepath.Join("/Users/user", ".netrc"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WithEnv(t, tt.env, func() {
				path, err := netrc.DefaultPath()
				require.NoError(t, err)
				require.Equal(t, tt.want, path)
			})
		})
	}
}
