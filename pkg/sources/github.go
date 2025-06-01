// Package sources provides implementations of the core.Source interface.
package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

// GitHubSource implements core.Source for GitHub releases.
type GitHubSource struct {
	*HTTPSource
}

// NewGitHubSource creates a new GitHubSource instance.
func NewGitHubSource(logger *logrus.Logger) *GitHubSource {
	return &GitHubSource{
		HTTPSource: NewHTTPSource(logger, "github"),
	}
}

// List is not implemented for GitHub source as it requires specific release URLs.
func (s *GitHubSource) List(ctx context.Context) ([]*core.Artifact, error) {
	return nil, fmt.Errorf("listing artifacts from GitHub is not supported, use specific release URLs")
}

// createGitHubRequest creates an HTTP request for downloading from GitHub with authentication.
func createGitHubRequest(ctx context.Context, location string) (*http.Request, error) {
	req, err := CreateHTTPRequest(ctx, location)
	if err != nil {
		return nil, err
	}

	// Add GitHub-specific authentication if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	return req, nil
}

// Get retrieves an artifact from GitHub.
func (s *GitHubSource) Get(ctx context.Context, artifact *core.Artifact) (io.ReadCloser, error) {
	if artifact == nil {
		return nil, fmt.Errorf("artifact cannot be nil")
	}

	logger := s.GetLogger().WithFields(logrus.Fields{
		"source":        "github",
		"url":           artifact.Location,
		"artifact_name": artifact.Name,
	})

	logger.Info("Starting GitHub artifact download")

	if err := core.ValidateGitHubURL(artifact.Location); err != nil {
		logger.WithError(err).Error("Invalid GitHub URL")

		return nil, err
	}

	logger.Debug("GitHub URL validation passed")

	if err := s.EnsureTempDir(); err != nil {
		logger.WithError(err).Error("Failed to create temporary directory")

		return nil, err
	}

	logger.Debug("Creating HTTP request for GitHub download")

	req, err := createGitHubRequest(ctx, artifact.Location)
	if err != nil {
		logger.WithError(err).Error("Failed to create HTTP request")

		return nil, err
	}

	resp, err := s.DoRequest(req)
	if err != nil {
		logger.WithError(err).Error("Failed to download artifact from GitHub")

		return nil, err
	}
	defer resp.Body.Close()

	reader, err := s.DownloadToTempFile(resp)
	if err != nil {
		logger.WithError(err).Error("Failed to save artifact to temporary file")

		return nil, err
	}

	logger.Info("Successfully completed GitHub artifact download")

	return reader, nil
}

// Validate checks if the source is properly configured.
func (s *GitHubSource) Validate() error {
	return nil
}
