package archive

import (
	"context"
	"fmt"
	"os"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/sirupsen/logrus"
)

// ZipExtractionParams holds the parameters for zip extraction.
type ZipExtractionParams struct {
	Ctx      context.Context
	Logger   *logrus.Logger
	Artifact *core.Artifact
	Opts     *core.MirrorOptions
}

// HandleZipExtraction handles the extraction of a zip file and returns the path to the extracted file.
func HandleZipExtraction(params ZipExtractionParams) (string, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "garf-unzip-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	extractOpts := ExtractOptions{
		DestinationDir:       tempDir,
		PreserveOriginalName: true,
	}

	extractedPath, err := ExtractSingleFile(params.Artifact.Location, extractOpts)
	if err != nil {
		return "", fmt.Errorf("failed to extract zip: %w", err)
	}

	return extractedPath, nil
}

// ProcessArtifact processes a single artifact, handling zip extraction if needed.
func ProcessArtifact(
	ctx context.Context,
	logger *logrus.Logger,
	mirror *mirror.DefaultMirror,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
) error {
	if IsZipFile(artifact.Location) {
		extractParams := ZipExtractionParams{
			Ctx:      ctx,
			Logger:   logger,
			Artifact: artifact,
			Opts:     opts,
		}

		extractedPath, err := HandleZipExtraction(extractParams)
		if err != nil {
			return fmt.Errorf("failed to handle zip extraction: %w", err)
		}

		// Update the artifact name and location for the extracted file
		artifact.Name = extractedPath
		artifact.Location = extractedPath

		// Mirror the extracted file
		results := mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)
		for result := range results {
			if result.Error != nil {
				return result.Error
			}
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
