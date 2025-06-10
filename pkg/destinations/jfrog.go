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
	"sync"
	"time"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/urlprocessor"
	"github.com/sirupsen/logrus"
)

const (
	// httpClientTimeout is the default timeout for HTTP requests to JFrog Artifactory.
	httpClientTimeout = 30 * time.Second
)

// JFrogConfig contains the configuration for connecting to JFrog Artifactory.
type JFrogConfig struct {
	URL      string // Base URL of the Artifactory instance (e.g., https://artifactory.example.com)
	User     string // Username for authentication
	Password string // Password for authentication
	DestPath string // Path within Artifactory where artifacts will be stored

	// SourcePathStrip is an optional prefix to strip from source URLs before processing.
	// This is useful for JFrog-to-JFrog mirroring where you want to remove the source
	// repository path. For example, setting this to "artifactory.corp.net/staging/"
	// will strip that prefix from source URLs before generating the destination path.
	SourcePathStrip string
}

// JFrogDestination implements the core.Destination interface for JFrog Artifactory.
type JFrogDestination struct {
	config       JFrogConfig
	logger       *logrus.Logger
	urlProcessor *urlprocessor.PathBuilder
	httpClient   *http.Client
	baseURL      *url.URL

	// Thread-safety fields for lazy initialization
	clientOnce sync.Once
	urlOnce    sync.Once
	urlErr     error
}

// Ensure JFrogDestination implements the Destination interface.
var _ core.Destination = (*JFrogDestination)(nil)

// NewJFrogDestination creates a new JFrogDestination with the provided configuration.
func NewJFrogDestination(config JFrogConfig, logger *logrus.Logger) *JFrogDestination {
	return &JFrogDestination{
		config:       config,
		logger:       logger,
		urlProcessor: urlprocessor.NewWithLogger(logger),
	}
}

// BuildTargetURL constructs the full URL for an artifact in JFrog Artifactory.
// It includes the base URL, destination path, and matrix parameters for metadata.
func (d *JFrogDestination) BuildTargetURL(artifact *core.Artifact, raw bool) (*url.URL, error) {
	if err := d.validateForURLBuilding(); err != nil {
		return nil, err
	}

	structuredPath, err := d.buildStructuredPath(artifact, raw)
	if err != nil {
		return nil, err
	}

	return d.buildFullURL(structuredPath, artifact.Metadata)
}

// BuildDestinationPath calculates the destination path for an artifact without performing an upload.
// This is used for dry-run scenarios to predict the final location.
// The returned path is the path component of the full target URL.
func (d *JFrogDestination) BuildDestinationPath(artifact *core.Artifact, raw bool) (string, error) {
	structuredPath, err := d.buildStructuredPath(artifact, raw)
	if err != nil {
		return "", err
	}

	// The destination path is DestPath joined with the processed structured path.
	finalDestPath := path.Join(d.config.DestPath, structuredPath)

	// Ensure leading slash
	if !strings.HasPrefix(finalDestPath, "/") {
		finalDestPath = "/" + finalDestPath
	}

	d.logger.WithField("final_dest_path", finalDestPath).Info("Calculated final destination path")

	return finalDestPath, nil
}

// validateForURLBuilding performs common validation needed for URL building operations.
func (d *JFrogDestination) validateForURLBuilding() error {
	_, err := d.getBaseURL()

	return err
}

// buildStructuredPath creates the structured path component for an artifact.
func (d *JFrogDestination) buildStructuredPath(artifact *core.Artifact, raw bool) (string, error) {
	dLogger := d.logger.WithFields(logrus.Fields{
		"artifact_name":     artifact.Name,
		"artifact_location": artifact.Location,
		"raw_mode":          raw,
		"dest_path":         d.config.DestPath,
		"source_path_strip": d.config.SourcePathStrip,
	})
	dLogger.Debug("Building structured path for artifact")

	if artifact.Location == "" {
		return "", fmt.Errorf("artifact location cannot be empty")
	}

	sourceURL, err := url.Parse(artifact.Location)
	if err != nil {
		return "", fmt.Errorf("failed to parse artifact location URL %s: %w", artifact.Location, err)
	}

	return d.urlProcessor.BuildStructuredPath(
		sourceURL,
		artifact.Name,
		d.config.SourcePathStrip,
		raw,
	)
}

