package cmd_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/albertocavalcante/garf/cmd"
	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/albertocavalcante/garf/pkg/sources"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// configTestCase defines a test case for ValidateAndGetConfig testing.
type configTestCase struct {
	name          string
	source        string
	destination   string
	jfrogURL      string
	jfrogUser     string
	jfrogPassword string
	envVars       map[string]string
	shouldErr     bool
}

// validateConfigTestCases contains all test cases for ValidateAndGetConfig.
var validateConfigTestCases = []configTestCase{
	{
		name:        "Valid config with all required params via env vars",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: false,
	},
	{
		name:          "Valid config with all required params via flags",
		source:        "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination:   "repo-local",
		jfrogURL:      "https://example.jfrog.io/artifactory",
		jfrogUser:     "user",
		jfrogPassword: "password",
		envVars:       map[string]string{},
		shouldErr:     false,
	},
	{
		name:        "Missing source",
		source:      "",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing destination",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing JFROG_URL",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing JFROG_USER",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing JFROG_PASSWORD",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":  "https://example.jfrog.io/artifactory",
			"JFROG_USER": "user",
		},
		shouldErr: true,
	},
	{
		name:          "Flag overrides env var",
		source:        "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination:   "repo-local",
		jfrogURL:      "https://flag.example.com/artifactory",
		jfrogUser:     "flag-user",
		jfrogPassword: "flag-password",
		envVars: map[string]string{
			"JFROG_URL":      "https://env.example.com/artifactory",
			"JFROG_USER":     "env-user",
			"JFROG_PASSWORD": "env-password",
		},
		shouldErr: false,
	},
}

// TestValidateAndGetConfig tests the ValidateAndGetConfig function with various
// combinations of valid and invalid inputs.
func TestValidateAndGetConfig(t *testing.T) {
	// Save original environment
	origEnv := saveEnvironment([]string{"JFROG_URL", "JFROG_USER", "JFROG_PASSWORD"})

	// Restore environment after tests
	defer restoreEnvironment(origEnv)

	for _, tc := range validateConfigTestCases {
		t.Run(tc.name, func(t *testing.T) {
			runValidateConfigTest(t, tc)
		})
	}
}

// runValidateConfigTest runs a single test case for ValidateAndGetConfig.
func runValidateConfigTest(t *testing.T, tc configTestCase) {
	// Clear environment
	os.Unsetenv("JFROG_URL")
	os.Unsetenv("JFROG_USER")
	os.Unsetenv("JFROG_PASSWORD")

	// Set test environment variables
	for k, v := range tc.envVars {
		os.Setenv(k, v)
	}

	// Run the function
	config, err := cmd.ValidateAndGetConfig(tc.source, tc.destination, tc.jfrogURL, tc.jfrogUser, tc.jfrogPassword)

	// Verify results
	if tc.shouldErr {
		require.Error(t, err, "Expected error for invalid input")
		require.Nil(t, config, "Config should be nil when error occurs")

		return
	}

	require.NoError(t, err, "No error expected for valid input")
	require.NotNil(t, config, "Config should not be nil")

	// Check if flag values take precedence
	if tc.jfrogURL != "" {
		require.Equal(t, tc.jfrogURL, config.URL)
	} else {
		require.Equal(t, tc.envVars["JFROG_URL"], config.URL)
	}

	if tc.jfrogUser != "" {
		require.Equal(t, tc.jfrogUser, config.User)
	} else {
		require.Equal(t, tc.envVars["JFROG_USER"], config.User)
	}

	if tc.jfrogPassword != "" {
		require.Equal(t, tc.jfrogPassword, config.Password)
	} else {
		require.Equal(t, tc.envVars["JFROG_PASSWORD"], config.Password)
	}
}

// Helper functions for environment management.
func saveEnvironment(keys []string) map[string]string {
	env := make(map[string]string)
	for _, key := range keys {
		env[key] = os.Getenv(key)
	}

	return env
}

func restoreEnvironment(env map[string]string) {
	for key, value := range env {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

// mockHTTPClient is a mock HTTP client that returns predefined responses.
type mockHTTPClient struct{}

func (m *mockHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	// For GitHub URLs, return a mock response
	if strings.Contains(req.URL.String(), "github.com") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("mock content"))),
		}, nil
	}

	// For other URLs, return a 404
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}, nil
}

// mockDestination is a mock destination that does nothing.
type mockDestination struct {
	logger *logrus.Logger
}

func newMockDestination(logger *logrus.Logger) *mockDestination {
	return &mockDestination{logger: logger}
}

func (m *mockDestination) Upload(ctx context.Context, artifact *core.Artifact, reader io.Reader) error {
	return nil
}

func (m *mockDestination) Validate() error {
	return nil
}

