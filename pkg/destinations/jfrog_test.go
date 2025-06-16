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

// Consolidated test environment setup.
type testEnv struct {
	logger     *logrus.Logger
	server     *httptest.Server
	config     destinations.JFrogConfig
	artifact   *core.Artifact
	lastPath   string
	lastMethod string
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	return &testEnv{
		logger: logger,
		config: destinations.JFrogConfig{
			URL:      "https://jfrog.example.com",
			User:     "testuser",
			Password: "testpass",
			DestPath: "generic-local",
		},
		artifact: &core.Artifact{
			Name:     "test-artifact",
			Version:  "1.0.0",
			Location: "https://github.com/example/repo/releases/download/v1.0.0/test-artifact.zip",
			Metadata: map[string]string{
				"prop1": "value1",
				"prop2": "value2",
			},
		},
	}
}

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

func (env *testEnv) cleanup() {
	if env.server != nil {
		env.server.Close()
	}
}

// Core validation tests - essential for config safety.
func TestJFrogDestinationValidate(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
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

// Consolidated URL building test - covers all URL construction scenarios.
func TestJFrogDestinationURLBuilding(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer env.cleanup()

	tests := []struct {
		name        string
		modifyConf  func(*destinations.JFrogConfig)
		modifyArt   func(*core.Artifact)
		raw         bool
		wantErr     bool
		errContains string
		pathChecks  []string
	}{
		{
			name:       "basic clean structure",
			modifyConf: func(c *destinations.JFrogConfig) {},
			pathChecks: []string{
				"/artifactory/generic-local/github.com/example/repo/v1.0.0/test-artifact",
				"prop1=value1",
				"prop2=value2",
			},
		},
		{
			name:       "basic raw structure",
			modifyConf: func(c *destinations.JFrogConfig) {},
			raw:        true,
			pathChecks: []string{
				"/artifactory/generic-local/github.com/example/repo/releases/download/",
				"v1.0.0/test-artifact",
				"prop1=value1",
			},
		},
		{
			name:       "nested destination path",
			modifyConf: func(c *destinations.JFrogConfig) { c.DestPath = "generic/sandbox-mirror" },
			pathChecks: []string{
				"/artifactory/generic/sandbox-mirror/github.com/example/repo/v1.0.0/test-artifact",
			},
		},
		{
			name:       "artifact with spaces",
			modifyConf: func(c *destinations.JFrogConfig) { c.DestPath = "my-repo" },
			modifyArt: func(a *core.Artifact) {
				a.Name = "my app 1.0.dmg"
				a.Location = "https://files.example.org/dist/my app 1.0.dmg"
			},
			pathChecks: []string{
				"/artifactory/my-repo/files.example.org/dist/my app 1.0.dmg",
			},
		},
		{
			name:        "invalid URL normalization",
			modifyConf:  func(c *destinations.JFrogConfig) { c.URL = "://invalid-url" },
			wantErr:     true,
			errContains: "failed to normalize JFrog URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := env.config
			tt.modifyConf(&config)

			artifact := *env.artifact
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

// Essential artifact existence checking.
func TestJFrogDestinationExists(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		if strings.Contains(r.URL.Path, "/artifactory/generic-local/github.com/example/repo/v1.0.0/test-artifact") {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer env.cleanup()

	tests := []struct {
		name      string
		modifyArt func(*core.Artifact)
		wantErr   bool
		exists    bool
	}{
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := *env.artifact
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

// Comprehensive source path stripping test.
func TestJFrogDestinationSourcePathStripping(t *testing.T) {
	env := setupTestEnv(t)
	env.setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer env.cleanup()

	tests := []struct {
		name            string
		sourceURL       string
		sourcePathStrip string
		raw             bool
		pathChecks      []string
		pathNotChecks   []string
	}{
		{
			name:            "JFrog to JFrog - strip staging prefix",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			pathChecks: []string{
				"/artifactory/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
			pathNotChecks: []string{
				"artifactory.corp.net",
				"staging",
			},
		},
		{
			name:            "strip with raw mode",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			raw:             true,
			pathChecks: []string{
				"/artifactory/generic-local/github.com/bazelbuild/bazel/releases/download/7.2.1/test-artifact",
			},
			pathNotChecks: []string{
				"artifactory.corp.net",
				"staging",
			},
		},
		{
			name:            "no stripping when prefix not found",
			sourceURL:       "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			pathChecks: []string{
				"/artifactory/generic-local/github.com/bazelbuild/bazel/7.2.1/test-artifact",
			},
		},
		{
			name:            "empty strip prefix preserves full structure",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-win.exe",
			sourcePathStrip: "",
			pathChecks: []string{
				"/artifactory/generic-local/artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/7.2.1/test-artifact",
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
				Metadata: map[string]string{
					"type": "binary",
				},
			}

			dest := destinations.NewJFrogDestination(config, env.logger)
			_, err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), tt.raw)
			require.NoError(t, err)

			for _, check := range tt.pathChecks {
				require.Contains(t, env.lastPath, check,
					"Expected path check failed: %s", check)
			}

			for _, notCheck := range tt.pathNotChecks {
				require.NotContains(t, env.lastPath, notCheck,
					"Unwanted path found: %s", notCheck)
			}
		})
	}
}

// URL normalization test - critical for our new feature.
func TestJFrogDestination_URLNormalization(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name          string
		inputURL      string
		expectedError string
		description   string
	}{
		{
			name:        "URL without /artifactory - should add it",
			inputURL:    "http://localhost:8082",
			description: "Should automatically add /artifactory to URLs without it",
		},
		{
			name:        "URL with /artifactory - should preserve it",
			inputURL:    "http://localhost:8082/artifactory",
			description: "Should preserve existing /artifactory path",
		},
		{
			name:        "URL with custom path containing /artifactory",
			inputURL:    "http://localhost:8082/my-custom/artifactory",
			description: "Should preserve custom paths that contain /artifactory",
		},
		{
			name:        "URL with custom path not containing /artifactory",
			inputURL:    "http://localhost:8082/my-custom/path",
			description: "Should append /artifactory to custom paths",
		},
		{
			name:        "HTTPS URL without /artifactory",
			inputURL:    "https://company.jfrog.io",
			description: "Should work with HTTPS URLs",
		},
		{
			name:          "Invalid URL - empty",
			inputURL:      "",
			expectedError: "URL cannot be empty",
			description:   "Should reject empty URLs",
		},
		{
			name:          "Invalid URL - no scheme",
			inputURL:      "localhost:8082",
			expectedError: "invalid URL scheme",
			description:   "Should reject URLs without scheme",
		},
		{
			name:          "Invalid URL - bad scheme",
			inputURL:      "ftp://localhost:8082",
			expectedError: "invalid URL scheme",
			description:   "Should reject URLs with invalid schemes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := destinations.JFrogConfig{
				URL:      tt.inputURL,
				User:     "test",
				Password: "test",
				DestPath: "generic-local",
			}

			dest := destinations.NewJFrogDestination(config, logger)

			artifact := &core.Artifact{
				Name:     "test.zip",
				Location: "https://github.com/owner/repo/releases/download/v1.0.0/test.zip",
				Metadata: map[string]string{},
			}

			_, err := dest.BuildTargetURL(artifact, false)

			if tt.expectedError != "" {
				require.Error(t, err, tt.description)
				require.Contains(t, err.Error(), tt.expectedError, tt.description)

				return
			}

			require.NoError(t, err, tt.description)
		})
	}
}

// Verification test for essential regression prevention.
func TestJFrogDestination_BugFixVerification(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name              string
		jfrogURL          string
		destPath          string
		artifactLocation  string
		sourcePathStrip   string
		expectedTargetURL string
		description       string
	}{
		{
			name:              "Original bug scenario - URL with /artifactory preserved",
			jfrogURL:          "http://localhost:8082/artifactory",
			destPath:          "generic-local",
			artifactLocation:  "https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip",
			expectedTargetURL: "http://localhost:8082/artifactory/generic-local/github.com/bazelbuild/bazel/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip",
			description:       "The original bug scenario - should preserve /artifactory in final URL",
		},
		{
			name:              "URL without /artifactory gets it added",
			jfrogURL:          "http://localhost:8082",
			destPath:          "generic-local",
			artifactLocation:  "https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip",
			expectedTargetURL: "http://localhost:8082/artifactory/generic-local/github.com/bazelbuild/bazel/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip",
			description:       "URL without /artifactory should get it added automatically",
		},
		{
			name:              "Source path stripping with GitHub preservation",
			jfrogURL:          "https://jfrog.example.com/artifactory",
			destPath:          "generic-local",
			artifactLocation:  "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
			sourcePathStrip:   "artifactory.corp.net/staging/",
			expectedTargetURL: "https://jfrog.example.com/artifactory/generic-local/github.com/bazelbuild/bazel/v8.2.1/bazel-win.exe",
			description:       "Should strip staging prefix and preserve clean GitHub structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := destinations.JFrogConfig{
				URL:             tt.jfrogURL,
				User:            "developer",
				Password:        "password",
				DestPath:        tt.destPath,
				SourcePathStrip: tt.sourcePathStrip,
			}

			dest := destinations.NewJFrogDestination(config, logger)

			artifact := &core.Artifact{
				Name:     filepath.Base(tt.artifactLocation),
				Location: tt.artifactLocation,
				Metadata: map[string]string{},
			}

			targetURL, err := dest.BuildTargetURL(artifact, false)
			require.NoError(t, err, tt.description)

			actualURL := targetURL.String()
			require.Equal(t, tt.expectedTargetURL, actualURL, tt.description)

			// Verify /artifactory is present and not duplicated
			require.Contains(t, actualURL, "/artifactory/", "Final URL must contain /artifactory/")
			require.NotEqual(t, 2, strings.Count(actualURL, "/artifactory/"), "Should not have duplicate /artifactory/")
		})
	}
}
