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

// testEnv holds test environment configuration.
type testEnv struct {
	configPath string
	cleanup    func()
	logger     *logrus.Logger
	mirror     *mirror.DefaultMirror
}

// setupTestEnv creates a new test environment with all necessary components.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Save and set environment variables
	envVars := map[string]string{
		"JFROG_URL":      "http://localhost:8081",
		"JFROG_USER":     "admin",
		"JFROG_PASSWORD": "password",
	}
	origEnv := saveEnvironment(envVars)

	// Create logger
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress log output during tests

	// Setup mock mirror
	testMirror := setupMockMirror(logger)

	// Create config file
	configPath := createTestConfig(t)

	cleanup := func() {
		restoreEnvironment(origEnv)
	}

	return &testEnv{
		configPath: configPath,
		cleanup:    cleanup,
		logger:     logger,
		mirror:     testMirror,
	}
}

// createTestConfig creates a test configuration file.
func createTestConfig(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
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

	return configPath
}

// setupMockMirror creates a mock mirror instance for testing.
func setupMockMirror(logger *logrus.Logger) *mirror.DefaultMirror {
	mockClient := &http.Client{Transport: &mockHTTPClient{}}
	source := sources.NewGitHubSource(logger)
	source.SetClient(mockClient)

	testMirror := mirror.NewDefaultMirror(logger)
	_ = testMirror.AddSource("default", source)
	_ = testMirror.AddDestination("default", newMockDestination(logger))

	return testMirror
}

// testCase represents a generic test case structure.
type testCase struct {
	name                   string
	useConfig              bool
	source                 string
	destination            string
	jfrogURL               string
	jfrogUser              string
	jfrogPassword          string
	jfrogPasswordFromStdin bool
	wantErr                bool
	errMsg                 string
}

// commandConfig holds configuration for command setup.
type commandConfig struct {
	flags    *cmd.MirrorFlags
	testCase testCase
}

// setupCommand creates and configures a test command.
func setupCommand(t *testing.T, cfg commandConfig, env *testEnv) *cobra.Command {
	t.Helper()

	mirrorCmd := cmd.NewMirrorCmd()

	flags := cfg.flags
	if flags == nil {
		flags = &cmd.MirrorFlags{TestMirror: env.mirror}
	}

	// Set flags
	setFlags(flags, cfg.testCase, env.configPath)

	// Set command arguments
	args := buildArgs(cfg.testCase, env.configPath)
	mirrorCmd.SetArgs(args)

	// Set custom RunE function
	mirrorCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return flags.RunE(cmd, args)
	}

	return mirrorCmd
}

// setFlags sets flags on the MirrorFlags struct.
func setFlags(flags *cmd.MirrorFlags, tc testCase, configPath string) {
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

	flags.JFrogPasswordFromStdin = tc.jfrogPasswordFromStdin
}

// buildArgs builds command-line arguments.
func buildArgs(tc testCase, configPath string) []string {
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

// Helper functions for environment management.
func saveEnvironment(vars map[string]string) map[string]string {
	env := make(map[string]string)
	for key := range vars {
		env[key] = os.Getenv(key)
		os.Setenv(key, vars[key])
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
	if strings.Contains(req.URL.String(), "github.com") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("mock content"))),
		}, nil
	}

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

func (m *mockDestination) Exists(ctx context.Context, artifact *core.Artifact, raw bool) (bool, error) {
	return false, nil
}

func (m *mockDestination) Put(ctx context.Context, artifact *core.Artifact, reader io.Reader, raw bool) error {
	return nil
}

// Test functions.
func TestMirrorCmdRequiredFlags(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := setupCommand(t, commandConfig{testCase: tc}, env)
			err := cmd.Execute()

			if tc.wantErr {
				require.Error(t, err)

				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestMirrorCmdFlagRegistration(t *testing.T) {
	expectedFlags := []string{
		"config", "source", "destination", "from-file",
		"raw", "properties", "unzip", "dry-run",
		"dry-run-mode", "source-path-strip", "jfrog-url", "jfrog-user", "jfrog-password",
	}

	mirrorCmd := cmd.NewMirrorCmd()
	for _, flagName := range expectedFlags {
		flag := mirrorCmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Flag %s should be registered", flagName)
	}

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
		},
		{
			name: "valid dry run mode upload",
			flags: &cmd.MirrorFlags{
				DryRun:     true,
				DryRunMode: "upload",
			},
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

func TestGetConfig(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	testCases := []struct {
		name         string
		jfrogURL     string
		destination  string
		envVars      map[string]string
		expectedURL  string
		expectedPath string
	}{
		{
			name:         "From flag with complete URL",
			jfrogURL:     "https://example.jfrog.io/artifactory",
			destination:  "repo-local",
			expectedURL:  "https://example.jfrog.io/artifactory",
			expectedPath: "repo-local",
		},
		{
			name:         "URL normalization - adding scheme",
			jfrogURL:     "example.jfrog.io",
			destination:  "repo-local",
			expectedURL:  "http://example.jfrog.io/artifactory",
			expectedPath: "repo-local",
		},
		{
			name:         "From env var",
			destination:  "repo-local",
			envVars:      map[string]string{"JFROG_URL": "https://example.jfrog.io/artifactory"},
			expectedURL:  "https://example.jfrog.io/artifactory",
			expectedPath: "repo-local",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVars != nil {
				origEnv := saveEnvironment(tc.envVars)
				defer restoreEnvironment(origEnv)
			}

			flags := &cmd.MirrorFlags{
				Source:        "https://github.com/example/repo/releases/download/v1.0/file.zip",
				Destination:   tc.destination,
				JFrogURL:      tc.jfrogURL,
				JFrogUser:     "user",
				JFrogPassword: "password",
			}

			config, err := cmd.ValidateAndGetConfig(flags)
			require.NoError(t, err)
			require.Equal(t, tc.expectedURL, config.Destination.URL)
			require.Equal(t, tc.expectedPath, config.Destination.DestPath)
		})
	}
}

func TestSourcePathStripValidation(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	testCases := []struct {
		name            string
		sourcePathStrip string
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "valid path strip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			wantErr:         false,
		},
		{
			name:            "empty path strip",
			sourcePathStrip: "",
			wantErr:         false,
		},
		{
			name:            "path traversal attack",
			sourcePathStrip: "../../malicious",
			wantErr:         true,
			errMsg:          "source path strip cannot contain '..' for security reasons",
		},
		{
			name:            "https scheme",
			sourcePathStrip: "https://malicious.com/",
			wantErr:         true,
			errMsg:          "source path strip should not include the URL scheme",
		},
		{
			name:            "http scheme",
			sourcePathStrip: "http://malicious.com/",
			wantErr:         true,
			errMsg:          "source path strip should not include the URL scheme",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flags := &cmd.MirrorFlags{
				Source:          "https://github.com/example/repo/releases/download/v1.0/file.zip",
				Destination:     "test-repo",
				SourcePathStrip: tc.sourcePathStrip,
				JFrogURL:        "https://test.jfrog.io/artifactory",
				JFrogUser:       "user",
				JFrogPassword:   "password",
			}

			_, err := cmd.ValidateAndGetConfig(flags)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
