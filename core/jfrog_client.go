package core

import (
	"fmt"

	"github.com/albertocavalcante/garf/artifact"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/config"
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
func (c *JFrogClient) UploadGenericArtifact(file, repoKey string, coordinates *artifact.ArtifactCoordinates) error {
	params := services.NewUploadParams()
	params.Pattern = file
	params.Target = constructTargetPath(repoKey, coordinates)

	totalUploaded, totalFailed, err := c.UploadFiles(params)
	if err != nil {
		return err
	}

	fmt.Printf("Total uploaded: %d\n", totalUploaded)
	fmt.Printf("Total failed: %d\n", totalFailed)

	return nil
}

// constructTargetPath constructs the target path for uploading the artifact.
func constructTargetPath(repoKey string, coordinates *artifact.ArtifactCoordinates) string {
	return fmt.Sprintf("%s/%s", repoKey, coordinates.UrlPath())
}
