package destinations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/destinations"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// testEnv holds test environment configuration.
type testEnv struct {
	logger     *logrus.Logger
	server     *httptest.Server
	config     destinations.JFrogConfig
	artifact   *core.Artifact
	lastPath   string
	lastMethod string
}

// setupTestEnv creates a new test environment.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	env := &testEnv{
		logger: logrus.New(),
	}
	env.logger.SetLevel(logrus.DebugLevel)

	// Setup default config
	env.config = destinations.JFrogConfig{
		URL:      "https://jfrog.example.com",
		User:     "testuser",
		Password: "testpass",
		DestPath: "generic-local",
	}

	// Setup default artifact
	env.artifact = &core.Artifact{
		Name:     "test-artifact",
		Version:  "1.0.0",
		Location: "https://github.com/example/repo/releases/download/v1.0.0/test-artifact.zip",
		Metadata: map[string]string{
			"prop1": "value1",
			"prop2": "value2",
		},
	}

	return env
}

// setupTestServer creates a test server with the given handler.
func (env *testEnv) setupTestServer(handler http.HandlerFunc) {
	env.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env.lastPath = r.URL.Path
		env.lastMethod = r.Method

		// Check authentication
		user, pass, ok := r.BasicAuth()
		if !ok || user != env.config.User || pass != env.config.Password {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		handler(w, r)
	}))

	env.config.URL = env.server.URL
}

// cleanup performs test environment cleanup.
func (env *testEnv) cleanup() {
	if env.server != nil {
		env.server.Close()
	}
}

// getValidationTestCases returns test cases for validation testing.
func getValidationTestCases() []struct {
	name        string
	modifyConf  func(*destinations.JFrogConfig)
	wantErr     bool
	errContains string
} {
	return []struct {
		name        string
		modifyConf  func(*destinations.JFrogConfig)
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid config",
			modifyConf: func(c *destinations.JFrogConfig) {},
			wantErr:    false,
		},
		{
			name:        "empty URL",
			modifyConf:  func(c *destinations.JFrogConfig) { c.URL = "" },
			wantErr:     true,
			errContains: "JFrog URL cannot be empty",
		},
		{
			name:        "invalid URL",
			modifyConf:  func(c *destinations.JFrogConfig) { c.URL = "invalid-url" },
			wantErr:     true,
			errContains: "invalid JFrog URL scheme",
		},
		{
			name:        "empty user",
			modifyConf:  func(c *destinations.JFrogConfig) { c.User = "" },
			wantErr:     true,
			errContains: "JFrog user cannot be empty",
		},
		{
			name:        "empty password",
			modifyConf:  func(c *destinations.JFrogConfig) { c.Password = "" },
			wantErr:     true,
			errContains: "JFrog password cannot be empty",
		},
		{
			name:        "empty destination path",
			modifyConf:  func(c *destinations.JFrogConfig) { c.DestPath = "" },
			wantErr:     true,
			errContains: "JFrog destination path cannot be empty",
		},
	}
}

func TestJFrogDestinationValidate(t *testing.T) {
	env := setupTestEnv(t)
	tests := getValidationTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := env.config
			tt.modifyConf(&config)
			dest := destinations.NewJFrogDestination(config, env.logger)
			err := dest.Validate()

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)

				return
			}

			require.NoError(t, err)
		})
	}
}

// urlHandlingTestCase defines a test case for URL handling.
type urlHandlingTestCase struct {
	name        string
	destPath    string
	raw         bool
	pathChecks  []string
	modifyArt   func(*core.Artifact)
	wantErr     bool
	errContains string
}

// getSimplePathTestCases returns test cases for simple path handling.
func getSimplePathTestCases() []urlHandlingTestCase {
	return []urlHandlingTestCase{
		{
			name:     "simple path with clean structure",
			destPath: "generic-local",
			raw:      false,
			pathChecks: []string{
				"/generic-local/github.com/example/repo/v1.0.0/test-artifact",
				"prop1=value1",
				"prop2=value2",
			},
		},
		{
			name:     "simple path with raw structure",
			destPath: "generic-local",
			raw:      true,
			pathChecks: []string{
				"/generic-local/github.com/example/repo/releases/download/",
				"v1.0.0/test-artifact",
				"prop1=value1",
				"prop2=value2",
			},
		},
	}
}

// getNestedPathTestCases returns test cases for nested path handling.
func getNestedPathTestCases() []urlHandlingTestCase {
	return []urlHandlingTestCase{
		{
			name:     "nested path with clean structure",
			destPath: "generic/sandbox-mirror",
			raw:      false,
			pathChecks: []string{
				"/generic/sandbox-mirror/github.com/example/repo/v1.0.0/test-artifact",
				"prop1=value1",
				"prop2=value2",
			},
		},
		{
			name:     "nested path with raw structure",
			destPath: "generic/sandbox-mirror",
			raw:      true,
			pathChecks: []string{
				"/generic/sandbox-mirror/github.com/example/repo/releases/download/",
				"v1.0.0/test-artifact",
				"prop1=value1",
				"prop2=value2",
			},
		},
	}
}