// buildFullURL constructs the complete URL including base URL, path, and matrix parameters.
func (d *JFrogDestination) buildFullURL(structuredPath string, metadata map[string]string) (*url.URL, error) {
	baseURL, err := d.getBaseURL()
	if err != nil {
		return nil, err
	}

	// Combine the base URL path (e.g., "/artifactory") with destination path and structured path
	artifactPath := path.Join(d.config.DestPath, structuredPath)

	// Join the base URL path with the artifact path
	// This preserves any existing path in the base URL (like /artifactory)
	combinedPath := path.Join(baseURL.Path, artifactPath)

	// Ensure the path starts with a slash as it's a URL path component
	if !strings.HasPrefix(combinedPath, "/") {
		combinedPath = "/" + combinedPath
	}

	// Make a copy of baseURL to avoid modifying the original shared instance
	targetURL := *baseURL // Shallow copy is fine as we only modify Path/RawPath
	targetURL.Path = combinedPath

	if len(metadata) > 0 {
		matrixParams := d.buildMatrixParams(metadata)
		targetURL.Path = combinedPath + matrixParams
		d.logger.WithFields(logrus.Fields{
			"metadata":    metadata,
			"matrix_path": targetURL.Path,
		}).Debug("Added matrix parameters to URL")
	}

	d.logger.WithFields(logrus.Fields{
		"base_url_path": baseURL.Path,
		"artifact_path": artifactPath,
		"combined_path": combinedPath,
		"final_url":     targetURL.String(),
	}).Info("Built final target URL for JFrog upload")

	return &targetURL, nil
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

// Put uploads an artifact to JFrog Artifactory.
func (d *JFrogDestination) Put(ctx context.Context, artifact *core.Artifact, content io.Reader, raw bool) (string, error) {
	dLogger := d.logger.WithFields(logrus.Fields{
		"name":     artifact.Name,
		"version":  artifact.Version,
		"location": artifact.Location,
		"raw_mode": raw,
	})
	dLogger.Info("Starting JFrog artifact upload")

	targetURL, err := d.BuildTargetURL(artifact, raw)
	if err != nil {
		return "", err // Error already logged by BuildTargetURL
	}

	req, err := d.createRequest(ctx, targetURL, content)
	if err != nil {
		return "", err // Error already logged by createRequest
	}

	dLogger.Info("Sending HTTP request to JFrog")

	resp, err := d.getHTTPClient().Do(req)
	if err != nil {
		dLogger.WithError(err).Error("Failed to send HTTP request to JFrog")

		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}

	defer resp.Body.Close()

	if err := d.handleResponse(resp); err != nil {
		return "", err // Error already logged by handleResponse
	}

	dLogger.WithField("destination_url", targetURL.String()).Info("Successfully uploaded artifact to JFrog")

	return targetURL.String(), nil
}

// Exists checks if an artifact already exists in JFrog Artifactory.
func (d *JFrogDestination) Exists(ctx context.Context, artifact *core.Artifact, raw bool) (bool, error) {
	dLogger := d.logger.WithFields(logrus.Fields{
		"name":     artifact.Name,
		"version":  artifact.Version,
		"location": artifact.Location,
		"raw_mode": raw,
	})
	dLogger.Debug("Checking artifact existence")

	if artifact.Location == "" {
		return false, fmt.Errorf("artifact location cannot be empty")
	}

	targetURL, err := d.BuildTargetURL(artifact, raw)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL.String(), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	req.SetBasicAuth(d.config.User, d.config.Password)

	resp, err := d.getHTTPClient().Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute HEAD request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		dLogger.Info("Artifact exists")

		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		dLogger.Info("Artifact does not exist")

		return false, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		dLogger.WithError(err).Error("Failed to read response body")

		return false, fmt.Errorf("unexpected status code %d from Artifactory while checking existence: could not read response body: %w", resp.StatusCode, err)
	}

	dLogger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"response":    string(bodyBytes),
	}).Error("Received unexpected status code while checking artifact existence")

	return false, fmt.Errorf("unexpected status code %d from Artifactory while checking existence: %s", resp.StatusCode, string(bodyBytes))
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
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			d.logger.WithError(err).Error("Failed to read response body")

			return fmt.Errorf("failed to upload artifact: HTTP %d - could not read response body: %w", resp.StatusCode, err)
		}

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

// getHTTPClient returns the HTTP client, creating it lazily if needed.
func (d *JFrogDestination) getHTTPClient() *http.Client {
	d.clientOnce.Do(func() {
		d.httpClient = &http.Client{
			Timeout: httpClientTimeout,
		}
	})

	return d.httpClient
}

// getBaseURL returns the parsed base URL, parsing it lazily if needed.
func (d *JFrogDestination) getBaseURL() (*url.URL, error) {
	d.urlOnce.Do(func() {
		// Normalize the URL to ensure /artifactory path is present
		normalizedURL, err := d.normalizeJFrogURL(d.config.URL)
		if err != nil {
			d.urlErr = fmt.Errorf("failed to normalize JFrog URL %q: %w", d.config.URL, err)

			return
		}

		parsedURL, err := url.Parse(normalizedURL)
		if err != nil {
			d.urlErr = fmt.Errorf("invalid JFrog URL %q: %w", normalizedURL, err)

			return
		}

		d.baseURL = parsedURL
	})

	return d.baseURL, d.urlErr
}

// normalizeJFrogURL ensures the JFrog URL has the /artifactory path component.
// It handles both cases: URLs with and without /artifactory.
func (d *JFrogDestination) normalizeJFrogURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL to work with its components
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Validate scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid URL scheme %q, must be http or https", u.Scheme)
	}

	// Validate host
	if u.Host == "" {
		return "", fmt.Errorf("URL must have a host")
	}

	// Handle the path component
	path := u.Path

	// If path is empty or just "/", add /artifactory
	if path == "" || path == "/" {
		u.Path = "/artifactory"
		d.logger.WithFields(logrus.Fields{
			"original_url":   rawURL,
			"normalized_url": u.String(),
		}).Debug("Added /artifactory to JFrog URL")
	} else if !strings.Contains(path, "/artifactory") {
		// If path doesn't contain /artifactory, append it
		u.Path = strings.TrimSuffix(path, "/") + "/artifactory"
		d.logger.WithFields(logrus.Fields{
			"original_url":   rawURL,
			"normalized_url": u.String(),
		}).Debug("Appended /artifactory to existing JFrog URL path")
	} else {
		// Path already contains /artifactory, keep as-is
		d.logger.WithFields(logrus.Fields{
			"url": rawURL,
		}).Debug("JFrog URL already contains /artifactory, no normalization needed")
	}

	return u.String(), nil
}
