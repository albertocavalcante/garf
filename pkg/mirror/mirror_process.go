package mirror

import (
	"fmt"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/core/config"
	"github.com/albertocavalcante/garf/pkg/destinations"
	"github.com/albertocavalcante/garf/pkg/sources"
	"github.com/sirupsen/logrus"
)

// SetupSource creates and configures a mirroring source based on the provided configuration.
// It currently supports a GitHub source when the source type is "github" and returns an error if an unsupported
// source type is specified or if the created source fails validation.
func SetupSource(logger *logrus.Logger, config *config.Config) (core.Source, error) {
	var source core.Source

	switch config.Source.Type {
	case "github":
		source = sources.NewGitHubSource(logger)
	case "generic":
		source = sources.NewGenericSource(logger)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", config.Source.Type)
	}

	if err := source.Validate(); err != nil {
		return nil, fmt.Errorf("source validation failed: %w", err)
	}

	return source, nil
}

// SetupDestination creates and configures a destination based on the provided configuration.
func SetupDestination(logger *logrus.Logger, config *config.Config) (core.Destination, error) {
	var destination core.Destination

	switch config.Destination.Type {
	case "jfrog":
		destConfig := destinations.JFrogConfig{
			URL:             config.Destination.URL,
			User:            config.Destination.User,
			Password:        config.Destination.Password,
			DestPath:        config.Destination.DestPath,
			SourcePathStrip: config.Destination.SourcePathStrip,
		}
		destination = destinations.NewJFrogDestination(destConfig, logger)
	default:
		return nil, fmt.Errorf("unsupported destination type: %s", config.Destination.Type)
	}

	if err := destination.Validate(); err != nil {
		return nil, fmt.Errorf("destination validation failed: %w", err)
	}

	return destination, nil
}

// ProcessMirrorResults processes the results from the mirror operation.
func ProcessMirrorResults(logger *logrus.Logger, results <-chan MirrorResult) error {
	var lastErr error

	for result := range results {
		if result.Error != nil {
			logger.WithError(result.Error).Error("Failed to mirror artifact")
			lastErr = result.Error
		} else {
			logger.WithFields(logrus.Fields{
				"artifact": result.Artifact.Name,
				"version":  result.Artifact.Version,
			}).Info("Successfully mirrored artifact")
		}
	}

	return lastErr
}
