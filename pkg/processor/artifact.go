// Package processor provides functionality for processing artifacts.
package processor

import (
	"context"
	"fmt"
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
	_ *logrus.Logger, // Unused for now, may be used for future logging
	mirror *mirror.DefaultMirror,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
) error {
	if archive.IsZipFile(artifact.Location) {
		if err := handleZipExtraction(ctx, mirror, artifact, opts); err != nil {
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

// handleZipExtraction extracts the zip file located at the artifact's Location into its parent directory
// while preserving the original file name. It updates the artifact's Name and Location to correspond to the
// extracted file and then mirrors the file using the provided mirror instance.
func handleZipExtraction(
	ctx context.Context,
	mirror *mirror.DefaultMirror,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
) error {
	extractOpts := archive.ExtractOptions{
		DestinationDir:       filepath.Dir(artifact.Location),
		PreserveOriginalName: true,
	}

	extractedPath, err := archive.ExtractSingleFile(artifact.Location, extractOpts)
	if err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	// Update the artifact name and location for the extracted file
	artifact.Name = filepath.Base(extractedPath)
	artifact.Location = extractedPath

	// Mirror the extracted file
	extractResults := mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)
	for extractResult := range extractResults {
		if extractResult.Error != nil {
			return extractResult.Error
		}
	}

	return nil
}
