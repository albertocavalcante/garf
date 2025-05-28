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

// GenericSource implements core.Source for generic HTTP URLs.
// It doesn't perform any special validation or processing - it just downloads from any HTTP/HTTPS URL.
type GenericSource struct {
	client  *http.Client
	logger  *logrus.Logger
	tempDir string
}

// NewGenericSource creates a new GenericSource instance.
func NewGenericSource(logger *logrus.Logger) *GenericSource {
	if logger == nil {
		logger = logrus.New()
	}

	return &GenericSource{
		client: &http.Client{},
		logger: logger,
	}
}

// List is not implemented for generic source as it requires specific URLs.
func (s *GenericSource) List(ctx context.Context) ([]*core.Artifact, error) {
	return nil, fmt.Errorf("listing artifacts from generic source is not supported, use specific URLs")
}

// ensureTempDir ensures that a temporary directory exists for downloads.
func (s *GenericSource) ensureTempDir() error {
	if s.tempDir != "" {
		s.logger.WithField("temp_dir", s.tempDir).Debug("Using existing temporary directory")
		return nil
	}

	tempDir, err := os.MkdirTemp("", "garf-generic-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	s.tempDir = tempDir
	s.logger.WithField("temp_dir", s.tempDir).Info("Created temporary directory for downloads")

	return nil
}

// createGenericRequest creates an HTTP request for downloading from any URL.
func createGenericRequest(ctx context.Context, location string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")

	return req, nil
}

// downloadToTempFile downloads the content from the response to a temporary file.
func (s *GenericSource) downloadToTempFile(resp *http.Response) (io.ReadCloser, error) {
	tempFile, err := os.CreateTemp(s.tempDir, "download-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"temp_file":      tempFile.Name(),
		"content_length": resp.ContentLength,
	}).Info("Downloading artifact to temporary file")

	bytesWritten, err := io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to save artifact: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"temp_file":     tempFile.Name(),
		"bytes_written": bytesWritten,
	}).Info("Successfully downloaded artifact to temporary file")

	if _, err := tempFile.Seek(0, 0); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	return &cleanupReadCloser{
		ReadCloser: tempFile,
		cleanup: func() {
			s.logger.WithField("temp_file", tempFile.Name()).Debug("Cleaning up temporary file")
			tempFile.Close()
			os.Remove(tempFile.Name())
		},
	}, nil
}

// Get retrieves an artifact from any HTTP/HTTPS URL.
func (s *GenericSource) Get(ctx context.Context, artifact *core.Artifact) (io.ReadCloser, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"source":        "generic",
		"url":           artifact.Location,
		"artifact_name": artifact.Name,
	})

	logger.Info("Starting generic artifact download")

	if err := s.ensureTempDir(); err != nil {
		logger.WithError(err).Error("Failed to create temporary directory")
		return nil, err
	}

	logger.Debug("Creating HTTP request for generic download")

	req, err := createGenericRequest(ctx, artifact.Location)
	if err != nil {
		logger.WithError(err).Error("Failed to create HTTP request")
		return nil, err
	}

	logger.WithFields(logrus.Fields{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": req.Header,
	}).Debug("Sending HTTP request")

	resp, err := s.client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to download artifact")
		return nil, fmt.Errorf("failed to download artifact: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"status_code":    resp.StatusCode,
		"content_length": resp.ContentLength,
		"content_type":   resp.Header.Get("Content-Type"),
	}).Info("Received response")

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		logger.WithField("status_code", resp.StatusCode).Error("Server returned non-200 status code")
		return nil, fmt.Errorf("failed to download artifact: HTTP %d", resp.StatusCode)
	}

	reader, err := s.downloadToTempFile(resp)
	resp.Body.Close()

	if err != nil {
		logger.WithError(err).Error("Failed to save artifact to temporary file")
		return nil, err
	}

	logger.Info("Successfully completed generic artifact download")

	return reader, nil
}

// Validate checks if the source is properly configured.
func (s *GenericSource) Validate() error {
	return nil
}

// Close cleans up any temporary resources.
func (s *GenericSource) Close() error {
	if s.tempDir != "" {
		if err := os.RemoveAll(s.tempDir); err != nil {
			return fmt.Errorf("failed to clean up temporary directory: %w", err)
		}
	}
	return nil
}

// SetClient sets a custom HTTP client (useful for testing).
func (s *GenericSource) SetClient(client *http.Client) {
	s.client = client
}
