package core

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/albertocavalcante/garf/pkg/progress"
)

// DownloadArtifact downloads a GitHub release artifact from the specified URL into a temporary directory.
// It creates a temporary directory, retrieves the artifact via an HTTP GET request, and saves the file using the base name from the URL.
// If a non-nil progress function is provided, the download progress is tracked and reported.
// The function returns the full file path of the downloaded artifact, or an error if any step of the download process fails.
func DownloadArtifact(artifactURL string, progressFunc progress.ProgressFunc) (string, error) {
	tempDir, err := os.MkdirTemp("", "garf-download-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	filename := filepath.Base(artifactURL)
	filePath := filepath.Join(tempDir, filename)

	resp, err := http.Get(artifactURL)
	if err != nil {
		return "", fmt.Errorf("failed to download artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status code: %d", resp.StatusCode)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Get total size if available
	var total int64
	if resp.ContentLength > 0 {
		total = resp.ContentLength
	}

	// Create progress reader if progress function is provided
	var reader io.Reader = resp.Body
	if progressFunc != nil {
		reader = progress.NewReader(resp.Body, total, progressFunc)
	}

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", fmt.Errorf("failed to copy artifact content: %w", err)
	}

	return filePath, nil
}
