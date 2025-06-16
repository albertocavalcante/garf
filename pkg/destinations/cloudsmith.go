// Package destinations provides implementations for various registry destinations.
package destinations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/albertocavalcante/garf/pkg/core"
	cloudsmith "github.com/cloudsmith-io/cloudsmith-api-go"
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
	config CloudsmithConfig
	logger *logrus.Logger
	client *cloudsmith.APIClient
}

// NewCloudsmithDestination creates a new Cloudsmith destination.
func NewCloudsmithDestination(config CloudsmithConfig, logger *logrus.Logger) core.Destination {
	// Create Cloudsmith configuration
	configuration := cloudsmith.NewConfiguration()
	if config.URL != "" {
		configuration.Servers = []cloudsmith.ServerConfiguration{
			{
				URL: config.URL,
			},
		}
	}

	// Create API client
	client := cloudsmith.NewAPIClient(configuration)

	return &CloudsmithDestination{
		config: config,
		logger: logger,
		client: client,
	}
}

// Put stores an artifact in the Cloudsmith repository.
func (d *CloudsmithDestination) Put(ctx context.Context, artifact *core.Artifact, content io.Reader, raw bool) (string, error) {
	d.logger.WithFields(logrus.Fields{
		"artifact_name": artifact.Name,
		"destination":   d.config.DestPath,
		"url":           d.config.URL,
		"raw":           raw,
	}).Info("Cloudsmith destination: Starting Put operation")

	// Parse destination path to get owner and repo
	const expectedPathParts = 2

	parts := strings.Split(d.config.DestPath, "/")
	if len(parts) != expectedPathParts {
		return "", fmt.Errorf("invalid destination path format, expected 'owner/repo', got: %s", d.config.DestPath)
	}

	owner, repo := parts[0], parts[1]

	// For now, use the simplified approach of uploading directly to a fixed URL
	// This would need to be adjusted based on actual Cloudsmith API documentation
	packageURL, err := d.uploadArtifact(ctx, owner, repo, artifact, content, raw)
	if err != nil {
		return "", fmt.Errorf("failed to upload artifact: %w", err)
	}

	d.logger.WithFields(logrus.Fields{
		"artifact_name": artifact.Name,
		"package_url":   packageURL,
	}).Info("Cloudsmith destination: Successfully uploaded artifact")

	return packageURL, nil
}

