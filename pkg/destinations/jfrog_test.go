package destinations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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
				"/generic-local/github.com/example/repo/v1.0.0/test-artifact.zip",
			},
		},
		{
			name:     "simple path with raw structure",
			destPath: "generic-local",
			raw:      true,
			pathChecks: []string{
				"/generic-local/github.com/example/repo/releases/download/",
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
				"/generic/sandbox-mirror/github.com/example/repo/v1.0.0/test-artifact.zip",
			},
		},
		{
			name:     "nested path with raw structure",
			destPath: "generic/sandbox-mirror",
			raw:      true,
			pathChecks: []string{
				"/generic/sandbox-mirror/github.com/example/repo/releases/download/",
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
			err := dest.Put(context.Background(), &artifact, strings.NewReader("test content"), tt.raw)

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
				"/generic-local/github.com/example/repo/v1.0.0/test-artifact.zip",
			},
		},
		{
			name:       "with properties",
			modifyConf: func(c *destinations.JFrogConfig) { c.DestPath = "generic/sandbox-mirror" },
			urlChecks: []string{
				"/generic/sandbox-mirror/github.com/example/repo/v1.0.0/test-artifact.zip",
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
		if strings.Contains(r.URL.Path, "/generic-local/github.com/example/repo/v1.0.0/test-artifact.zip") {
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
