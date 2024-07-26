package core

import (
	"fmt"
	"os"

	"github.com/albertocavalcante/garf/artifact"
	"github.com/albertocavalcante/garf/pkg/pointer"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/config"
)

// NewJFrogRtManager creates a new ArtifactoryServicesManager
func NewJFrogRtManager() (*artifactory.ArtifactoryServicesManager, error) {
	rtDetails := auth.NewArtifactoryDetails()

	// Artifactory URL
	jfrogUrl := os.Getenv("JFROG_URL")
	if jfrogUrl == "" {
		return nil, fmt.Errorf("JFROG_URL environment variable is required")
	}
	rtDetails.SetUrl(jfrogUrl)

	// Artifactory Authentication
	jfrogUser := os.Getenv("JFROG_USER")
	jfrogPassword := os.Getenv("JFROG_PASSWORD")
	if jfrogUser == "" || jfrogPassword == "" {
		return nil, fmt.Errorf("JFROG_USER and JFROG_PASSWORD environment variables are required")
	}
	rtDetails.SetUser(jfrogUser)
	rtDetails.SetPassword(jfrogPassword)

	serviceConfig, err := config.NewConfigBuilder().SetServiceDetails(rtDetails).Build()
	if err != nil {
		return nil, err
	}

	rtManager, err := artifactory.New(serviceConfig)
	if err != nil {
		return nil, err
	}

	return &rtManager, nil
}

// UploadGenericArtifact uploads a generic artifact to Artifactory
func UploadGenericArtifact(file string, repoKey string, coordinates *artifact.ArtifactCoordinates) error {
	rtManagerPtr, err := NewJFrogRtManager()
	if err != nil {
		return err
	}
	rtManager := pointer.Deref(rtManagerPtr, nil)

	params := services.NewUploadParams()
	params.Pattern = file
	params.Target = repoKey + "/" + coordinates.UrlPath()

	totalUploaded, totalFailed, err := rtManager.UploadFiles(params)
	if err != nil {
		return err
	}
	fmt.Printf("Total uploaded: %d\n", totalUploaded)
	fmt.Printf("Total failed: %d\n", totalFailed)

	return nil
}
