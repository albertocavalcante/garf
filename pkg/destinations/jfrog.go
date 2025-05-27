// Package destinations provides implementations of the core.Destination interface.
package destinations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/urlprocessor"
	"github.com/sirupsen/logrus"
)

// JFrogConfig contains the configuration for connecting to JFrog Artifactory.
type JFrogConfig struct {
	URL      string // Base URL of the Artifactory instance (e.g., https://artifactory.example.com)
	User     string // Username for authentication
	Password string // Password for authentication
	DestPath string // Path within Artifactory where artifacts will be stored
}

// JFrogDestination implements the core.Destination interface for JFrog Artifactory.
type JFrogDestination struct {
	config      JFrogConfig
	client      *http.Client
	logger      *logrus.Logger
	pathBuilder *urlprocessor.PathBuilder
}

// NewJFrogDestination creates a new JFrogDestination with the provided configuration.
func NewJFrogDestination(config JFrogConfig, logger *logrus.Logger) *JFrogDestination {
	return &JFrogDestination{
		config:      config,
		client:      &http.Client{},
		logger:      logger,
		pathBuilder: urlprocessor.NewWithLogger(logger),
	}
}

// BuildTargetURL constructs the full URL for an artifact in JFrog Artifactory.
// It includes the base URL, destination path, and matrix parameters for metadata.
func (d *JFrogDestination) BuildTargetURL(artifact *core.Artifact, raw bool) (*url.URL, error) {
	logger := d.logger.WithFields(logrus.Fields{
		"artifact_name":     artifact.Name,
		"artifact_location": artifact.Location,
		"raw_mode":          raw,
		"base_url":          d.config.URL,
		"dest_path":         d.config.DestPath,
	})

	logger.Debug("Building target URL for JFrog upload")

	// Parse the base JFrog URL
	targetURL, err := url.Parse(d.config.URL)
	if err != nil {
		logger.WithError(err).Error("Failed to parse base JFrog URL")

		return nil, fmt.Errorf("invalid JFrog URL: %w", err)
	}

	logger.WithField("parsed_base_url", targetURL.String()).Debug("Parsed base JFrog URL")

	// Build the artifact path
	artifactPath, err := d.buildArtifactPath(artifact, targetURL, raw)
	if err != nil {
		logger.WithError(err).Error("Failed to build artifact path")

		return nil, err
	}

	logger.WithField("artifact_path", artifactPath).Debug("Built artifact path")

	targetURL.Path = artifactPath

	// Add matrix parameters for metadata
	if len(artifact.Metadata) > 0 {
		matrixPath := artifactPath + d.buildMatrixParams(artifact.Metadata)
		targetURL.Path = matrixPath
		targetURL.RawPath = matrixPath
		logger.WithFields(logrus.Fields{
			"metadata":    artifact.Metadata,
			"matrix_path": matrixPath,
		}).Debug("Added matrix parameters to URL")
	}

	logger.WithField("final_url", targetURL.String()).Info("Built final target URL for JFrog upload")

	return targetURL, nil
}

// buildMatrixParams converts metadata to JFrog matrix parameters.
func (d *JFrogDestination) buildMatrixParams(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	// Sort keys for consistent ordering
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	// Build parameters
	params := make([]string, 0, len(metadata))
	for _, k := range keys {
		params = append(params, fmt.Sprintf("%s=%s", k, url.QueryEscape(metadata[k])))
	}

	return ";" + strings.Join(params, ";")
}

// buildArtifactPath constructs the storage path for the artifact.
func (d *JFrogDestination) buildArtifactPath(artifact *core.Artifact, targetURL *url.URL, raw bool) (string, error) {
	logger := d.logger.WithFields(logrus.Fields{
		"artifact_location": artifact.Location,
		"artifact_name":     artifact.Name,
		"raw_mode":          raw,
	})

	if artifact.Location == "" {
		artifactPath := path.Join(targetURL.Path, d.config.DestPath, artifact.Name)
		logger.WithField("artifact_path", artifactPath).Debug("Built simple artifact path (no location)")

		return artifactPath, nil
	}

	sourceURL, err := url.Parse(artifact.Location)
	if err != nil {
		logger.WithError(err).Error("Failed to parse artifact location URL")

		return "", fmt.Errorf("invalid source location: %w", err)
	}

	logger.WithField("parsed_source_url", sourceURL.String()).Debug("Parsed source URL")

	// Get the structured path from our path builder
	structuredPath := d.pathBuilder.ProcessURL(sourceURL, raw)
	logger.WithField("structured_path", structuredPath).Debug("Generated structured path from URL processor")

	// Replace the filename in the structured path with the artifact name
	// This is important for cases like ZIP extraction where the artifact name
	// might be different from the filename in the URL
	structuredDir := path.Dir(structuredPath)
	if structuredDir == "." {
		// If there's no directory structure, just use the artifact name
		structuredPath = artifact.Name
	} else {
		// Replace the filename with the artifact name
		structuredPath = path.Join(structuredDir, artifact.Name)
	}

	logger.WithField("final_structured_path", structuredPath).Debug("Updated structured path with artifact name")

	artifactPath := path.Join(targetURL.Path, d.config.DestPath, structuredPath)
	logger.WithField("final_artifact_path", artifactPath).Debug("Built final artifact path")

	return artifactPath, nil
}

