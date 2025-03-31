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

// Extract extracts a zip file to the specified destination directory.
func (e *ZipError) Extract(destDir string) error {
	if err := os.MkdirAll(destDir, DefaultDirMode); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return nil
}

// IsZipFile checks if the given file path has a .zip extension.
func IsZipFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".zip")
}

// validateExtractOptions checks the provided extraction options and returns an error if they are invalid.
// Currently, no validation is performed since the DestinationDir option is optional.
func validateExtractOptions(options ExtractOptions) error {
	// DestinationDir is optional, so no validation needed
	return nil
}

// createDestinationDir creates the destination directory specified by destDir using the default permission mode. It returns a ZipError if the directory cannot be created, or nil if the directory exists or is successfully created.
func createDestinationDir(destDir string) error {
	if err := os.MkdirAll(destDir, DefaultDirMode); err != nil {
		return &ZipError{Op: "create_dest_dir", Err: fmt.Errorf("failed to create destination directory: %w", err)}
	}

	return nil
}

// findSingleFileInZip returns the single non-directory file from the provided ZIP archive.
// It iterates through the archive's entries and ensures that exactly one regular file is present.
// If multiple non-directory files are found, or if no non-directory file exists (even if directories are present),
// it returns a ZipError describing the encountered issue.
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

// extractFileToDestination extracts the contents of a ZIP archive entry to the specified destination path.
// It opens the file from the archive, creates the destination file, and copies its data.
// If any step fails, it returns a ZipError that wraps the underlying error.
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

// ExtractSingleFile extracts a single non-directory file from the specified ZIP archive.
// It validates the extraction options and creates the destination directory if one is provided.
// The function opens the ZIP file at zipPath, locates the sole file eligible for extraction,
// determines its destination path (using the file's original name, optionally joined with the destination directory),
// and extracts the file to that location.
// It returns the destination path if successful or an error if any step fails.
func ExtractSingleFile(zipPath string, options ExtractOptions) (string, error) {
	if err := validateExtractOptions(options); err != nil {
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
