package core

import (
	"fmt"
	"os"

	"github.com/albertocavalcante/garf/pkg/pointer"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/config"
)

// NewJFrogRtManager creates a new ArtifactoryServicesManager
func NewJFrogRtManager() (*artifactory.ArtifactoryServicesManager, error) {
	rtDetails := auth.NewArtifactoryDetails()
	// Leave hardcoded for now
	rtDetails.SetUrl("https://albertocavalcante.jfrog.io/artifactory")

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

// JFrogArtifactoryVersion prints the version of the Artifactory instance
func JFrogArtifactoryVersion() error {
	rtManagerPtr, err := NewJFrogRtManager()
	if err != nil {
		return err
	}
	rtManager := pointer.Deref(rtManagerPtr, nil)

	version, err := rtManager.GetVersion()
	if err != nil {
		return err
	}

	fmt.Printf("Version: %s\n", version)
	return nil
}