func (m *mockDestination) Close() error {
	return nil
}

func (m *mockDestination) Exists(ctx context.Context, artifact *core.Artifact) (bool, error) {
	return false, nil
}

func (m *mockDestination) Put(ctx context.Context, artifact *core.Artifact, reader io.Reader) error {
	return nil
}

// setupMockMirrorForTest creates and configures a mock mirror for testing.
func setupMockMirrorForTest(t *testing.T) *mirror.DefaultMirror {
	t.Helper()

	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress log output during tests

	// Setup a mock HTTP client to avoid real network calls
	mockClient := &http.Client{
		Transport: &mockHTTPClient{},
	}

	// Create a GitHub source with the mock client
	source := sources.NewGitHubSource(logger)
	source.SetClient(mockClient)

	// Create a test mirror
	testMirror := mirror.NewDefaultMirror(logger)

	// Add the source and destination
	err := testMirror.AddSource("default", source)
	require.NoError(t, err)

	err = testMirror.AddDestination("default", newMockDestination(logger))
	require.NoError(t, err)

	return testMirror
}

type testCase struct {
	name                   string
	useConfig              bool
	source                 string
	destination            string
	jfrogURL               string
	jfrogUser              string
	jfrogPassword          string
	jfrogPasswordFromStdin bool
	stdinInput             string
	wantErr                bool
	errMsg                 string
}

func TestMirrorCmdRequiredFlags(t *testing.T) {
	configPath, cleanup := setupTestEnvironment(t)
	defer cleanup()

	testCases := []testCase{
		{
			name:    "missing required flags",
			wantErr: true,
			errMsg:  "required flag(s) \"destination\", \"source\" not set",
		},
		{
			name:      "using config file",
			useConfig: true,
			wantErr:   false,
		},
		{
			name:        "source and destination directly",
			source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.tar.gz",
			destination: "http://localhost:8081/artifactory/generic-local/artifact.tar.gz",
			wantErr:     false,
		},
		{
			name:          "with JFrog credentials as flags",
			source:        "https://github.com/owner/repo/releases/download/v1.0.0/artifact.tar.gz",
			destination:   "repo-local",
			jfrogURL:      "http://localhost:8081/artifactory",
			jfrogUser:     "flag-user",
			jfrogPassword: "flag-password",
			wantErr:       false,
		},
		{
			name:                   "with JFrog password from stdin",
			source:                 "https://github.com/owner/repo/releases/download/v1.0.0/artifact.tar.gz",
			destination:            "repo-local",
			jfrogURL:               "http://localhost:8081/artifactory",
			jfrogUser:              "stdin-user",
			jfrogPasswordFromStdin: true,
			stdinInput:             "stdin-password",
			wantErr:                false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, tc, configPath)
		})
	}
}

// setupTestCommand creates and configures a test command with the given test case.
func setupTestCommand(t *testing.T, tc testCase, configPath string) (*cobra.Command, *cmd.MirrorFlags) {
	t.Helper()

	// Setup test mirror
	testMirror := setupMockMirrorForTest(t)

	// Create command
	mirrorCmd := cmd.NewMirrorCmd()
	flags := &cmd.MirrorFlags{
		TestMirror: testMirror,
	}

	// Set flags directly
	setDirectFlags(flags, tc, configPath)

	// Set command arguments
	args := buildCommandArgs(tc, configPath)
	mirrorCmd.SetArgs(args)

	// Set custom RunE function
	mirrorCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return flags.RunE(cmd, args)
	}

	return mirrorCmd, flags
}

// setDirectFlags sets flags directly on the MirrorFlags struct.
func setDirectFlags(flags *cmd.MirrorFlags, tc testCase, configPath string) {
	if tc.useConfig {
		flags.ConfigFile = configPath
	}

	if tc.source != "" {
		flags.Source = tc.source
	}

	if tc.destination != "" {
		flags.Destination = tc.destination
	}

	if tc.jfrogURL != "" {
		flags.JFrogURL = tc.jfrogURL
	}

	if tc.jfrogUser != "" {
		flags.JFrogUser = tc.jfrogUser
	}

	if tc.jfrogPassword != "" {
		flags.JFrogPassword = tc.jfrogPassword
	}

	if tc.jfrogPasswordFromStdin {
		flags.JFrogPasswordFromStdin = true
	}
}

// buildCommandArgs builds the command-line arguments for the test case.
func buildCommandArgs(tc testCase, configPath string) []string {
	args := []string{}

	if tc.useConfig {
		args = append(args, "--config", configPath)
	}

	if tc.source != "" {
		args = append(args, "--source", tc.source)
	}

	if tc.destination != "" {
		args = append(args, "--destination", tc.destination)
	}

	if tc.jfrogURL != "" {
		args = append(args, "--jfrog-url", tc.jfrogURL)
	}

	if tc.jfrogUser != "" {
		args = append(args, "--jfrog-user", tc.jfrogUser)
	}

	if tc.jfrogPassword != "" {
		args = append(args, "--jfrog-password", tc.jfrogPassword)
	}

	if tc.jfrogPasswordFromStdin {
		args = append(args, "--jfrog-password-stdin")
	}

	return args
}

