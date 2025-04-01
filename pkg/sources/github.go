// Package sources provides implementations of the core.Source interface.
package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

const (
	// GitHubHost is the standard GitHub hostname.
	GitHubHost = "github.com"
)

// GitHubSource implements core.Source for GitHub releases.
type GitHubSource struct {
	client  *http.Client
	logger  *logrus.Logger
	tempDir string
}

// NewGitHubSource creates a new GitHubSource instance.
func NewGitHubSource(logger *logrus.Logger) *GitHubSource {
	if logger == nil {
		logger = logrus.New()
	}

	return &GitHubSource{
		client: &http.Client{},
		logger: logger,
	}
}

// List is not implemented for GitHub source as it requires specific release URLs.
func (s *GitHubSource) List(ctx context.Context) ([]*core.Artifact, error) {
	return nil, fmt.Errorf("listing artifacts from GitHub is not supported, use specific release URLs")
}

// validateGitHubURL validates that the given URL is a GitHub URL.
func validateGitHubURL(location string) error {
	parsedURL, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Allow test server URLs during testing
	if strings.Contains(parsedURL.Host, "127.0.0.1") || strings.Contains(parsedURL.Host, "localhost") {
		return nil
	}

	if !strings.Contains(parsedURL.Host, GitHubHost) {
		return fmt.Errorf("not a GitHub URL: %s", location)
	}

	return nil
}

// ensureTempDir ensures that a temporary directory exists for downloads.
func (s *GitHubSource) ensureTempDir() error {
	if s.tempDir != "" {
		return nil
	}

	tempDir, err := os.MkdirTemp("", "garf-github-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	s.tempDir = tempDir

	return nil
}

// createGitHubRequest creates an HTTP request for downloading from GitHub.
func createGitHubRequest(ctx context.Context, location string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	return req, nil
}

// downloadToTempFile downloads the content from the response to a temporary file.
func (s *GitHubSource) downloadToTempFile(resp *http.Response) (io.ReadCloser, error) {
	tempFile, err := os.CreateTemp(s.tempDir, "download-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())

		return nil, fmt.Errorf("failed to save artifact: %w", err)
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())

		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	return &cleanupReadCloser{
		ReadCloser: tempFile,
		cleanup: func() {
			tempFile.Close()
			os.Remove(tempFile.Name())
		},
	}, nil
}

// Get retrieves an artifact from GitHub.
func (s *GitHubSource) Get(ctx context.Context, artifact *core.Artifact) (io.ReadCloser, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"source": "github",
		"url":    artifact.Location,
	})

	if err := validateGitHubURL(artifact.Location); err != nil {
		logger.WithError(err).Error("Invalid URL")

		return nil, err
	}

	if err := s.ensureTempDir(); err != nil {
		logger.WithError(err).Error("Failed to create temporary directory")

		return nil, err
	}

	req, err := createGitHubRequest(ctx, artifact.Location)
	if err != nil {
		logger.WithError(err).Error("Failed to create request")

		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to download artifact")

		return nil, fmt.Errorf("failed to download artifact: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()

		return nil, fmt.Errorf("failed to download artifact: HTTP %d", resp.StatusCode)
	}

	reader, err := s.downloadToTempFile(resp)
	resp.Body.Close()

	if err != nil {
		logger.WithError(err).Error("Failed to handle download")

		return nil, err
	}

	return reader, nil
}

// Validate checks if the source is properly configured.
func (s *GitHubSource) Validate() error {
	return nil
}

// Close cleans up any temporary resources.
func (s *GitHubSource) Close() error {
	if s.tempDir != "" {
		if err := os.RemoveAll(s.tempDir); err != nil {
			return fmt.Errorf("failed to clean up temporary directory: %w", err)
		}

		s.tempDir = ""
	}

	return nil
}

// cleanupReadCloser wraps an io.ReadCloser and performs cleanup when closed.
type cleanupReadCloser struct {
	io.ReadCloser
	cleanup func()
}

func (c *cleanupReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cleanup()

	return err
}

// SetClient sets the HTTP client for the source.
func (s *GitHubSource) SetClient(client *http.Client) {
	s.client = client
}
