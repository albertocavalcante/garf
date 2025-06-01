package destinations

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

// JFrogTestCase holds common fields for JFrog destination test cases.
type JFrogTestCase struct {
	Name            string
	ArtifactName    string // Add this field to specify artifact name explicitly
	SourceURL       string
	SourcePathStrip string
	DestPath        string
	ExpectedPath    string
	Description     string
	Raw             bool // For tests that vary the raw flag
	Metadata        map[string]string
	WantErr         bool
	ErrContains     string
	PathChecks      []string // For asserting parts of the path
	PathNotChecks   []string // For asserting parts NOT in the path
}

// NewTestLogger creates a new logger configured for tests.
func NewTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel) // Or a configurable level

	return logger
}

// TestServerEnv holds the environment for a test server.
type TestServerEnv struct {
	Server      *httptest.Server
	RequestPath string
	LastMethod  string
	Config      JFrogConfig // The config pointing to this server
	Logger      *logrus.Logger
}

// SetupTestServer creates and configures a new HTTP test server.
// The returned JFrogConfig will have its URL pointing to this server.
func SetupTestServer(t *testing.T, baseConfig JFrogConfig, handler http.HandlerFunc) *TestServerEnv {
	t.Helper()

	env := &TestServerEnv{
		Logger: NewTestLogger(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env.RequestPath = r.URL.Path
		env.LastMethod = r.Method

		// Optionally remove matrix parameters from RequestPath for simpler assertions
		if idx := strings.Index(env.RequestPath, ";"); idx != -1 {
			env.RequestPath = env.RequestPath[:idx]
		}

		// Basic Auth check (optional, can be part of the provided handler if more complex)
		user, pass, ok := r.BasicAuth()
		if !ok || user != baseConfig.User || pass != baseConfig.Password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		if handler != nil {
			handler(w, r)
		} else {
			w.WriteHeader(http.StatusCreated) // Default success for Put/Exists
		}
	}))

	env.Server = server
	env.Config = baseConfig
	env.Config.URL = server.URL // Update config to use the test server

	return env
}

// DefaultJFrogConfig returns a default JFrog configuration for tests.
func DefaultJFrogConfig() JFrogConfig {
	return JFrogConfig{
		URL:      "https://jfrog.example.com", // Will be overridden by test server
		User:     "testuser",
		Password: "testpass",
		DestPath: "generic-local",
	}
}

// DefaultArtifact returns a default core.Artifact for tests.
// Allow metadata to be customized.
func DefaultArtifact(t *testing.T, name, location string, metadata map[string]string) *core.Artifact {
	t.Helper()
	// If metadata is explicitly nil, it should remain nil for the artifact.
	// Only provide a default if the map is, for example, uninitialized by a test case that doesn't care about metadata explicitly.
	// However, for this specific scenario, if nil is passed, we want nil in the artifact.
	// The test case in jfrog_source_strip_bug_test.go passes nil to signify no metadata.
	// So, we will not default metadata here if it's nil.
	return &core.Artifact{
		Name:     name,
		Version:  "1.0.0", // Or make configurable
		Location: location,
		Metadata: metadata, // Use the passed metadata directly
	}
}

// StandardGitHubArtURL is a common GitHub URL for tests.
const StandardGitHubArtURL = "https://github.com/org/repo/releases/download/v1.0.0/artifact.zip"

// StandardBCRArtURL is a common BCR URL for tests.
const StandardBCRArtURL = "https://bcr.bazel.build/modules/lib/v1.2.3/source.json"

// StandardGenericArtURL is a common generic URL for tests.
const StandardGenericArtURL = "https://myget.org/F/feed/package/1.0.0"

// ExtractArtifactNameFromURL safely extracts the artifact name from a URL or provides a default.
func ExtractArtifactNameFromURL(sourceURL string) string {
	if sourceURL == "" {
		return "artifact"
	}

	if slash := strings.LastIndex(sourceURL, "/"); slash >= 0 {
		return sourceURL[slash+1:]
	}

	return "artifact" // Fallback default
}

// GetArtifactNameFromTestCase returns the artifact name from a test case, extracting from URL if not specified.
func GetArtifactNameFromTestCase(tc JFrogTestCase) string {
	if tc.ArtifactName != "" {
		return tc.ArtifactName
	}

	return ExtractArtifactNameFromURL(tc.SourceURL)
}
