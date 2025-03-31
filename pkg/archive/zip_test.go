package archive_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/albertocavalcante/garf/pkg/archive"
	"github.com/stretchr/testify/require"
)

func TestIsZipFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "valid zip file",
			path:     "test.zip",
			expected: true,
		},
		{
			name:     "uppercase extension",
			path:     "test.ZIP",
			expected: true,
		},
		{
			name:     "non-zip file",
			path:     "test.txt",
			expected: false,
		},
		{
			name:     "path with directory",
			path:     "dir/test.zip",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := archive.IsZipFile(tt.path)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestZipError(t *testing.T) {
	tests := []struct {
		name          string
		op            string
		underlyingErr error
		expectedMsg   string
	}{
		{
			name:          "with underlying error",
			op:            "extract",
			underlyingErr: os.ErrNotExist,
			expectedMsg:   "zip operation extract failed: file does not exist",
		},
		{
			name:          "without underlying error",
			op:            "extract",
			underlyingErr: nil,
			expectedMsg:   "zip operation extract failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &archive.ZipError{
				Op:  tt.op,
				Err: tt.underlyingErr,
			}

			require.Equal(t, tt.expectedMsg, err.Error())
		})
	}
}

func TestExtractSingleFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zip_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test case 1: Single file in zip
	singleFileZip := filepath.Join(tempDir, "single.zip")
	err = createTestZip(singleFileZip, []string{"test.txt"})
	require.NoError(t, err)

	extractedFile, err := archive.ExtractSingleFile(singleFileZip, archive.ExtractOptions{PreserveOriginalName: true})
	require.NoError(t, err)
	require.Equal(t, "test.txt", filepath.Base(extractedFile))

	// Test case 2: Multiple files in zip
	multiFileZip := filepath.Join(tempDir, "multi.zip")
	err = createTestZip(multiFileZip, []string{"test1.txt", "test2.txt"})
	require.NoError(t, err)

	_, err = archive.ExtractSingleFile(multiFileZip, archive.ExtractOptions{PreserveOriginalName: true})
	if zipErr, ok := err.(*archive.ZipError); ok {
		require.Equal(t, "extract", zipErr.Op)
		require.Contains(t, zipErr.Error(), "multiple files found")
	} else {
		t.Fatal("expected ZipError")
	}

	// Test case 3: Directory in zip
	dirZip := filepath.Join(tempDir, "dir.zip")
	err = createZipWithDirectory(dirZip)
	require.NoError(t, err)

	_, err = archive.ExtractSingleFile(dirZip, archive.ExtractOptions{PreserveOriginalName: true})
	if zipErr, ok := err.(*archive.ZipError); ok {
		require.Equal(t, "extract", zipErr.Op)
		require.Contains(t, zipErr.Error(), "directory found")
	} else {
		t.Fatal("expected ZipError")
	}

	// Test case 4: Non-existent zip file
	_, err = archive.ExtractSingleFile("nonexistent.zip", archive.ExtractOptions{PreserveOriginalName: true})
	require.Error(t, err)
}

// Helper function to create a test zip file with multiple files.
func createTestZip(zipPath string, files []string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for i := range files {
		filename := files[i]

		f, err := w.Create(filename)
		if err != nil {
			return err
		}

		_, err = f.Write([]byte("test content"))
		if err != nil {
			return err
		}
	}

	return nil
}

// Helper function to create a zip file with a directory.
func createZipWithDirectory(zipPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	_, err = w.Create("testdir/")

	return err
}
