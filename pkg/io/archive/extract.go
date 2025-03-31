package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

// HandleZipExtraction extracts the zip file located at the artifact's location into a temporary directory, preserving its original name. If extraction fails, the temporary directory is removed and an error is returned; otherwise, it returns the full path to the extracted file.
func HandleZipExtraction(params ZipExtractionParams) (string, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "garf-unzip-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	extractOpts := ExtractOptions{
		DestinationDir:       tempDir,
		PreserveOriginalName: true,
	}

	extractedPath, err := ExtractSingleFile(params.Artifact.Location, extractOpts)
	if err != nil {
		// Clean up the temporary directory if extraction fails
		os.RemoveAll(tempDir)

		return "", fmt.Errorf("failed to extract zip: %w", err)
	}

	return extractedPath, nil
}

// ProcessArtifact processes the provided artifact by conditionally extracting zip files before mirroring them.
// 
// If the artifact's location indicates a zip file, it extracts the file to a temporary directory using HandleZipExtraction,
// then updates the artifact's name and location to point to the extracted file. Otherwise, it mirrors the artifact as is.
// An error is returned if any extraction or mirroring step fails.
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
		artifact.Name = filepath.Base(extractedPath)
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
