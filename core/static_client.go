package core

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadArtifact downloads a GitHub Release artifact to a temporary directory.
func DownloadArtifact(artifactURL string) (string, error) {
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

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to copy artifact content: %w", err)
	}

	return filePath, nil
}
