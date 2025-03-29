// Package destinations provides implementations of the core.Destination interface.
package destinations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

// JFrogConfig holds the configuration for JFrog Artifactory.
type JFrogConfig struct {
	URL      string
	User     string
	Password string
}

// JFrogDestination implements core.Destination for JFrog Artifactory.
type JFrogDestination struct {
	config JFrogConfig
	client *http.Client
	logger *logrus.Logger
}

// NewJFrogDestination returns a new JFrogDestination instance configured for interacting with JFrog Artifactory.
// It leverages the provided configuration to set endpoint and credentials and uses the supplied logger for logging.
// The function also initializes an HTTP client for issuing requests to the server.
func NewJFrogDestination(config JFrogConfig, logger *logrus.Logger) *JFrogDestination {
	return &JFrogDestination{
		config: config,
		client: &http.Client{},
		logger: logger,
	}
}

// createRequest creates a new HTTP request for uploading an artifact.
func (d *JFrogDestination) createRequest(
	ctx context.Context,
	targetURL *url.URL,
	content io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, targetURL.String(), content)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	req.SetBasicAuth(d.config.User, d.config.Password)

	// Add headers
	req.Header.Set("Content-Type", "application/octet-stream")

	return req, nil
}

// handleResponse handles the HTTP response from the server.
func (d *JFrogDestination) handleResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to upload artifact: HTTP %d - %s", resp.StatusCode, body)
	}

	return nil
}

// buildTargetURL builds the target URL for the artifact.
func (d *JFrogDestination) buildTargetURL(artifact *core.Artifact) (*url.URL, error) {
	targetURL, err := url.Parse(d.config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid JFrog URL: %w", err)
	}

	// Add the artifact path
	artifactPath := path.Join(targetURL.Path, "artifactory", artifact.Location)
	targetURL.Path = artifactPath

	// Add properties as matrix parameters if any
	if len(artifact.Metadata) > 0 {
		params := make([]string, 0, len(artifact.Metadata))
		for k, v := range artifact.Metadata {
			params = append(params, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
		}

		matrixPath := artifactPath + ";" + strings.Join(params, ";")
		targetURL.Path = matrixPath
		targetURL.RawPath = matrixPath
	}

	return targetURL, nil
}

// Put uploads an artifact to JFrog Artifactory.
func (d *JFrogDestination) Put(ctx context.Context, artifact *core.Artifact, content io.Reader) error {
	logger := d.logger.WithFields(logrus.Fields{
		"name":     artifact.Name,
		"version":  artifact.Version,
		"location": artifact.Location,
	})

	// Validate artifact
	if artifact.Location == "" {
		logger.Error("Artifact location cannot be empty")

		return fmt.Errorf("artifact location cannot be empty")
	}

	// Build target URL
	targetURL, err := d.buildTargetURL(artifact)
	if err != nil {
		logger.WithError(err).Error("Failed to build target URL")

		return err
	}

	// Create request
	req, err := d.createRequest(ctx, targetURL, content)
	if err != nil {
		logger.WithError(err).Error("Failed to create request")

		return err
	}

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to upload artifact")

		return fmt.Errorf("failed to upload artifact: %w", err)
	}
	defer resp.Body.Close()

	// Handle response
	if err := d.handleResponse(resp); err != nil {
		logger.WithError(err).Error("Upload failed")

		return err
	}

	logger.Info("Successfully uploaded artifact")

	return nil
}

// Exists checks if an artifact exists in JFrog Artifactory.
func (d *JFrogDestination) Exists(ctx context.Context, artifact *core.Artifact) (bool, error) {
	logger := d.logger.WithFields(logrus.Fields{
		"name":     artifact.Name,
		"version":  artifact.Version,
		"location": artifact.Location,
	})

	// Validate artifact
	if artifact.Location == "" {
		logger.Error("Artifact location cannot be empty")

		return false, fmt.Errorf("artifact location cannot be empty")
	}

	// Construct the target URL
	targetURL, err := url.Parse(d.config.URL)
	if err != nil {
		logger.WithError(err).Error("Invalid JFrog URL")

		return false, fmt.Errorf("invalid JFrog URL: %w", err)
	}

	// Add the artifact path
	targetURL.Path = path.Join(targetURL.Path, "artifactory", artifact.Location)

	// Create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL.String(), nil)
	if err != nil {
		logger.WithError(err).Error("Failed to create request")

		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	req.SetBasicAuth(d.config.User, d.config.Password)

	// Send the request
	resp, err := d.client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to check artifact existence")

		return false, fmt.Errorf("failed to check artifact existence: %w", err)
	}
	defer resp.Body.Close()

	exists := resp.StatusCode == http.StatusOK
	logger.WithField("exists", exists).Info("Checked artifact existence")

	return exists, nil
}

// Validate checks if the JFrog configuration is valid.
func (d *JFrogDestination) Validate() error {
	if d.config.URL == "" {
		return fmt.Errorf("JFrog URL cannot be empty")
	}

	parsedURL, err := url.Parse(d.config.URL)
	if err != nil {
		return fmt.Errorf("invalid JFrog URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid JFrog URL scheme: %s", parsedURL.Scheme)
	}

	if d.config.User == "" {
		return fmt.Errorf("JFrog user cannot be empty")
	}

	if d.config.Password == "" {
		return fmt.Errorf("JFrog password cannot be empty")
	}

	return nil
}
