package core

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/progress"
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

// NewJFrogClient creates and returns a new JFrogClient configured for Artifactory interactions.
// It constructs connection details from the provided JFrogConfig (URL, user, and password),
// builds the corresponding service configuration, and initializes the underlying ArtifactoryServicesManager.
// The function returns the initialized client, or an error if any step of the initialization process fails.
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

// SetupProgressReader returns a reader that tracks read progress using the provided callback.
// If no progress function is supplied, the original reader is returned unchanged.
// When the underlying reader supports seeking, it attempts to determine the total size
// of the content to enable more accurate progress reporting.
func setupProgressReader(content io.Reader, progressFunc progress.ProgressFunc) (io.Reader, error) {
	if progressFunc == nil {
		return content, nil
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
	return progress.NewReader(content, total, progressFunc), nil
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
		return fmt.Errorf("failed to create temp file: %v", err)
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Setup progress reader if needed
	if progressFunc != nil {
		reader, err := setupProgressReader(content, progressFunc)
		if err != nil {
			return err
		}

		content = reader
	}

	// Copy content to temp file
	if _, err := io.Copy(tempFile, content); err != nil {
		return fmt.Errorf("failed to write to temp file: %v", err)
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
	_, totalFailed, err := c.ArtifactoryServicesManager.UploadFiles(artifactory.UploadServiceOptions{}, params)
	if err != nil {
		return fmt.Errorf("failed to upload artifact: %v", err)
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
