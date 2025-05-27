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

type parseTestCase struct {
	name           string
	netrcContent   string
	targetHost     string
	expectedLogin  string
	expectedPass   string
	expectedFound  bool
	expectParseErr bool // If true, expects parseNetrcFile to return an error
}

// TestParseNetrcScenarios provides table-driven tests for various .netrc parsing scenarios.
func TestParseNetrcScenarios(t *testing.T) {
	testCases := []parseTestCase{
		{
			name:          "basic credential retrieval",
			netrcContent:  "machine host1.example.com login user1 password pass1",
			targetHost:    "host1.example.com",
			expectedLogin: "user1",
			expectedPass:  "pass1",
			expectedFound: true,
		},
		{
			name:          "missing host",
			netrcContent:  "machine other.example.com login otheruser password otherpass",
			targetHost:    "missing.example.com",
			expectedFound: false,
		},
		{
			name:          "incomplete credentials - missing password keyword",
			netrcContent:  "machine host1.example.com login user1",
			targetHost:    "host1.example.com",
			expectedFound: false, // Login found but no password means not a complete credential pair
		},
		{
			name:          "incomplete credentials - missing password value",
			netrcContent:  "machine host1.example.com login user1 password",
			targetHost:    "host1.example.com",
			expectedFound: false, // Login found, password keyword present, but no value
		},
		{
			name: "multiple hosts, target found",
			netrcContent: "machine host1 login u1 password p1\n" +
				"machine host2 login u2 password p2\n" +
				"machine host3 login u3 password p3",
			targetHost:    "host2",
			expectedLogin: "u2",
			expectedPass:  "p2",
			expectedFound: true,
		},
		{
			name: "comments and whitespace",
			netrcContent: "# Comment line\n" +
				"  machine host.comment login user_comment password pass_comment  \n" +
				"\tmachine otherhost login uo password po # Trailing comment",
			targetHost:    "host.comment",
			expectedLogin: "user_comment",
			expectedPass:  "pass_comment",
			expectedFound: true,
		},
		{
			name:          "first match wins for specific host",
			netrcContent:  "machine host1 login u1_first password p1_first\nmachine host1 login u1_second password p1_second",
			targetHost:    "host1",
			expectedLogin: "u1_first",
			expectedPass:  "p1_first",
			expectedFound: true,
		},
		{
			name:          "default keyword used when host not found",
			netrcContent:  "machine host1 login u1 password p1\ndefault login def_user password def_pass",
			targetHost:    "unknown.host",
			expectedLogin: "def_user",
			expectedPass:  "def_pass",
			expectedFound: true,
		},
		{
			name:          "specific host overrides default",
			netrcContent:  "machine host1 login u1 password p1\ndefault login def_user password def_pass",
			targetHost:    "host1",
			expectedLogin: "u1",
			expectedPass:  "p1",
			expectedFound: true,
		},
		{
			name:          "first default wins if multiple defaults",
			netrcContent:  "default login def1 password p1\nmachine host1 login u1 password p1\ndefault login def2 password p2",
			targetHost:    "unknown.host",
			expectedLogin: "def1",
			expectedPass:  "p1",
			expectedFound: true,
		},
		{
			name:          "macdef skipped, finds subsequent entry",
			netrcContent:  "macdef init\n  echo hello\nmachine host1 login u1 password p1",
			targetHost:    "host1",
			expectedLogin: "u1",
			expectedPass:  "p1",
			expectedFound: true,
		},
		{
			name:          "quoted login with spaces",
			netrcContent:  `machine host.quoted login "user name" password pass`,
			targetHost:    "host.quoted",
			expectedLogin: "user name",
			expectedPass:  "pass",
			expectedFound: true,
		},
		{
			name:          "quoted password with spaces",
			netrcContent:  `machine host.quoted login user password "pass phrase"`,
			targetHost:    "host.quoted",
			expectedLogin: "user",
			expectedPass:  "pass phrase",
			expectedFound: true,
		},
		{
			name:          "quoted login and password with spaces",
			netrcContent:  `machine host.quoted login "user name" password "pass phrase"`,
			targetHost:    "host.quoted",
			expectedLogin: "user name",
			expectedPass:  "pass phrase",
			expectedFound: true,
		},
		{
			name:          "escaped quotes in password",
			netrcContent:  `machine host1 login user1 password "pass\"word"`, // pass"word
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  `pass"word`,
			expectedFound: true,
		},
		{
			name:          "escaped backslash in password",
			netrcContent:  `machine host1 login user1 password "pass\\word"`, // pass\word
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  `pass\word`,
			expectedFound: true,
		},
		{
			name:          "escaped newline in password",
			netrcContent:  `machine host1 login user1 password "pass\nword"`, // pass<newline>word
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  "pass\nword",
			expectedFound: true,
		},
		{
			name:          "mixed keywords on one line",
			netrcContent:  "machine host1 login user1 password pass1 machine host2 login user2 password pass2",
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  "pass1",
			expectedFound: true,
		},
		{
			name:          "mixed keywords on one line, target second",
			netrcContent:  "machine host1 login user1 password pass1 machine host2 login user2 password pass2",
			targetHost:    "host2",
			expectedLogin: "user2",
			expectedPass:  "pass2",
			expectedFound: true,
		},
		{
			name:          "keywords spread across lines (machine then login/pass)",
			netrcContent:  "machine host1\nlogin user1\npassword pass1",
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  "pass1",
			expectedFound: true,
		},
		{
			name:          "login and password on same line after machine on previous",
			netrcContent:  "machine host1\nlogin user1 password pass1",
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  "pass1",
			expectedFound: true,
		},
		{
			name:          "empty file",
			netrcContent:  "",
			targetHost:    "anyhost",
			expectedFound: false,
		},
		{
			name:          "only comments",
			netrcContent:  "# machine host1 login user1 password pass1",
			targetHost:    "host1",
			expectedFound: false,
		},
		{
			name:          "unknown keywords are ignored",
			netrcContent:  "machine host1 foobar baz login user1 qux password pass1",
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  "pass1",
			expectedFound: true,
		},
		{
			name:          "password before login for a machine is ignored",
			netrcContent:  "machine host1 password pass1 login user1",
			targetHost:    "host1",
			expectedLogin: "user1", // Assuming parser resets or requires login first
			expectedPass:  "",      // Password without prior login for the block should be ignored
			expectedFound: false,   // Thus, not found as a complete pair
		},
		{
			name:          "dangling backslash in quoted password",
			netrcContent:  `machine host1 login user1 password "pass\\"`, // pass\
			targetHost:    "host1",
			expectedLogin: "user1",
			expectedPass:  "pass\\",
			expectedFound: true,
		},
		{
			name:          "empty quoted string for login",
			netrcContent:  `machine host1 login "" password "pass"`,
			targetHost:    "host1",
			expectedLogin: "",
			expectedPass:  "pass",
			expectedFound: true,
		},
		{
			name:          "empty quoted string for password",
			netrcContent:  `machine host1 login "user" password ""`,
			targetHost:    "host1",
			expectedLogin: "user",
			expectedPass:  "",
			expectedFound: true,
		},
		// New edge case tests
		{
			name:          "malformed entry - machine without value",
			netrcContent:  "machine\nlogin user password pass",
			targetHost:    "anyhost",
			expectedFound: false,
		},
		{
			name:          "malformed entry - login without value",
			netrcContent:  "machine host1 login\npassword pass",
			targetHost:    "host1",
			expectedFound: false,
		},
		{
			name:          "unclosed quoted string",
			netrcContent:  `machine host1 login "unclosed password pass`,
			targetHost:    "host1",
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use parseNetrcFile directly to test its logic thoroughly
			login, pass, found, err := netrc.ParseNetrcFile(tc.netrcContent, tc.targetHost)

			if tc.expectParseErr {
				require.Error(t, err, "Expected a parsing error but got none")

				return
			}

			require.NoError(t, err, "Got unexpected parsing error")

			require.Equal(t, tc.expectedFound, found)

			if tc.expectedFound {
				require.Equal(t, tc.expectedLogin, login, "Login mismatch")
				require.Equal(t, tc.expectedPass, pass, "Password mismatch")
			} else {
				// If not found, login/pass should be empty, good to assert this explicitly
				require.Empty(t, login, "Login should be empty when not found")
				require.Empty(t, pass, "Password should be empty when not found")
			}
		})
	}
}

