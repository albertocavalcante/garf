// Package archive provides functionality for handling various archive formats.
package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultDirMode is the default mode for created directories.
const DefaultDirMode = 0o755

// ExtractOptions defines the options for extracting files from a ZIP archive.
type ExtractOptions struct {
	// DestinationDir specifies where the extracted file should be placed
	DestinationDir string
	// PreserveOriginalName determines if the original filename should be kept
	PreserveOriginalName bool
}

// ZipError represents an error that occurred during ZIP operations.
type ZipError struct {
	Op  string
	Err error
}

// Error returns the error message for the zip operation.
func (e *ZipError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("zip operation %s failed: %v", e.Op, e.Err)
	}

	return fmt.Sprintf("zip operation %s failed", e.Op)
}

func (e *ZipError) Unwrap() error {
	return e.Err
}

// IsZipFile returns true if the provided file path ends with the ".zip" extension (case-insensitive).
func IsZipFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".zip")
}

// ValidateExtractOptions checks the extraction options for ZIP archives.
// Since no validations are required (e.g., the destination directory is optional), the function always returns nil.
func validateExtractOptions() error {
	// DestinationDir is optional, so no validation needed
	return nil
}

// createDestinationDir creates the destination directory if it doesn't exist.
func createDestinationDir(destDir string) error {
	if err := os.MkdirAll(destDir, DefaultDirMode); err != nil {
		return &ZipError{Op: "create_dest_dir", Err: fmt.Errorf("failed to create destination directory: %w", err)}
	}

	return nil
}

// findSingleFileInZip finds the first non-directory file in the ZIP archive.
func findSingleFileInZip(r *zip.Reader) (*zip.File, error) {
	var singleFile *zip.File

	hasDirectory := false

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			hasDirectory = true

			continue
		}

		if singleFile != nil {
			return nil, &ZipError{Op: "extract", Err: fmt.Errorf("multiple files found")}
		}

		singleFile = f
	}

	if singleFile == nil {
		if hasDirectory {
			return nil, &ZipError{Op: "extract", Err: fmt.Errorf("directory found")}
		}

		return nil, &ZipError{Op: "extract", Err: fmt.Errorf("no files found in archive")}
	}

	return singleFile, nil
}

// extractFileToDestination extracts a single file from the ZIP archive to the destination.
func extractFileToDestination(zipFile *zip.File, destPath string) error {
	rc, err := zipFile.Open()
	if err != nil {
		return &ZipError{Op: "open_file", Err: fmt.Errorf("failed to open file in archive: %w", err)}
	}
	defer rc.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return &ZipError{Op: "create_dest_file", Err: fmt.Errorf("failed to create destination file: %w", err)}
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, rc); err != nil {
		return &ZipError{Op: "copy_file", Err: fmt.Errorf("failed to copy file contents: %w", err)}
	}

	return nil
}

// ExtractSingleFile extracts a single file from the ZIP archive at zipPath.
// 
// It validates the extraction options and creates the destination directory (if specified).
// The function opens the ZIP file, locates a single non-directory file within it, and
// extracts that file to a destination path derived from options. If DestinationDir is provided,
// the file is extracted into that directory; otherwise, the file retains its original name.
// Returns the full path to the extracted file, or an error if any operation fails.
func ExtractSingleFile(zipPath string, options ExtractOptions) (string, error) {
	if err := validateExtractOptions(); err != nil {
		return "", err
	}

	// Create destination directory if specified
	if options.DestinationDir != "" {
		if err := createDestinationDir(options.DestinationDir); err != nil {
			return "", err
		}
	}

	// Open the ZIP file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", &ZipError{Op: "open", Err: fmt.Errorf("failed to open zip file: %w", err)}
	}
	defer reader.Close()

	// Find the single file to extract
	singleFile, err := findSingleFileInZip(&reader.Reader)
	if err != nil {
		return "", err
	}

	// Determine the destination path
	destPath := singleFile.Name
	if options.DestinationDir != "" {
		destPath = filepath.Join(options.DestinationDir, destPath)
	}

	// Extract the file
	if err := extractFileToDestination(singleFile, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}
