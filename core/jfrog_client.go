package core

import (
	"fmt"
	"strings"

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
func (c *JFrogClient) UploadGenericArtifact(file, targetPath string, properties []string) error {
	opts := artifactory.UploadServiceOptions{
		FailFast: true,
	}

	params := services.NewUploadParams()
	params.Pattern = file
	params.Target = targetPath

	if len(properties) > 0 {
		targetProps, err := CreateTargetProperties(properties)
		if err != nil {
			return err
		}

		params.SetTargetProps(targetProps)
	}

	totalUploaded, totalFailed, err := c.UploadFiles(opts, params)
	if err != nil {
		return err
	}

	fmt.Printf("Total uploaded: %d\n", totalUploaded)
	fmt.Printf("Total failed: %d\n", totalFailed)

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
