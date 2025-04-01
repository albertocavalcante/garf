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

// createTestLogger creates a new logger for testing.
func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	return logger
}

func TestJFrogDestinationValidate(t *testing.T) {
	logger := createTestLogger()

	// Valid config
	validConfig := destinations.JFrogConfig{
		URL:      "https://jfrog.example.com",
		User:     "testuser",
		Password: "testpass",
		DestPath: "generic-local",
	}

	dest := destinations.NewJFrogDestination(validConfig, logger)
	err := dest.Validate()
	require.NoError(t, err)

	// Empty URL
	invalidConfig := validConfig
	invalidConfig.URL = ""
	dest = destinations.NewJFrogDestination(invalidConfig, logger)
	err = dest.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "JFrog URL cannot be empty")

	// Invalid URL
	invalidConfig = validConfig
	invalidConfig.URL = "invalid-url"
	dest = destinations.NewJFrogDestination(invalidConfig, logger)
	err = dest.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid JFrog URL scheme")

	// Empty user
	invalidConfig = validConfig
	invalidConfig.User = ""
	dest = destinations.NewJFrogDestination(invalidConfig, logger)
	err = dest.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "JFrog user cannot be empty")

	// Empty password
	invalidConfig = validConfig
	invalidConfig.Password = ""
	dest = destinations.NewJFrogDestination(invalidConfig, logger)
	err = dest.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "JFrog password cannot be empty")

	// Empty destination path
	invalidConfig = validConfig
	invalidConfig.DestPath = ""
	dest = destinations.NewJFrogDestination(invalidConfig, logger)
	err = dest.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "JFrog destination path cannot be empty")
}

func TestJFrogDestinationPut(t *testing.T) {
	logger := createTestLogger()

	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		// For PUT requests
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	config := destinations.JFrogConfig{
		URL:      server.URL,
		User:     "testuser",
		Password: "testpass",
		DestPath: "generic-local",
	}

	artifact := &core.Artifact{
		Name:     "test-artifact",
		Version:  "1.0.0",
		Location: "test-repo/test-artifact",
	}

	// Test successful upload
	dest := destinations.NewJFrogDestination(config, logger)
	err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
	require.NoError(t, err)

	// Test empty location
	artifact.Location = ""
	err = dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
	require.Error(t, err)
}

