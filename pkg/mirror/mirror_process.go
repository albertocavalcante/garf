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

// downloadArtifact retrieves the specified artifact from the source and writes its contents to a temporary file within the given directory.
// It returns the full path to the temporary file along with a ReadCloser for the artifact's content (which must be closed by the caller) and an error if any step of the process fails.
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

// processZipArtifact extracts the contents of a zip artifact into a temporary directory if the artifact's location ends with ".zip".
// If the artifact is a zip file, it extracts the file at tmpFile to tmpDir while preserving the original file name.
// If extraction fails, it returns an error describing the failure; otherwise, it returns nil.
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

// uploadToDestinations concurrently uploads the given artifact's content to each provided destination.
// It spawns a separate goroutine for each destination, waits for all uploads to complete,
// and returns the last error encountered if any upload fails.
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

// ProcessAndUploadArtifact orchestrates the artifact mirroring process by downloading the artifact from the source,
// processing it if it is a ZIP archive, and uploading it to all configured destinations.
// It creates a temporary directory for intermediate processing, which is cleaned up automatically.
// Returns an error if any step in the download, processing, or upload operations fails.
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
// It supports a GitHub source type and returns an error if the source type is unsupported
// or if the created source fails validation.
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

// SetupDestination creates and configures a destination based on the provided configuration. It currently supports JFrog destinations, returning an error if an unsupported destination type is specified or if the destination fails validation.
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

// ProcessMirrorResults logs the outcomes of artifact mirroring received from the results channel.
// It iterates over each result, logging error details when an artifact fails to mirror and success information,
// including the artifact's name and version, when the mirroring succeeds. The function returns the last error
// encountered, or nil if all mirror operations were successful.
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
