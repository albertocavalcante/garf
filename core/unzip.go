package core

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractOptions contains options for extracting files from archives.
type ExtractOptions struct {
	// DestinationDir is the directory where files should be extracted
	// If empty, files will be extracted to the same directory as the archive
	DestinationDir string
}

// ExtractSingleFileFromZip attempts to extract a single file from a zip archive.
// If the zip contains exactly one file, it extracts that file to the destination path.
// Returns the path to the extracted file and a flag indicating if extraction occurred.
func ExtractSingleFileFromZip(zipFilePath string, options *ExtractOptions) (string, bool, error) {
	// Open the zip file
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return "", false, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Check if this is a zip with a single file
	if len(reader.File) != 1 {
		return "", false, fmt.Errorf("zip file contains %d files, expected exactly 1", len(reader.File))
	}

	zipFile := reader.File[0]

	// Skip directories
	if zipFile.FileInfo().IsDir() {
		return "", false, fmt.Errorf("zip file contains a directory, not a file")
	}

	// Determine the directory for the extracted file
	var dirPath string
	if options != nil && options.DestinationDir != "" {
		dirPath = options.DestinationDir
	} else {
		dirPath = filepath.Dir(zipFilePath)
	}

	// Get the file name for the extracted file
	fileName := zipFile.Name

	// Remove any path information from the file name
	fileName = filepath.Base(fileName)

	// Create the full path for the extracted file
	extractedPath := filepath.Join(dirPath, fileName)

	// Create the output file
	outFile, err := os.Create(extractedPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Open the file inside the zip
	inFile, err := zipFile.Open()
	if err != nil {
		return "", false, fmt.Errorf("failed to open file inside zip: %w", err)
	}
	defer inFile.Close()

	// Copy the file contents
	_, err = io.Copy(outFile, inFile)
	if err != nil {
		return "", false, fmt.Errorf("failed to extract file from zip: %w", err)
	}

	return extractedPath, true, nil
}

// IsZipFile checks if the given file path has a .zip extension.
func IsZipFile(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".zip"
}
