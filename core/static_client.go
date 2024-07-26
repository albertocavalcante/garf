package core

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadArtifact downloads a GitHub Release artifact to a temporary directory and returns the path to the downloaded file.
func DownloadArtifact(artifactURL string) (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "garf-download-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	// defer os.RemoveAll(tempDir) // Clean up the temporary directory when done

	// Get the filename from the URL
	filename := filepath.Base(artifactURL)
	filePath := filepath.Join(tempDir, filename)

	// Download the artifact
	resp, err := http.Get(artifactURL)
	if err != nil {
		return "", fmt.Errorf("failed to download artifact: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status code: %d", resp.StatusCode)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy the content to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to copy artifact content: %w", err)
	}

	return filePath, nil
}
