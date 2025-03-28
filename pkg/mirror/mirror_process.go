package mirror

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/albertocavalcante/garf/pkg/archive"
	"github.com/albertocavalcante/garf/pkg/config"
	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/destinations"
	"github.com/albertocavalcante/garf/pkg/sources"
	"github.com/sirupsen/logrus"
)

// downloadArtifact downloads an artifact from the source and saves it to a temporary file.
func downloadArtifact(
	ctx context.Context,
	source core.Source,
	artifact *core.Artifact,
	tmpDir string,
) (string, io.ReadCloser, error) {
	content, err := source.Get(ctx, artifact)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	// Create a temporary file for the artifact
	tmpFile := filepath.Join(tmpDir, filepath.Base(artifact.Location))

	file, err := os.Create(tmpFile)
	if err != nil {
		content.Close()

		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Copy the content to the temporary file
	if _, err := io.Copy(file, content); err != nil {
		file.Close()
		content.Close()

		return "", nil, fmt.Errorf("failed to copy artifact content: %w", err)
	}

	file.Close()

	return tmpFile, content, nil
}

// processZipArtifact processes a zip artifact if needed.
func processZipArtifact(artifact *core.Artifact, tmpFile string, tmpDir string) error {
	if !strings.HasSuffix(artifact.Location, ".zip") {
		return nil
	}

	options := archive.ExtractOptions{
		DestinationDir:       tmpDir,
		PreserveOriginalName: true,
	}
	if _, err := archive.ExtractSingleFile(tmpFile, options); err != nil {
		return fmt.Errorf("failed to unzip artifact: %w", err)
	}

	return nil
}

// uploadToDestinations uploads the artifact to all configured destinations.
func uploadToDestinations(
	ctx context.Context,
	artifact *core.Artifact,
	content io.Reader,
	destinations []core.Destination,
) error {
	var wg sync.WaitGroup

	errChan := make(chan error, len(destinations))

	for _, dest := range destinations {
		wg.Add(1)

		go func(d core.Destination) {
			defer wg.Done()

			if err := d.Put(ctx, artifact, content); err != nil {
				errChan <- fmt.Errorf("failed to upload to destination: %w", err)

				return
			}
		}(dest)
	}

	// Wait for all uploads to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	return lastErr
}

// ProcessAndUploadArtifact processes and uploads an artifact to all configured destinations.
func ProcessAndUploadArtifact(
	ctx context.Context,
	logger *logrus.Logger,
	artifact *core.Artifact,
	source core.Source,
	destinations []core.Destination,
	opts *core.MirrorOptions,
) error {
	// Create a temporary directory for processing
	tmpDir, err := os.MkdirTemp("", "garf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download the artifact
	tmpFile, content, err := downloadArtifact(ctx, source, artifact, tmpDir)
	if err != nil {
		return err
	}
	defer content.Close()

	// Process the artifact if it's a zip file
	if err := processZipArtifact(artifact, tmpFile, tmpDir); err != nil {
		return err
	}

	// Upload to all destinations
	return uploadToDestinations(ctx, artifact, content, destinations)
}

// SetupSource creates and configures a source based on the provided configuration.
func SetupSource(logger *logrus.Logger, config *config.Config) (core.Source, error) {
	var source core.Source

	switch config.Source.Type {
	case "github":
		source = sources.NewGitHubSource(logger)
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
			URL:      config.Destination.URL,
			User:     config.Destination.User,
			Password: config.Destination.Password,
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