// getErrorTestCases returns test cases for error handling.
func getErrorTestCases() []urlHandlingTestCase {
	return []urlHandlingTestCase{
		{
			name:        "invalid JFrog URL",
			destPath:    "generic-local",
			modifyArt:   func(a *core.Artifact) {},
			wantErr:     true,
			errContains: "invalid JFrog URL",
		},
	}
}

// getURLHandlingTestCases returns test cases for URL handling testing.
func getURLHandlingTestCases() []urlHandlingTestCase {
	var tests []urlHandlingTestCase
	tests = append(tests, getSimplePathTestCases()...)
	tests = append(tests, getNestedPathTestCases()...)
	tests = append(tests, getErrorTestCases()...)

	return tests
}

func TestJFrogDestinationURLHandling(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer env.cleanup()

	tests := getURLHandlingTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := env.config
			config.DestPath = tt.destPath

			if tt.wantErr {
				config.URL = "://invalid-url"
			}

			artifact := *env.artifact // Create a copy
			if tt.modifyArt != nil {
				tt.modifyArt(&artifact)
			}

			dest := destinations.NewJFrogDestination(config, env.logger)
			_, err := dest.Put(context.Background(), &artifact, strings.NewReader("test content"), tt.raw)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)

				return
			}

			require.NoError(t, err)

			for _, check := range tt.pathChecks {
				require.Contains(t, env.lastPath, check)
			}
		})
	}
}

// getTargetURLTestCases returns test cases for BuildTargetURL testing.
func getTargetURLTestCases() []struct {
	name        string
	modifyConf  func(*destinations.JFrogConfig)
	modifyArt   func(*core.Artifact)
	wantErr     bool
	errContains string
	urlChecks   []string
} {
	return []struct {
		name        string
		modifyConf  func(*destinations.JFrogConfig)
		modifyArt   func(*core.Artifact)
		wantErr     bool
		errContains string
		urlChecks   []string
	}{
		{
			name:       "simple path",
			modifyConf: func(c *destinations.JFrogConfig) {},
			urlChecks: []string{
				"/generic-local/github.com/example/repo/v1.0.0/test-artifact",
			},
		},
		{
			name:       "with properties",
			modifyConf: func(c *destinations.JFrogConfig) { c.DestPath = "generic/sandbox-mirror" },
			urlChecks: []string{
				"/generic/sandbox-mirror/github.com/example/repo/v1.0.0/test-artifact",
				";platform=linux",
				";type=binary",
			},
			modifyArt: func(a *core.Artifact) {
				a.Metadata = map[string]string{
					"type":     "binary",
					"platform": "linux",
				}
			},
		},
		{
			name:        "invalid URL",
			modifyConf:  func(c *destinations.JFrogConfig) { c.URL = "://invalid-url" },
			wantErr:     true,
			errContains: "invalid JFrog URL",
		},
	}
}

func TestJFrogDestinationBuildTargetURL(t *testing.T) {
	env := setupTestEnv(t)
	tests := getTargetURLTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := env.config
			tt.modifyConf(&config)

			artifact := *env.artifact
			if tt.modifyArt != nil {
				tt.modifyArt(&artifact)
			}

			dest := destinations.NewJFrogDestination(config, env.logger)
			url, err := dest.BuildTargetURL(&artifact, false)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)

				return
			}

			require.NoError(t, err)

			urlStr := url.String()
			for _, check := range tt.urlChecks {
				require.Contains(t, urlStr, check)
			}
		})
	}
}

// existsTestCase defines a test case for artifact existence checking.
type existsTestCase struct {
	name      string
	modifyArt func(*core.Artifact)
	wantErr   bool
	exists    bool
}

// getExistsTestCases returns test cases for artifact existence checking.
func getExistsTestCases() []existsTestCase {
	return []existsTestCase{
		{
			name:      "artifact exists",
			modifyArt: func(a *core.Artifact) {},
			exists:    true,
		},
		{
			name: "artifact does not exist",
			modifyArt: func(a *core.Artifact) {
				a.Name = "non-existing"
				a.Location = "https://github.com/example/repo/releases/download/v1.0.0/non-existing.zip"
			},
			exists: false,
		},
		{
			name:      "empty location",
			modifyArt: func(a *core.Artifact) { a.Location = "" },
			wantErr:   true,
			exists:    false,
		},
	}
}