func TestJFrogDestinationExists(t *testing.T) {
	logger := createTestLogger()

	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		// For HEAD requests
		if r.Method == http.MethodHead {
			// Return OK for test-artifact, not for non-existing
			if r.URL.Path == "/artifactory/generic-local/test-artifact" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	config := destinations.JFrogConfig{
		URL:      server.URL,
		User:     "testuser",
		Password: "testpass",
		DestPath: "generic-local",
	}

	// Test artifact exists
	artifact := &core.Artifact{
		Name:     "test-artifact",
		Version:  "1.0.0",
		Location: "test-repo/test-artifact",
	}

	dest := destinations.NewJFrogDestination(config, logger)
	exists, err := dest.Exists(context.Background(), artifact, false)
	require.NoError(t, err)
	require.True(t, exists)

	// Test artifact does not exist
	artifact = &core.Artifact{
		Name:     "non-existing",
		Version:  "1.0.0",
		Location: "test-repo/non-existing",
	}

	exists, err = dest.Exists(context.Background(), artifact, false)
	require.NoError(t, err)
	require.False(t, exists)

	// Test empty location
	artifact.Location = ""
	exists, err = dest.Exists(context.Background(), artifact, false)
	require.Error(t, err)
	require.False(t, exists)
}

func TestJFrogDestinationURLHandlingSimplePath(t *testing.T) {
	logger := createTestLogger()

	config := destinations.JFrogConfig{
		URL:      "https://jfrog.example.com",
		User:     "testuser",
		Password: "testpass",
		DestPath: "generic-local",
	}
	artifact := &core.Artifact{
		Name:     "test-artifact",
		Version:  "1.0.0",
		Location: "https://github.com/example/repo/releases/download/v1.0.0/test-artifact.zip",
	}

	// Create a test server that captures the request URL
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config.URL = server.URL
	dest := destinations.NewJFrogDestination(config, logger)

	// Test with raw=false (clean structure)
	err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
	require.NoError(t, err)
	// Check partial path to avoid linter line length issues
	require.Contains(t, capturedPath, "/artifactory/generic-local/github.com/example/repo/v1.0.0/")
	require.Contains(t, capturedPath, "test-artifact.zip")

	// Test with raw=true (preserve full path)
	err = dest.Put(context.Background(), artifact, strings.NewReader("test content"), true)
	require.NoError(t, err)
	// Check partial path to avoid linter line length issues
	require.Contains(t, capturedPath, "/artifactory/generic-local/github.com/example/repo/releases/download/")
	require.Contains(t, capturedPath, "test-artifact.zip")
}

func TestJFrogDestinationURLHandlingNestedPath(t *testing.T) {
	logger := createTestLogger()

	config := destinations.JFrogConfig{
		URL:      "https://jfrog.example.com",
		User:     "testuser",
		Password: "testpass",
		DestPath: "generic/sandbox-mirror",
	}
	artifact := &core.Artifact{
		Name:     "test-artifact",
		Version:  "1.0.0",
		Location: "https://github.com/example/repo/releases/download/v1.0.0/test-artifact.zip",
		Metadata: map[string]string{
			"prop1": "value1",
			"prop2": "value2",
		},
	}

	// Create a test server that captures the request URL
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config.URL = server.URL
	dest := destinations.NewJFrogDestination(config, logger)

	// Test with raw=false
	err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
	require.NoError(t, err)
	// Check partial path to avoid linter line length issues
	require.Contains(t, capturedPath, "/artifactory/generic/sandbox-mirror/github.com/example/repo/v1.0.0/")
	require.Contains(t, capturedPath, "test-artifact.zip")
	require.Contains(t, capturedPath, "prop1=value1")
	require.Contains(t, capturedPath, "prop2=value2")

	// Test with raw=true
	err = dest.Put(context.Background(), artifact, strings.NewReader("test content"), true)
	require.NoError(t, err)
	// Check partial path to avoid linter line length issues
	require.Contains(t, capturedPath, "/artifactory/generic/sandbox-mirror/github.com/example/repo/releases/download/")
	require.Contains(t, capturedPath, "test-artifact.zip")
	require.Contains(t, capturedPath, "prop1=value1")
	require.Contains(t, capturedPath, "prop2=value2")
}

func TestJFrogDestinationURLHandlingInvalidURLs(t *testing.T) {
	logger := createTestLogger()

	testCases := []struct {
		name          string
		config        destinations.JFrogConfig
		artifact      *core.Artifact
		errorContains string
	}{
		{
			name: "invalid JFrog URL",
			config: destinations.JFrogConfig{
				URL:      "://invalid-url",
				User:     "testuser",
				Password: "testpass",
				DestPath: "generic-local",
			},
			artifact: &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: "https://github.com/example/repo/releases/download/v1.0.0/test-artifact.zip",
			},
			errorContains: "invalid JFrog URL",
		},
		{
			name: "invalid source location",
			config: destinations.JFrogConfig{
				URL:      "https://jfrog.example.com",
				User:     "testuser",
				Password: "testpass",
				DestPath: "generic-local",
			},
			artifact: &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: "://invalid-url",
			},
			errorContains: "invalid source location",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dest := destinations.NewJFrogDestination(tc.config, logger)
			err := dest.Put(context.Background(), tc.artifact, strings.NewReader("test content"), false)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errorContains)
		})
	}
}

func TestJFrogDestinationBuildTargetURLSimplePath(t *testing.T) {
	logger := logrus.New()

	config := destinations.JFrogConfig{
		URL:      "https://jfrog.example.com",
		User:     "user",
		Password: "pass",
		DestPath: "generic-local",
	}
	artifact := &core.Artifact{
		Name:     "test.txt",
		Location: "https://example.com/test.txt",
	}

	dest := destinations.NewJFrogDestination(config, logger)
	url, err := dest.BuildTargetURL(artifact, false)

	require.NoError(t, err)
	require.Contains(t, url.String(), "https://jfrog.example.com/artifactory/generic-local/test.txt")
}

func TestJFrogDestinationBuildTargetURLWithProperties(t *testing.T) {
	logger := logrus.New()

	config := destinations.JFrogConfig{
		URL:      "https://jfrog.example.com",
		User:     "user",
		Password: "pass",
		DestPath: "generic/sandbox-mirror",
	}
	artifact := &core.Artifact{
		Name:     "test.txt",
		Location: "https://example.com/test.txt",
		Metadata: map[string]string{
			"type":     "binary",
			"platform": "linux",
		},
	}

	dest := destinations.NewJFrogDestination(config, logger)
	url, err := dest.BuildTargetURL(artifact, false)

	require.NoError(t, err)

	urlStr := url.String()
	require.Contains(t, urlStr, "https://jfrog.example.com/artifactory/generic/sandbox-mirror/test.txt")
	require.Contains(t, urlStr, "type=binary")
	require.Contains(t, urlStr, "platform=linux")
}

func TestJFrogDestinationBuildTargetURLInvalidURL(t *testing.T) {
	logger := logrus.New()

	config := destinations.JFrogConfig{
		URL:      "://invalid-url",
		User:     "user",
		Password: "pass",
		DestPath: "generic-local",
	}
	artifact := &core.Artifact{
		Name:     "test.txt",
		Location: "https://example.com/test.txt",
	}

	dest := destinations.NewJFrogDestination(config, logger)
	_, err := dest.BuildTargetURL(artifact, false)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid JFrog URL")
}