// simulateStdinInput simulates input from stdin for testing.
func simulateStdinInput(t *testing.T, input string) (*os.File, *os.File, func()) {
	r, w, err := os.Pipe()
	require.NoError(t, err)

	oldStdin := os.Stdin
	os.Stdin = r

	_, err = w.WriteString(input + "\n")
	require.NoError(t, err)

	cleanup := func() {
		w.Close()

		os.Stdin = oldStdin
	}

	return r, w, cleanup
}

// validateCommandResult validates the result of command execution.
func validateCommandResult(t *testing.T, err error, tc testCase) {
	t.Helper()

	if tc.wantErr {
		require.Error(t, err)
		require.Contains(t, err.Error(), tc.errMsg)

		return
	}

	require.NoError(t, err)
}

func runTestCase(t *testing.T, tc testCase, configPath string) {
	t.Helper()

	// Setup stdin if needed
	if tc.jfrogPasswordFromStdin {
		_, _, cleanup := simulateStdinInput(t, tc.stdinInput)
		defer cleanup()
	}

	// Setup test command
	mirrorCmd, _ := setupTestCommand(t, tc, configPath)

	// Run command
	err := mirrorCmd.Execute()

	// Validate results
	validateCommandResult(t, err, tc)
}

func TestMirrorCmdFlagRegistration(t *testing.T) {
	// Create a new mirror command
	mirrorCmd := cmd.NewMirrorCmd()

	// Test that all expected flags are registered
	expectedFlags := []string{
		"config",
		"source",
		"destination",
		"from-file",
		"raw",
		"properties",
		"unzip",
		"dry-run",
		"dry-run-mode",
		"jfrog-url",
		"jfrog-user",
		"jfrog-password",
	}

	for _, flagName := range expectedFlags {
		flag := mirrorCmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Flag %s should be registered", flagName)
	}

	// Test that flags are not registered multiple times
	// This will panic if flags are registered twice
	require.NotPanics(t, func() {
		cmd.NewMirrorCmd()
	})
}

func TestMirrorFlagsValidation(t *testing.T) {
	tests := []struct {
		name      string
		flags     *cmd.MirrorFlags
		wantError bool
	}{
		{
			name: "valid dry run mode all",
			flags: &cmd.MirrorFlags{
				DryRun:     true,
				DryRunMode: "all",
			},
			wantError: false,
		},
		{
			name: "valid dry run mode upload",
			flags: &cmd.MirrorFlags{
				DryRun:     true,
				DryRunMode: "upload",
			},
			wantError: false,
		},
		{
			name: "invalid dry run mode",
			flags: &cmd.MirrorFlags{
				DryRun:     true,
				DryRunMode: "invalid",
			},
			wantError: true,
		},
		{
			name: "no dry run - no validation needed",
			flags: &cmd.MirrorFlags{
				DryRun:     false,
				DryRunMode: "invalid",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flags.ValidateDryRunMode()
			if tt.wantError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

// setupTestEnvironment sets up the test environment with config file and environment variables.
func setupTestEnvironment(t *testing.T) (string, func()) {
	t.Helper()

	// Save original environment variables
	origJfrogURL := os.Getenv("JFROG_URL")
	origJfrogUser := os.Getenv("JFROG_USER")
	origJfrogPass := os.Getenv("JFROG_PASSWORD")

	// Set test environment variables
	os.Setenv("JFROG_URL", "http://localhost:8081")
	os.Setenv("JFROG_USER", "admin")
	os.Setenv("JFROG_PASSWORD", "password")

	// Create temporary directory
	tmpDir := t.TempDir()

	// Create config file
	configPath := filepath.Join(tmpDir, "test.yaml")
	configContent := []byte(`
source:
  type: github
  url: https://github.com/owner/repo/releases/download/v1.0.0/artifact.tar.gz
destination:
  type: jfrog
  url: http://localhost:8081/artifactory/generic-local/artifact.tar.gz
  user: admin
  password: password
log_level: info
concurrent: 4
`)
	err := os.WriteFile(configPath, configContent, 0o644)
	require.NoError(t, err)

	cleanup := func() {
		os.Setenv("JFROG_URL", origJfrogURL)
		os.Setenv("JFROG_USER", origJfrogUser)
		os.Setenv("JFROG_PASSWORD", origJfrogPass)
	}

	return configPath, cleanup
}
