// Package destinations provides implementations for various registry destinations.
package destinations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

const (
	// cloudsmithTimeout is the default timeout for HTTP requests to Cloudsmith.
	cloudsmithTimeout = 30 * time.Second
)

// CloudsmithConfig holds the configuration for Cloudsmith destinations.
type CloudsmithConfig struct {
	// URL is the base URL of the Cloudsmith repository (e.g., https://api.cloudsmith.io)
	URL string

	// User is the username or API key for authentication
	User string

	// Password is the password or API secret for authentication
	Password string

	// DestPath is the destination path in the repository (format: owner/repo)
	DestPath string

	// SourcePathStrip is the path prefix to strip from source URLs
	SourcePathStrip string
}

// CloudsmithDestination implements the core.Destination interface for Cloudsmith repositories.
type CloudsmithDestination struct {
	config     CloudsmithConfig
	logger     *logrus.Logger
	httpClient *http.Client
}

// NewCloudsmithDestination creates a new Cloudsmith destination.
func NewCloudsmithDestination(config CloudsmithConfig, logger *logrus.Logger) core.Destination {
	return &CloudsmithDestination{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: cloudsmithTimeout,
		},
	}
}

// uploadFileResponse represents the response from Cloudsmith file upload API.
type uploadFileResponse struct {
	Identifier string `json:"identifier"`
}