func TestGetHostCredentials(t *testing.T) {
	// Create .netrc file and set NETRC environment variable
	host := "myhost.example.com"
	netrcContent := "machine " + host + " login envuser password envpass\n"
	netrcPath := testutil.CreateTemporaryNetrc(t, netrcContent)

	testutil.WithEnv(t, map[string]string{"NETRC": netrcPath}, func() {
		// Use the convenience function
		creds, found, err := netrc.GetHostCredentials(host)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "envuser", creds.Login)
		require.Equal(t, "envpass", creds.Password)
		require.False(t, creds.IsEmpty())
	})
}

func TestNonExistentFile(t *testing.T) {
	// Create a non-existent file path
	netrcPath := filepath.Join(t.TempDir(), "non-existent-file")

	// Create a File instance with the non-existent file
	f, err := netrc.NewFile(netrcPath)
	require.NoError(t, err) // NewFile itself shouldn't error for non-existent path if path is valid format

	// Attempt to get credentials
	creds, found, err := f.GetCredentials("anyhost.example.com")

	// Should not error, just indicate not found because file doesn't exist
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, creds.Login)
	require.Empty(t, creds.Password)
	require.True(t, creds.IsEmpty())
}

func TestFilePermissionValidation(t *testing.T) {
	// Skip this test on Windows as permission validation is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	host := "secure.example.com"
	netrcContent := "machine " + host + " login user password pass\n"

	// Create a .netrc file with insecure permissions
	dir := t.TempDir()
	netrcPath := filepath.Join(dir, ".netrc")
	require.NoError(t, os.WriteFile(netrcPath, []byte(netrcContent), 0o644)) // Readable by others

	f, err := netrc.NewFile(netrcPath)
	require.NoError(t, err)

	// Should fail due to insecure permissions
	_, _, err = f.GetCredentials(host)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insecure permissions")

	// Fix permissions and try again
	require.NoError(t, os.Chmod(netrcPath, 0o600))

	creds, found, err := f.GetCredentials(host)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "user", creds.Login)
	require.Equal(t, "pass", creds.Password)
}