// Put uploads an artifact to JFrog Artifactory.
func (d *JFrogDestination) Put(ctx context.Context, artifact *core.Artifact, content io.Reader, raw bool) error {
	logger := d.logger.WithFields(logrus.Fields{
		"name":     artifact.Name,
		"version":  artifact.Version,
		"location": artifact.Location,
		"raw_mode": raw,
	})

	logger.Info("Starting JFrog artifact upload")

	if artifact.Location == "" {
		logger.Error("Artifact location cannot be empty")

		return fmt.Errorf("artifact location cannot be empty")
	}

	// Build the target URL
	targetURL, err := d.BuildTargetURL(artifact, raw)
	if err != nil {
		logger.WithError(err).Error("Failed to build target URL")

		return err
	}

	logger.WithField("target_url", targetURL.String()).Info("Built target URL for upload")

	// Create and send request
	req, err := d.createRequest(ctx, targetURL, content)
	if err != nil {
		logger.WithError(err).Error("Failed to create HTTP request")

		return err
	}

	logger.WithFields(logrus.Fields{
		"method":       req.Method,
		"url":          req.URL.String(),
		"content_type": req.Header.Get("Content-Type"),
	}).Info("Sending HTTP request to JFrog")

	resp, err := d.client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Failed to send HTTP request to JFrog")

		return fmt.Errorf("failed to upload artifact: %w", err)
	}
	defer resp.Body.Close()

	logger.WithFields(logrus.Fields{
		"status_code":    resp.StatusCode,
		"content_type":   resp.Header.Get("Content-Type"),
		"content_length": resp.Header.Get("Content-Length"),
	}).Info("Received response from JFrog")

	// Handle response
	if err := d.handleResponse(resp); err != nil {
		logger.WithError(err).Error("JFrog upload failed")

		return err
	}

	logger.Info("Successfully uploaded artifact to JFrog")

	return nil
}

// Exists checks if an artifact already exists in JFrog Artifactory.
func (d *JFrogDestination) Exists(ctx context.Context, artifact *core.Artifact, raw bool) (bool, error) {
	logger := d.logger.WithFields(logrus.Fields{
		"name":     artifact.Name,
		"version":  artifact.Version,
		"location": artifact.Location,
	})

	if artifact.Location == "" {
		logger.Error("Artifact location cannot be empty")

		return false, fmt.Errorf("artifact location cannot be empty")
	}

	// Build the target URL
	targetURL, err := d.BuildTargetURL(artifact, raw)
	if err != nil {
		logger.WithError(err).Error("Failed to build target URL")

		return false, err
	}

	// Create and send HEAD request
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL.String(), nil)
	if err != nil {
		logger.WithError(err).Error("Failed to create request")

		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(d.config.User, d.config.Password)

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

// createRequest creates an HTTP request with authentication and headers.
func (d *JFrogDestination) createRequest(
	ctx context.Context,
	targetURL *url.URL,
	content io.Reader,
) (*http.Request, error) {
	d.logger.WithFields(logrus.Fields{
		"method": http.MethodPut,
		"url":    targetURL.String(),
	}).Debug("Creating HTTP PUT request for JFrog upload")

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, targetURL.String(), content)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication and headers
	req.SetBasicAuth(d.config.User, d.config.Password)
	req.Header.Set("Content-Type", "application/octet-stream")

	d.logger.WithFields(logrus.Fields{
		"user":         d.config.User,
		"content_type": req.Header.Get("Content-Type"),
	}).Debug("Added authentication and headers to request")

	return req, nil
}

// handleResponse processes HTTP responses from JFrog.
func (d *JFrogDestination) handleResponse(resp *http.Response) error {
	d.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
	}).Debug("Processing JFrog response")

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		d.logger.WithFields(logrus.Fields{
			"status_code":   resp.StatusCode,
			"status":        resp.Status,
			"response_body": string(body),
		}).Error("JFrog returned error status code")

		return fmt.Errorf("failed to upload artifact: HTTP %d - %s", resp.StatusCode, body)
	}

	d.logger.WithField("status_code", resp.StatusCode).Debug("JFrog upload successful")

	return nil
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

	if d.config.DestPath == "" {
		return fmt.Errorf("JFrog destination path cannot be empty")
	}

	return nil
}

// GetConfig returns the JFrog configuration (used for testing).
func (d *JFrogDestination) GetConfig() JFrogConfig {
	return d.config
}
