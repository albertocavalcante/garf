package core

import (
	"fmt"
	"io"
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

// NewJFrogClient creates a new JFrogClient (ArtifactoryServicesManager).
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

// UploadGenericArtifact uploads a generic artifact to Artifactory.
func (c *JFrogClient) UploadGenericArtifact(artifact *core.Artifact, content io.Reader, progressFunc progress.ProgressFunc) error {
	// Create upload parameters
	params := services.NewUploadParams()
	params.Pattern = artifact.Location
	params.Target = artifact.Location

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

	// If we have a progress function, wrap the reader
	if progressFunc != nil {
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
		reader := progress.NewReader(content, total, progressFunc)
		content = reader
	}

	// Upload the artifact
	opts := artifactory.UploadServiceOptions{
		FailFast: true,
	}

	_, totalFailed, err := c.UploadFiles(opts, params)
	if err != nil {
		return err
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
