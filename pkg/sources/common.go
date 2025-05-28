// Package sources provides implementations of the core.Source interface.
package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

// HTTPSource provides common functionality for HTTP-based sources.
type HTTPSource struct {
	client  *http.Client
	logger  *logrus.Logger
	tempDir string
	prefix  string // Used for temp directory naming (e.g., "github", "generic")
}

// NewHTTPSource creates a new HTTPSource instance.
func NewHTTPSource(logger *logrus.Logger, prefix string) *HTTPSource {
	if logger == nil {
		logger = logrus.New()
	}

	return &HTTPSource{
		client: &http.Client{},
		logger: logger,
		prefix: prefix,
	}
}

// EnsureTempDir ensures that a temporary directory exists for downloads.
func (s *HTTPSource) EnsureTempDir() error {
	if s.tempDir != "" {
		s.logger.WithField("temp_dir", s.tempDir).Debug("Using existing temporary directory")
		return nil
	}

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("garf-%s-*", s.prefix))
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	s.tempDir = tempDir
	s.logger.WithField("temp_dir", s.tempDir).Info("Created temporary directory for downloads")

	return nil
}

// DownloadToTempFile downloads the content from the response to a temporary file.
func (s *HTTPSource) DownloadToTempFile(resp *http.Response) (io.ReadCloser, error) {
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

// DoRequest performs an HTTP request and returns the response.
func (s *HTTPSource) DoRequest(req *http.Request) (*http.Response, error) {
	s.logger.WithFields(logrus.Fields{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": req.Header,
	}).Debug("Sending HTTP request")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifact: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"status_code":    resp.StatusCode,
		"content_length": resp.ContentLength,
		"content_type":   resp.Header.Get("Content-Type"),
	}).Info("Received response")

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download artifact: HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

// Close cleans up any temporary resources.
func (s *HTTPSource) Close() error {
	if s.tempDir != "" {
		if err := os.RemoveAll(s.tempDir); err != nil {
			return fmt.Errorf("failed to clean up temporary directory: %w", err)
		}
		s.tempDir = ""
	}
	return nil
}

// SetClient sets a custom HTTP client (useful for testing).
func (s *HTTPSource) SetClient(client *http.Client) {
	s.client = client
}

// GetLogger returns the logger instance.
func (s *HTTPSource) GetLogger() *logrus.Logger {
	return s.logger
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

// CreateHTTPRequest creates a basic HTTP GET request with common headers.
func CreateHTTPRequest(ctx context.Context, location string) (*http.Request, error) {
	if location == "" {
		return nil, fmt.Errorf("location cannot be empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")
	return req, nil
}
