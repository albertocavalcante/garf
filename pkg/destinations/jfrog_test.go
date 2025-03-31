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

func setupTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		switch r.Method {
		case http.MethodPut:
			// Check if the URL contains matrix parameters
			if r.URL.Path == "/artifactory/test-repo/test-artifact;prop1=value1;prop2=value2" {
				w.WriteHeader(http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		case http.MethodHead:
			if r.URL.Path == "/artifactory/test-repo/test-artifact" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

func createValidConfig(server *httptest.Server) destinations.JFrogConfig {
	return destinations.JFrogConfig{
		URL:      server.URL,
		User:     "testuser",
		Password: "testpass",
	}
}

func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	return logger
}

func TestJFrogDestinationValidate(t *testing.T) {
	logger := createTestLogger()

	t.Run("valid config", func(t *testing.T) {
		config := destinations.JFrogConfig{
			URL:      "https://jfrog.example.com",
			User:     "testuser",
			Password: "testpass",
		}
		dest := destinations.NewJFrogDestination(config, logger)
		require.NoError(t, dest.Validate())
	})

	t.Run("missing URL", func(t *testing.T) {
		config := destinations.JFrogConfig{
			User:     "testuser",
			Password: "testpass",
		}
		dest := destinations.NewJFrogDestination(config, logger)
		require.Error(t, dest.Validate())
	})

	t.Run("missing user", func(t *testing.T) {
		config := destinations.JFrogConfig{
			URL:      "https://jfrog.example.com",
			Password: "testpass",
		}
		dest := destinations.NewJFrogDestination(config, logger)
		require.Error(t, dest.Validate())
	})

	t.Run("missing password", func(t *testing.T) {
		config := destinations.JFrogConfig{
			URL:  "https://jfrog.example.com",
			User: "testuser",
		}
		dest := destinations.NewJFrogDestination(config, logger)
		require.Error(t, dest.Validate())
	})

	t.Run("invalid URL scheme", func(t *testing.T) {
		config := destinations.JFrogConfig{
			URL:      "ftp://jfrog.example.com",
			User:     "testuser",
			Password: "testpass",
		}
		dest := destinations.NewJFrogDestination(config, logger)
		require.Error(t, dest.Validate())
	})
}

func TestJFrogDestinationPut(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	config := createValidConfig(server)
	logger := createTestLogger()
	dest := destinations.NewJFrogDestination(config, logger)

	t.Run("valid artifact", func(t *testing.T) {
		artifact := &core.Artifact{
			Name:     "test-artifact",
			Version:  "1.0.0",
			Location: "test-repo/test-artifact",
			Metadata: map[string]string{
				"prop1": "value1",
				"prop2": "value2",
			},
		}
		err := dest.Put(context.Background(), artifact, strings.NewReader("test content"))
		require.NoError(t, err)
	})

	t.Run("empty location", func(t *testing.T) {
		artifact := &core.Artifact{
			Name:     "test-artifact",
			Version:  "1.0.0",
			Location: "",
		}
		err := dest.Put(context.Background(), artifact, strings.NewReader("test content"))
		require.Error(t, err)
	})
}

func TestJFrogDestinationExists(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	config := createValidConfig(server)
	logger := createTestLogger()
	dest := destinations.NewJFrogDestination(config, logger)

	t.Run("existing artifact", func(t *testing.T) {
		artifact := &core.Artifact{
			Name:     "test-artifact",
			Version:  "1.0.0",
			Location: "test-repo/test-artifact",
		}
		exists, err := dest.Exists(context.Background(), artifact)
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("non-existing artifact", func(t *testing.T) {
		artifact := &core.Artifact{
			Name:     "non-existing",
			Version:  "1.0.0",
			Location: "test-repo/non-existing",
		}
		exists, err := dest.Exists(context.Background(), artifact)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("empty location", func(t *testing.T) {
		artifact := &core.Artifact{
			Name:     "test-artifact",
			Version:  "1.0.0",
			Location: "",
		}
		exists, err := dest.Exists(context.Background(), artifact)
		require.Error(t, err)
		require.False(t, exists)
	})
}
