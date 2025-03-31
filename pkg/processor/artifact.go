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

// ProcessArtifact processes an artifact based on its type.
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

// handleZipExtraction handles the extraction of a zip file and mirrors the extracted content.
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