// uploadArtifact performs the upload using a simplified approach.
func (d *CloudsmithDestination) uploadArtifact(ctx context.Context, owner, repo string, artifact *core.Artifact, content io.Reader, raw bool) (string, error) {
	// Read content into buffer for upload
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	// For raw packages, we'll use Cloudsmith's upload endpoint
	// This is a simplified implementation that would need refinement based on actual API docs
	uploadURL := fmt.Sprintf("https://upload.cloudsmith.io/%s/%s/%s", owner, repo, artifact.Name)

	// Create PUT request for file upload
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, strings.NewReader(string(contentBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	// Set authentication headers using basic auth
	req.SetBasicAuth(d.config.User, d.config.Password)
	req.Header.Set("Content-Type", "application/octet-stream")

	d.logger.WithFields(logrus.Fields{
		"upload_url":    uploadURL,
		"content_size":  len(contentBytes),
		"artifact_name": artifact.Name,
	}).Debug("Cloudsmith: Uploading artifact")

	// Execute request using default HTTP client
	client := &http.Client{Timeout: cloudsmithTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload artifact: %w", err)
	}

	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Extract version from artifact metadata or use default
	version := "1.0.0"

	if artifact.Metadata != nil {
		if v, exists := artifact.Metadata["version"]; exists {
			version = v
		}
	}

	// Construct a basic package URL for the uploaded artifact
	packageURL := fmt.Sprintf("https://cloudsmith.io/%s/%s/packages/detail/raw/%s/%s/", owner, repo, artifact.Name, version)

	d.logger.WithFields(logrus.Fields{
		"artifact_name": artifact.Name,
		"package_url":   packageURL,
	}).Debug("Cloudsmith: Artifact uploaded successfully")

	return packageURL, nil
}

// Exists checks if an artifact already exists in the Cloudsmith repository.
func (d *CloudsmithDestination) Exists(ctx context.Context, artifact *core.Artifact, raw bool) (bool, error) {
	// Parse destination path to get owner and repo
	const expectedPathParts = 2

	parts := strings.Split(d.config.DestPath, "/")
	if len(parts) != expectedPathParts {
		return false, fmt.Errorf("invalid destination path format, expected 'owner/repo', got: %s", d.config.DestPath)
	}

	owner, repo := parts[0], parts[1]

	d.logger.WithFields(logrus.Fields{
		"artifact_name": artifact.Name,
		"owner":         owner,
		"repo":          repo,
	}).Debug("Cloudsmith: Checking if artifact exists")

	// Create authentication context for the API call
	auth := context.WithValue(ctx, cloudsmith.ContextBasicAuth, cloudsmith.BasicAuth{
		UserName: d.config.User,
		Password: d.config.Password,
	})

	// Use the PackagesList API to search for existing packages
	packages, httpResp, err := d.client.PackagesApi.PackagesList(auth, owner, repo).Query(artifact.Name).Execute()
	if err != nil {
		// If we can't access the repository, assume package doesn't exist
		if httpResp != nil && (httpResp.StatusCode == http.StatusNotFound || httpResp.StatusCode == http.StatusForbidden) {
			return false, nil
		}

		return false, fmt.Errorf("failed to check package existence: %w", err)
	}
	defer httpResp.Body.Close()

	// Check if any packages match our artifact name
	for _, pkg := range packages {
		if pkg.Name.IsSet() && pkg.Name.Get() != nil && *pkg.Name.Get() == artifact.Name {
			d.logger.WithField("artifact_name", artifact.Name).Debug("Cloudsmith: Artifact already exists")

			return true, nil
		}
	}

	d.logger.WithField("artifact_name", artifact.Name).Debug("Cloudsmith: Artifact does not exist")

	return false, nil
}

// BuildDestinationPath constructs the destination path for an artifact.
func (d *CloudsmithDestination) BuildDestinationPath(artifact *core.Artifact, raw bool) (string, error) {
	// Remove the source path strip prefix if configured
	destPath := artifact.Name
	if d.config.SourcePathStrip != "" && strings.HasPrefix(artifact.Name, d.config.SourcePathStrip) {
		destPath = strings.TrimPrefix(artifact.Name, d.config.SourcePathStrip)
		// Remove leading slash if present
		destPath = strings.TrimPrefix(destPath, "/")
	}

	// For Cloudsmith, the destination path is just the artifact name
	// The owner/repo is handled in the API calls
	return destPath, nil
}

// Validate checks if the CloudsmithDestination configuration is valid.
func (d *CloudsmithDestination) Validate() error {
	if d.config.URL == "" {
		return fmt.Errorf("cloudsmith URL is required")
	}

	if d.config.User == "" {
		return fmt.Errorf("cloudsmith user is required")
	}

	if d.config.Password == "" {
		return fmt.Errorf("cloudsmith password is required")
	}

	if d.config.DestPath == "" {
		return fmt.Errorf("cloudsmith destination path is required")
	}

	// Validate that DestPath is in the format "owner/repo"
	const expectedPathParts = 2

	parts := strings.Split(d.config.DestPath, "/")
	if len(parts) != expectedPathParts || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("cloudsmith destination path must be in the format 'owner/repo', got: %s", d.config.DestPath)
	}

	return nil
}

// String returns a string representation of the CloudsmithDestination.
func (d *CloudsmithDestination) String() string {
	return fmt.Sprintf("CloudsmithDestination{URL: %s, DestPath: %s}", d.config.URL, d.config.DestPath)
}
