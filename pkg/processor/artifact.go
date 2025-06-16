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
	tempDir, cleanupFunc, err := createTempDir(logger)
	if err != nil {
		return err
	}
	defer cleanupFunc()

	extractedPath, err := extractZipFile(artifact.Location, tempDir, logger)
	if err != nil {
		return err
	}

	updateArtifactForExtraction(artifact, extractedPath, logger)

	return mirrorExtractedFile(ctx, mirror, artifact, opts, logger)
}

// createTempDir creates a temporary directory with cleanup function.
func createTempDir(logger *logrus.Logger) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "garf-unzip-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logIfNotNil(logger, "temp_dir", tempDir, "Created temporary directory for ZIP extraction")

	var cleanupDone bool

	cleanupFunc := func() {
		if !cleanupDone {
			cleanupDone = true

			if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
				logWarnIfNotNil(logger, cleanupErr, "temp_dir", tempDir, "Failed to clean up temporary directory")
			} else {
				logDebugIfNotNil(logger, "temp_dir", tempDir, "Cleaned up temporary directory")
			}
		}
	}

	return tempDir, cleanupFunc, nil
}

// extractZipFile extracts a single file from ZIP.
func extractZipFile(location, tempDir string, logger *logrus.Logger) (string, error) {
	extractOpts := archive.ExtractOptions{
		DestinationDir:       tempDir,
		PreserveOriginalName: true,
	}

	extractedPath, err := archive.ExtractSingleFile(location, extractOpts)
	if err != nil {
		return "", fmt.Errorf("failed to extract zip: %w", err)
	}

	if logger != nil {
		logger.WithFields(logrus.Fields{
			"original_location": location,
			"extracted_path":    extractedPath,
			"temp_dir":          tempDir,
		}).Info("Successfully extracted ZIP file")
	}

	return extractedPath, nil
}

// updateArtifactForExtraction updates artifact with extracted file info.
func updateArtifactForExtraction(artifact *core.Artifact, extractedPath string, logger *logrus.Logger) {
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
}

// mirrorExtractedFile mirrors the extracted file and handles results.
func mirrorExtractedFile(ctx context.Context, mirror *mirror.DefaultMirror, artifact *core.Artifact, opts *core.MirrorOptions, logger *logrus.Logger) error {
	extractResults := mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)

	var mirrorErrors []error

	for extractResult := range extractResults {
		if extractResult.Error != nil {
			mirrorErrors = append(mirrorErrors, extractResult.Error)
		} else if logger != nil {
			logger.WithFields(logrus.Fields{
				"artifact_name":    extractResult.Artifact.Name,
				"destination_path": extractResult.DestinationPath,
			}).Info("Successfully mirrored extracted file")
		}
	}

	if len(mirrorErrors) > 0 {
		return fmt.Errorf("failed to mirror extracted files: %v", mirrorErrors)
	}

	return nil
}

// Helper functions for conditional logging.
func logIfNotNil(logger *logrus.Logger, key, value, message string) {
	if logger != nil {
		logger.WithField(key, value).Debug(message)
	}
}

func logWarnIfNotNil(logger *logrus.Logger, err error, key, value, message string) {
	if logger != nil {
		logger.WithError(err).WithField(key, value).Warn(message)
	}
}

func logDebugIfNotNil(logger *logrus.Logger, key, value, message string) {
	if logger != nil {
		logger.WithField(key, value).Debug(message)
	}
}