// createPackageRequest represents the request to create a Raw package.
type createPackageRequest struct {
	PackageFile string `json:"package_file"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// Put stores an artifact in the Cloudsmith repository using their two-step upload process.
func (d *CloudsmithDestination) Put(ctx context.Context, artifact *core.Artifact, content io.Reader, raw bool) (string, error) {
	d.logger.WithFields(logrus.Fields{
		"artifact_name": artifact.Name,
		"destination":   d.config.DestPath,
		"url":           d.config.URL,
		"raw":           raw,
	}).Info("Cloudsmith destination: Starting Put operation")

	// Step 1: Upload the file and get an identifier
	identifier, err := d.uploadFile(ctx, artifact, content)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Step 2: Create the package using the identifier
	packageURL, err := d.createPackage(ctx, artifact, identifier, raw)
	if err != nil {
		return "", fmt.Errorf("failed to create package: %w", err)
	}

	d.logger.WithFields(logrus.Fields{
		"artifact_name": artifact.Name,
		"package_url":   packageURL,
	}).Info("Cloudsmith destination: Successfully uploaded artifact")

	return packageURL, nil
}

// uploadFile performs Step 1 of Cloudsmith upload: upload the file and get an identifier.
func (d *CloudsmithDestination) uploadFile(ctx context.Context, artifact *core.Artifact, content io.Reader) (string, error) {
	// Parse destination path to get owner and repo
	parts := strings.Split(d.config.DestPath, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid destination path format, expected 'owner/repo', got: %s", d.config.DestPath)
	}

	owner, repo := parts[0], parts[1]

	// Build upload URL: https://upload.cloudsmith.io/{owner}/{repo}/{package_name}
	uploadURL := fmt.Sprintf("https://upload.cloudsmith.io/%s/%s/%s", owner, repo, artifact.Name)

	// Read content into buffer for upload
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	// Create PUT request
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(contentBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	// Set authentication headers
	req.SetBasicAuth(d.config.User, d.config.Password)
	req.Header.Set("Content-Type", "application/octet-stream")

	d.logger.WithFields(logrus.Fields{
		"upload_url":    uploadURL,
		"content_size":  len(contentBytes),
		"artifact_name": artifact.Name,
	}).Debug("Cloudsmith: Uploading file")

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response to get identifier
	var uploadResp uploadFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("failed to parse upload response: %w", err)
	}

	if uploadResp.Identifier == "" {
		return "", fmt.Errorf("no identifier returned from upload")
	}

	d.logger.WithField("identifier", uploadResp.Identifier).Debug("Cloudsmith: File uploaded successfully")

	return uploadResp.Identifier, nil
}

// createPackage performs Step 2 of Cloudsmith upload: create the package using the identifier.
func (d *CloudsmithDestination) createPackage(ctx context.Context, artifact *core.Artifact, identifier string, raw bool) (string, error) {
	// Parse destination path to get owner and repo
	parts := strings.Split(d.config.DestPath, "/")
	owner, repo := parts[0], parts[1]

	// Build API URL for creating Raw package
	apiURL := fmt.Sprintf("%s/v1/packages/%s/%s/upload/raw/", d.config.URL, owner, repo)

	// Extract version from artifact metadata or use default
	version := "1.0.0"

	if artifact.Metadata != nil {
		if v, exists := artifact.Metadata["version"]; exists {
			version = v
		}
	}

	// Create package request
	packageReq := createPackageRequest{
		PackageFile: identifier,
		Name:        artifact.Name,
		Summary:     fmt.Sprintf("Artifact %s mirrored from %s", artifact.Name, artifact.Location),
		Description: fmt.Sprintf("Artifact %s uploaded via garf from source: %s", artifact.Name, artifact.Location),
		Version:     version,
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(packageReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal package request: %w", err)
	}

	// Create POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create package request: %w", err)
	}

	// Set authentication and content headers
	req.SetBasicAuth(d.config.User, d.config.Password)
	req.Header.Set("Content-Type", "application/json")

	d.logger.WithFields(logrus.Fields{
		"api_url":     apiURL,
		"package_req": packageReq,
	}).Debug("Cloudsmith: Creating package")

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create package: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("package creation failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// For Raw packages, build the expected destination path
	destPath, err := d.BuildDestinationPath(artifact, raw)
	if err != nil {
		return "", fmt.Errorf("failed to build destination path: %w", err)
	}

	d.logger.WithField("destination_path", destPath).Debug("Cloudsmith: Package created successfully")

	return destPath, nil
}

// Exists checks if an artifact already exists in the Cloudsmith repository.
func (d *CloudsmithDestination) Exists(ctx context.Context, artifact *core.Artifact, raw bool) (bool, error) {
	d.logger.WithFields(logrus.Fields{
		"artifact_name": artifact.Name,
		"destination":   d.config.DestPath,
		"url":           d.config.URL,
		"raw":           raw,
	}).Debug("Cloudsmith destination: Checking if artifact exists")

	// Parse destination path to get owner and repo
	parts := strings.Split(d.config.DestPath, "/")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid destination path format, expected 'owner/repo', got: %s", d.config.DestPath)
	}

	owner, repo := parts[0], parts[1]

	// Build API URL for listing packages
	apiURL := fmt.Sprintf("%s/v1/packages/%s/%s/", d.config.URL, owner, repo)

	// Create GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create existence check request: %w", err)
	}

	// Set authentication headers
	req.SetBasicAuth(d.config.User, d.config.Password)

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check package existence: %w", err)
	}
	defer resp.Body.Close()

	// If we can't access the repository, assume package doesn't exist
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)

		return false, fmt.Errorf("existence check failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// For now, we'll assume the package doesn't exist if we reach here
	// A full implementation would parse the package list and search for the specific artifact
	d.logger.Debug("Cloudsmith: Existence check completed, assuming package doesn't exist")

	return false, nil
}

// BuildDestinationPath builds the destination path without uploading.
func (d *CloudsmithDestination) BuildDestinationPath(artifact *core.Artifact, raw bool) (string, error) {
	if artifact == nil {
		return "", fmt.Errorf("artifact cannot be nil")
	}

	// Parse destination path
	parts := strings.Split(d.config.DestPath, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid destination path format, expected 'owner/repo', got: %s", d.config.DestPath)
	}

	owner, repo := parts[0], parts[1]

	// Build the destination path in Cloudsmith format that includes the owner/repo
	if raw {
		// In raw mode, preserve original structure but use Cloudsmith path format
		return fmt.Sprintf("/%s/%s/raw/names/%s/versions/latest/%s",
			owner, repo, artifact.Name, artifact.Name), nil
	}

	// In normal mode, use structured path
	return fmt.Sprintf("/%s/%s/raw/names/%s/versions/latest/%s",
		owner, repo, artifact.Name, artifact.Name), nil
}

// Validate checks if the destination configuration is valid.
func (d *CloudsmithDestination) Validate() error {
	if d.config.URL == "" {
		return fmt.Errorf("cloudsmith URL cannot be empty")
	}

	// Validate URL format
	_, err := url.Parse(d.config.URL)
	if err != nil {
		return fmt.Errorf("invalid cloudsmith URL: %w", err)
	}

	if d.config.User == "" {
		return fmt.Errorf("cloudsmith user cannot be empty")
	}

	if d.config.Password == "" {
		return fmt.Errorf("cloudsmith password cannot be empty")
	}

	if d.config.DestPath == "" {
		return fmt.Errorf("cloudsmith destination path cannot be empty")
	}

	// Validate destination path format (should be owner/repo)
	parts := strings.Split(d.config.DestPath, "/")
	if len(parts) != 2 {
		return fmt.Errorf("cloudsmith destination path must be in format 'owner/repo', got: %s", d.config.DestPath)
	}

	// Validate SourcePathStrip if specified using centralized validation
	if err := core.ValidateSourcePathStrip(d.config.SourcePathStrip); err != nil {
		return err
	}

	return nil
}

// String implements fmt.Stringer to provide a safe string representation.
func (d *CloudsmithDestination) String() string {
	return fmt.Sprintf("CloudsmithDestination{URL: %s, User: %s, DestPath: %s}",
		d.config.URL, d.config.User, d.config.DestPath)
}
