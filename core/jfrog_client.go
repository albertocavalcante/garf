package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/io/progress"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/config"
)

const (
	// propertyParts is the expected number of parts in a property (key=value).
	propertyParts = 2
)

// JFrogConfig contains the required properties to connect to JFrog Artifactory.
type JFrogConfig struct {
	Url      string
	User     string
	Password string
}

// JFrogClient is an embedded ArtifactoryServicesManager which adds convenience methods.
type JFrogClient struct {
	artifactory.ArtifactoryServicesManager
}

// NewJFrogClient creates a new JFrogClient using the provided configuration.
// It sets up the Artifactory connection details, builds the required service configuration,
// and initializes the underlying ArtifactoryServicesManager.
// If initialization fails at any step, an error is returned.
func NewJFrogClient(jc *JFrogConfig) (*JFrogClient, error) {
	// Create Artifactory details
	rtDetails := auth.NewArtifactoryDetails()
	rtDetails.SetUrl(jc.Url)
	rtDetails.SetUser(jc.User)
	rtDetails.SetPassword(jc.Password)

	// Build service configuration
	serviceConfig, err := config.NewConfigBuilder().SetServiceDetails(rtDetails).Build()
	if err != nil {
		return nil, err
	}

	// Create ArtifactoryServicesManager
	rtManager, err := artifactory.New(serviceConfig)
	if err != nil {
		return nil, err
	}

	return &JFrogClient{rtManager}, nil
}

// setupProgressReader returns an io.Reader that wraps the provided content to report progress during read operations.
// If the progress function is nil, the original reader is returned unchanged.
// When the reader supports seeking, it attempts to determine the total content size
// to enable accurate progress reporting.
func setupProgressReader(content io.Reader, progressFunc progress.ProgressFunc) io.Reader {
	if progressFunc == nil {
		return content
	}

	// Get the total size if available
	var total int64

	if seeker, ok := content.(io.Seeker); ok {
		// Save current position
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err == nil {
			// Seek to end to get size
			total, err = seeker.Seek(0, io.SeekEnd)
			if err == nil {
				// Restore position
				_, _ = seeker.Seek(pos, io.SeekStart)
			}
		}
	}

	// Create progress reader
	return progress.NewReader(content, total, progressFunc)
}

// UploadGenericArtifact uploads a generic artifact to Artifactory.
func (c *JFrogClient) UploadGenericArtifact(
	artifact *core.Artifact,
	content io.Reader,
	progressFunc progress.ProgressFunc,
) error {
	// Create a temporary file to upload from
	tempFile, err := os.CreateTemp("", "artifact-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Setup progress reader if needed
	if progressFunc != nil {
		reader := setupProgressReader(content, progressFunc)
		content = reader
	}

	// Copy content to temp file
	if _, err := io.Copy(tempFile, content); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Create upload parameters
	params := services.NewUploadParams()
	params.Pattern = tempFile.Name()
	params.Target = artifact.Location
	params.Flat = true
	params.Recursive = false
	params.IncludeDirs = false

	// Set properties if any
	if len(artifact.Metadata) > 0 {
		// Convert metadata map to slice of strings
		var props []string
		for k, v := range artifact.Metadata {
			props = append(props, fmt.Sprintf("%s=%s", k, v))
		}

		targetProps, err := CreateTargetProperties(props)
		if err != nil {
			return err
		}

		params.SetTargetProps(targetProps)
	}

	// Upload the artifact
	_, totalFailed, err := c.UploadFiles(artifactory.UploadServiceOptions{}, params)
	if err != nil {
		return fmt.Errorf("failed to upload artifact: %w", err)
	}

	if totalFailed > 0 {
		return fmt.Errorf("failed to upload %d files", totalFailed)
	}

	return nil
}

// CreateTargetProperties converts string properties to a utils.Properties struct.
func CreateTargetProperties(properties []string) (*utils.Properties, error) {
	targetProps := utils.NewProperties()

	for _, prop := range properties {
		key, value, err := ParseProperty(prop)
		if err != nil {
			return nil, err
		}

		targetProps.AddProperty(key, value)
	}

	return targetProps, nil
}

// ParseProperty splits a property string into key and value components.
func ParseProperty(prop string) (key, value string, err error) {
	parts := strings.SplitN(prop, "=", propertyParts)
	if len(parts) != propertyParts {
		return "", "", fmt.Errorf("invalid property format '%s', expected 'key=value'", prop)
	}

	return parts[0], parts[1], nil
}

// When the reader supports seeking, it attempts to determine the total content size
// to enable accurate progress reporting.
func (c *JFrogClient) Put(ctx context.Context, url string, content io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, content)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
