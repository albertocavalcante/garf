package core_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/albertocavalcante/garf/core"
	"github.com/stretchr/testify/require"
)

func TestIsZipFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "zip file with lowercase extension",
			filePath: "file.zip",
			expected: true,
		},
		{
			name:     "zip file with uppercase extension",
			filePath: "file.ZIP",
			expected: true,
		},
		{
			name:     "zip file with mixed case extension",
			filePath: "file.Zip",
			expected: true,
		},
		{
			name:     "non-zip file",
			filePath: "file.txt",
			expected: false,
		},
		{
			name:     "file with no extension",
			filePath: "file",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := core.IsZipFile(test.filePath)
			require.Equal(t, test.expected, result)
		})
	}
}

// unzipTestCase defines a test case for TestExtractSingleFileFromZip.
type unzipTestCase struct {
	name           string
	fileCount      int
	expectSuccess  bool
	includeDir     bool
	options        *core.ExtractOptions
	expectedErrMsg string
}

// runZipExtractionTest runs a single test case for zip extraction.
func runZipExtractionTest(t *testing.T, test unzipTestCase, tempDir string) {
	t.Helper()
	// Create a test zip file
	zipPath := filepath.Join(tempDir, test.name+".zip")
	createTestZip(t, zipPath, test.fileCount, test.includeDir)

	// Test extraction
	extractedPath, extracted, err := core.ExtractSingleFileFromZip(zipPath, test.options)

	if test.expectSuccess {
		require.NoError(t, err)
		require.True(t, extracted)
		require.NotEmpty(t, extractedPath)
		require.FileExists(t, extractedPath)

		// Verify extraction location if custom destination was provided
		if test.options != nil && test.options.DestinationDir != "" {
			require.Equal(t, test.options.DestinationDir, filepath.Dir(extractedPath),
				"File should be extracted to the specified destination directory")
		}
	} else {
		require.Error(t, err)
		require.False(t, extracted)
		require.Empty(t, extractedPath)

		if test.expectedErrMsg != "" {
			require.Contains(t, err.Error(), test.expectedErrMsg)
		}
	}
}

func TestExtractSingleFileFromZip(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "zip-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a separate temp dir for extraction destination testing
	extractDir, err := os.MkdirTemp("", "zip-extract-")
	require.NoError(t, err)
	defer os.RemoveAll(extractDir)

	// Test cases
	tests := []unzipTestCase{
		{
			name:          "single file zip",
			fileCount:     1,
			expectSuccess: true,
		},
		{
			name:          "single file zip with custom destination",
			fileCount:     1,
			expectSuccess: true,
			options: &core.ExtractOptions{
				DestinationDir: extractDir,
			},
		},
		{
			name:           "multiple files zip",
			fileCount:      3,
			expectSuccess:  false,
			expectedErrMsg: "zip file contains 3 files, expected exactly 1",
		},
		{
			name:           "empty zip",
			fileCount:      0,
			expectSuccess:  false,
			expectedErrMsg: "zip file contains 0 files, expected exactly 1",
		},
		{
			name:           "directory only",
			fileCount:      1,
			includeDir:     true,
			expectSuccess:  false,
			expectedErrMsg: "zip file contains a directory, not a file",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runZipExtractionTest(t, test, tempDir)
		})
	}
}

// Helper function to create a test zip file.
func createTestZip(t *testing.T, zipPath string, fileCount int, includeDir bool) {
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	if includeDir {
		// Add a directory entry
		dirHeader := &zip.FileHeader{
			Name:   "testdir/",
			Method: zip.Deflate,
		}
		dirHeader.SetMode(0o755 | os.ModeDir)
		_, err = zipWriter.CreateHeader(dirHeader)
		require.NoError(t, err)
	} else {
		// Add regular files
		for range make([]struct{}, fileCount) {
			fileName := "testfile.txt"
			file, err := zipWriter.Create(fileName)
			require.NoError(t, err)

			_, err = file.Write([]byte("test content"))
			require.NoError(t, err)
		}
	}
}

func TestUnzipFile(t *testing.T) {
	// Test implementation goes here
}
