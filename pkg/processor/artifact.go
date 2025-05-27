// Package processor provides functionality for processing artifacts.
package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/io/archive"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/sirupsen/logrus"
)

// ProcessArtifact processes an artifact using an appropriate mirroring strategy.
// It checks if the artifact is a zip file by inspecting its location; if so, it extracts
// the contents using handleZipExtraction and mirrors the resulting file, otherwise,
// it mirrors the artifact directly. Any error encountered during extraction or mirroring is returned.
func ProcessArtifact(
	ctx context.Context,
	logger *logrus.Logger,
	mirror *mirror.DefaultMirror,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
) error {
	if archive.IsZipFile(artifact.Location) {
		if err := handleZipExtraction(ctx, logger, mirror, artifact, opts); err != nil {
			return fmt.Errorf("failed to handle zip extraction: %w", err)
		}

		return nil
	}

	// Mirror the artifact as is
	results := mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)
	for result := range results {
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

// handleZipExtraction extracts the zip file located at the artifact's Location into a temporary directory
// while preserving the original file name. It updates the artifact's Name and Location to correspond to the
// extracted file and then mirrors the file using the provided mirror instance.
func handleZipExtraction(
	ctx context.Context,
	logger *logrus.Logger,
	mirror *mirror.DefaultMirror,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "garf-unzip-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	if logger != nil {
		logger.WithField("temp_dir", tempDir).Debug("Created temporary directory for ZIP extraction")
	}

	extractOpts := archive.ExtractOptions{
		DestinationDir:       tempDir,
		PreserveOriginalName: true,
	}

	extractedPath, err := archive.ExtractSingleFile(artifact.Location, extractOpts)
	if err != nil {
		// Clean up the temporary directory if extraction fails
		os.RemoveAll(tempDir)

		return fmt.Errorf("failed to extract zip: %w", err)
	}

	if logger != nil {
		logger.WithFields(logrus.Fields{
			"original_location": artifact.Location,
			"extracted_path":    extractedPath,
			"temp_dir":          tempDir,
		}).Info("Successfully extracted ZIP file")
	}

	// Update the artifact name and location for the extracted file
	originalLocation := artifact.Location
	artifact.Name = filepath.Base(extractedPath)
	artifact.Location = extractedPath

	if logger != nil {
		logger.WithFields(logrus.Fields{
			"original_name":      filepath.Base(originalLocation),
			"extracted_name":     artifact.Name,
			"original_location":  originalLocation,
			"extracted_location": artifact.Location,
		}).Debug("Updated artifact with extracted file information")
	}

	// Mirror the extracted file
	extractResults := mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)
	for extractResult := range extractResults {
		if extractResult.Error != nil {
			// Clean up the temporary directory if mirroring fails
			os.RemoveAll(tempDir)

			return extractResult.Error
		}

		if logger != nil {
			logger.WithFields(logrus.Fields{
				"artifact_name":    extractResult.Artifact.Name,
				"destination_path": extractResult.DestinationPath,
			}).Info("Successfully mirrored extracted file")
		}
	}

	// Clean up the temporary directory after successful mirroring
	if err := os.RemoveAll(tempDir); err != nil {
		if logger != nil {
			logger.WithError(err).WithField("temp_dir", tempDir).Warn("Failed to clean up temporary directory")
		}
		// Don't return error for cleanup failure as the main operation succeeded
	} else if logger != nil {
		logger.WithField("temp_dir", tempDir).Debug("Cleaned up temporary directory")
	}

	return nil
}