func TestJFrogDestinationExists(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		// The path should match what PathBuilder generates
		if strings.Contains(r.URL.Path, "/generic-local/github.com/example/repo/v1.0.0/test-artifact") {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer env.cleanup()

	tests := getExistsTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := *env.artifact // Create a copy
			tt.modifyArt(&artifact)

			dest := destinations.NewJFrogDestination(env.config, env.logger)
			exists, err := dest.Exists(context.Background(), &artifact, false)

			if tt.wantErr {
				require.Error(t, err)
				require.False(t, exists)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.exists, exists)
		})
	}
}

func TestJFrogDestinationArtifactNameInPath(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer env.cleanup()

	// Test case where artifact name differs from URL filename
	// This simulates the preserve-zip-name scenario
	artifact := &core.Artifact{
		Name:     "bazel_nojdk-8.2.1-windows-x86_64.exe", // Different from URL filename
		Version:  "8.2.1",
		Location: "https://github.com/bazelbuild/bazel/releases/download/8.2.1/bazel_nojdk-8.2.1-windows-x86_64.zip",
		Metadata: map[string]string{
			"type": "binary",
		},
	}

	dest := destinations.NewJFrogDestination(env.config, env.logger)
	_, err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
	require.NoError(t, err)

	// Verify that the path contains the artifact name (.exe) not the URL filename (.zip)
	require.Contains(t, env.lastPath, "bazel_nojdk-8.2.1-windows-x86_64.exe")
	require.NotContains(t, env.lastPath, "bazel_nojdk-8.2.1-windows-x86_64.zip")
	require.Contains(t, env.lastPath, "/generic-local/github.com/bazelbuild/bazel/8.2.1/bazel_nojdk-8.2.1-windows-x86_64.exe")
	require.Contains(t, env.lastPath, "type=binary")
}

// sourcePathStripTestCase defines a test case for source path stripping.
type sourcePathStripTestCase struct {
	name            string
	sourceURL       string
	sourcePathStrip string
	pathChecks      []string
	pathNotChecks   []string
}

// getSourcePathStripTestCases returns test cases for source path stripping.
func getSourcePathStripTestCases() []sourcePathStripTestCase {
	return []sourcePathStripTestCase{
		{
			name:            "JFrog to JFrog - strip staging prefix",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			pathChecks: []string{
				"/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
			pathNotChecks: []string{
				"artifactory.corp.net",
				"staging",
			},
		},
		{
			name:            "JFrog to JFrog - strip with trailing slash",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging",
			pathChecks: []string{
				"/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
			pathNotChecks: []string{
				"artifactory.corp.net",
				"staging",
			},
		},
		{
			name:            "JFrog to JFrog - strip host only",
			sourceURL:       "https://artifactory.corp.net/repo/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net",
			pathChecks: []string{
				"/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
			pathNotChecks: []string{
				"artifactory.corp.net",
			},
		},
		{
			name:            "No stripping when prefix not found",
			sourceURL:       "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			pathChecks: []string{
				"/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
		},
		{
			name:            "Empty strip prefix",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "",
			pathChecks: []string{
				"/generic-local/artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/test-artifact", // When strip prefix is empty, preserve full URL structure
			},
		},
		{
			name:            "Strip from host+path combination",
			sourceURL:       "https://example.com/artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			pathChecks: []string{
				"/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
			pathNotChecks: []string{
				"artifactory.corp.net",
				"staging",
			},
		},
	}
}

func TestJFrogDestinationSourcePathStripping(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer env.cleanup()

	tests := getSourcePathStripTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := env.config
			config.SourcePathStrip = tt.sourcePathStrip

			artifact := &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: tt.sourceURL,
				Metadata: map[string]string{
					"type": "binary",
				},
			}

			dest := destinations.NewJFrogDestination(config, env.logger)
			_, err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
			require.NoError(t, err)

			// Check that expected paths are present
			for _, check := range tt.pathChecks {
				require.Contains(t, env.lastPath, check, "Expected path check failed: %s", check)
			}

			// Check that unwanted paths are not present
			for _, notCheck := range tt.pathNotChecks {
				require.NotContains(t, env.lastPath, notCheck, "Unwanted path found: %s", notCheck)
			}
		})
	}
}

func TestJFrogDestinationSourcePathStrippingWithRawMode(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer env.cleanup()

	config := env.config
	config.SourcePathStrip = "artifactory.corp.net/staging/"

	artifact := &core.Artifact{
		Name:     "test-artifact",
		Version:  "1.0.0",
		Location: "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
		Metadata: map[string]string{
			"type": "binary",
		},
	}

	dest := destinations.NewJFrogDestination(config, env.logger)
	_, err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), true) // raw=true
	require.NoError(t, err)

	// In raw mode with stripping, we should get the raw structure but without the stripped prefix
	require.Contains(t, env.lastPath, "/generic-local/github.com/bazelbuild/bazel/releases/download/7.2.1/test-artifact")
}

