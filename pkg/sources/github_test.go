package sources_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/sources"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func setupTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("test content"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}))
}

func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	return logger
}

func TestGitHubSourceValidate(t *testing.T) {
	logger := createTestLogger()
	source := sources.NewGitHubSource(logger)
	require.NoError(t, source.Validate())
}

func createTestSource() *sources.GitHubSource {
	source := sources.NewGitHubSource(nil)
	source.SetClient(&http.Client{})

	return source
}

func TestGitHubSource(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	tests := []struct {
		name          string
		url           string
		validate      func(*testing.T, *sources.GitHubSource)
		validateError bool
	}{
		{
			name: "valid github release url",
			url:  server.URL + "/example/repo/releases/download/v1.0.0/artifact.zip",
			validate: func(t *testing.T, s *sources.GitHubSource) {
				require.NotNil(t, s)
			},
			validateError: false,
		},
		{
			name: "invalid url",
			url:  "not-a-url",
			validate: func(t *testing.T, s *sources.GitHubSource) {
				require.NotNil(t, s)
			},
			validateError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := createTestSource()

			// Create a test artifact to validate the URL
			artifact := &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: tt.url,
			}

			// Try to get the artifact to validate the URL
			_, err := source.Get(context.Background(), artifact)
			if tt.validateError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			tt.validate(t, source)
		})
	}
}

func TestGitHubSourceGet(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	tests := []struct {
		name          string
		artifact      *core.Artifact
		validate      func(*testing.T, *sources.GitHubSource)
		validateError bool
	}{
		{
			name: "valid artifact",
			artifact: &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: server.URL + "/example/repo/releases/download/v1.0.0/artifact.zip",
			},
			validate: func(t *testing.T, s *sources.GitHubSource) {
				require.NotNil(t, s)
			},
			validateError: false,
		},
		{
			name: "invalid artifact",
			artifact: &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: "not-a-url",
			},
			validate: func(t *testing.T, s *sources.GitHubSource) {
				require.NotNil(t, s)
			},
			validateError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := createTestSource()

			content, err := source.Get(context.Background(), tt.artifact)
			if tt.validateError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, content)
			tt.validate(t, source)
		})
	}
}

func TestGitHubSourceList(t *testing.T) {
	tests := []struct {
		name          string
		validate      func(*testing.T, *sources.GitHubSource)
		validateError bool
	}{
		{
			name: "list artifacts",
			validate: func(t *testing.T, s *sources.GitHubSource) {
				require.NotNil(t, s)
			},
			validateError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := sources.NewGitHubSource(nil)

			artifacts, err := source.List(context.Background())
			if tt.validateError {
				require.Error(t, err)
				require.Nil(t, artifacts)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, artifacts)
			tt.validate(t, source)
		})
	}
}