func TestInputValidation(t *testing.T) {
	netrcContent := "machine host1 login user1 password pass1\n"
	netrcPath := testutil.CreateTemporaryNetrc(t, netrcContent)

	f, err := netrc.NewFile(netrcPath)
	require.NoError(t, err)

	// Test empty host
	_, _, err = f.GetCredentials("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "host cannot be empty")
}

func TestDefaultPath(t *testing.T) {
	// Test with NETRC environment variable
	customNetrcPath := "/custom/path/" + netrc.DefaultNetrcFilename
	testutil.WithEnv(t, map[string]string{
		"NETRC":       customNetrcPath,
		"HOME":        "",
		"USERPROFILE": "",
	}, func() {
		path, err := netrc.DefaultPath()
		require.NoError(t, err)
		require.Equal(t, customNetrcPath, path)
	})

	// Test with USERPROFILE (Windows primary fallback)
	fakeUserProfile := t.TempDir() // Use TempDir to get a valid path
	testutil.WithEnv(t, map[string]string{
		"NETRC":       "",
		"USERPROFILE": fakeUserProfile,
		"HOME":        "",
	}, func() {
		path, err := netrc.DefaultPath()
		require.NoError(t, err)
		require.Equal(t, filepath.Join(fakeUserProfile, netrc.DefaultNetrcFilename), path)
	})

	// Test with HOME (Unix-like primary fallback, or Windows secondary)
	fakeHome := t.TempDir() // Use TempDir for a valid path
	testutil.WithEnv(t, map[string]string{
		"NETRC":       "",
		"USERPROFILE": "",
		"HOME":        fakeHome,
	}, func() {
		path, err := netrc.DefaultPath()
		require.NoError(t, err)
		require.Equal(t, filepath.Join(fakeHome, netrc.DefaultNetrcFilename), path)
	})
}

func TestCredentialsIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		creds    netrc.Credentials
		expected bool
	}{
		{
			name:     "empty credentials",
			creds:    netrc.Credentials{},
			expected: true,
		},
		{
			name:     "empty login and password",
			creds:    netrc.Credentials{Login: "", Password: ""},
			expected: true,
		},
		{
			name:     "has login only",
			creds:    netrc.Credentials{Login: "user", Password: ""},
			expected: false,
		},
		{
			name:     "has password only",
			creds:    netrc.Credentials{Login: "", Password: "pass"},
			expected: false,
		},
		{
			name:     "has both",
			creds:    netrc.Credentials{Login: "user", Password: "pass"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.creds.IsEmpty())
		})
	}
}