func TestJFrogDestinationBuildTargetURLWithSourcePathStrip(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name            string
		sourceURL       string
		sourcePathStrip string
		urlChecks       []string
		urlNotChecks    []string
	}{
		{
			name:            "strip staging prefix",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			urlChecks: []string{
				"/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
			urlNotChecks: []string{
				"artifactory.corp.net",
				"staging",
			},
		},
		{
			name:            "no stripping when prefix not found",
			sourceURL:       "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			urlChecks: []string{
				"/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := env.config
			config.SourcePathStrip = tt.sourcePathStrip

			artifact := &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: tt.sourceURL,
			}

			dest := destinations.NewJFrogDestination(config, env.logger)
			url, err := dest.BuildTargetURL(artifact, false)
			require.NoError(t, err)

			urlStr := url.String()
			for _, check := range tt.urlChecks {
				require.Contains(t, urlStr, check)
			}

			for _, notCheck := range tt.urlNotChecks {
				require.NotContains(t, urlStr, notCheck)
			}
		})
	}
}

func TestJFrogDestination_SourcePathStripping_BugFix(t *testing.T) {
	env := setupTestEnv(t)

	// Test the specific bug case reported by the user
	tests := []struct {
		name            string
		sourceURL       string
		sourcePathStrip string
		expectedPath    string
		description     string
	}{
		{
			name:            "Bug fix: JFrog staging to prod with generic URL",
			sourceURL:       "https://art.corp.net/artifactory/generic/project/staging/some-domain.com/path/bazel.exe",
			sourcePathStrip: "art.corp.net/artifactory/generic/project/staging/",
			expectedPath:    "generic-local/some-domain.com/path/bazel.exe;test=value", // Should preserve domain structure after stripping
			description:     "Should strip staging prefix but preserve domain path structure",
		},
		{
			name:            "Complex path stripping with GitHub releases",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			expectedPath:    "generic-local/github.com/bazelbuild/bazel/v8.2.1/bazel-win.exe;test=value", // Clean GitHub structure
			description:     "Should strip staging prefix and create clean GitHub release structure",
		},
		{
			name:            "Host-only stripping",
			sourceURL:       "https://artifactory.corp.net/repo/github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip: "artifactory.corp.net",
			expectedPath:    "generic-local/github.com/owner/repo/v1.0.0/file.zip;test=value",
			description:     "Should strip host and preserve GitHub structure",
		},
		{
			name:            "No stripping when prefix not found",
			sourceURL:       "https://github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			expectedPath:    "generic-local/github.com/owner/repo/v1.0.0/file.zip;test=value",
			description:     "Should not strip when prefix not found in URL",
		},
		{
			name:            "Generic URL stripping",
			sourceURL:       "https://artifactory.corp.net/staging/some-host.com/path/to/file.zip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			expectedPath:    "generic-local/some-host.com/path/to/file.zip;test=value", // Generic processor preserves structure
			description:     "Should strip prefix from generic URLs but preserve remaining structure",
		},
		{
			name:            "BCR URL preserves structure",
			sourceURL:       "https://bcr.bazel.build/modules/hermetic_cc_toolchain/4.0.0/source.json",
			sourcePathStrip: "",
			expectedPath:    "generic-local/bcr.bazel.build/modules/hermetic_cc_toolchain/4.0.0/source.json;test=value", // BCR URLs should preserve full structure
			description:     "BCR URLs should preserve full path structure as reported in the original bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create destination with source path strip configuration
			config := env.config
			config.SourcePathStrip = tt.sourcePathStrip
			dest := destinations.NewJFrogDestination(config, env.logger)

			// Create test artifact
			artifact := &core.Artifact{
				Name:     filepath.Base(tt.sourceURL),
				Location: tt.sourceURL,
				Metadata: map[string]string{
					"test": "value",
				},
			}

			// Build target URL (this internally calls stripSourcePath)
			targetURL, err := dest.BuildTargetURL(artifact, false) // raw=false for clean structure
			require.NoError(t, err, tt.description)

			// Verify the path is correct (targetURL.String() includes the full URL with matrix params)
			fullURL := targetURL.String()
			expectedFullURL := "https://jfrog.example.com/" + tt.expectedPath
			require.Equal(t, expectedFullURL, fullURL, tt.description)

			// Log for debugging
			t.Logf("Test: %s", tt.name)
			t.Logf("Source URL: %s", tt.sourceURL)
			t.Logf("Strip prefix: %s", tt.sourcePathStrip)
			t.Logf("Expected full URL: %s", expectedFullURL)
			t.Logf("Actual full URL: %s", fullURL)
		})
	}
}
