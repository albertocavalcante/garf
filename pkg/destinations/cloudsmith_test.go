package destinations

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewCloudsmithDestination(t *testing.T) {
	config := CloudsmithConfig{
		URL:      "https://api.cloudsmith.io",
		User:     "testuser",
		Password: "testpass",
		DestPath: "owner/repo",
	}

	logger := logrus.New()
	dest := NewCloudsmithDestination(config, logger)

	require.NotNil(t, dest)
	cloudsmithDest, ok := dest.(*CloudsmithDestination)
	require.True(t, ok)
	require.Equal(t, config.URL, cloudsmithDest.config.URL)
	require.Equal(t, config.User, cloudsmithDest.config.User)
	require.Equal(t, config.DestPath, cloudsmithDest.config.DestPath)
}

func TestCloudsmithDestination_Validate(t *testing.T) {
	logger := logrus.New()

	tests := []struct {
		name        string
		config      CloudsmithConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				User:     "testuser",
				Password: "testpass",
				DestPath: "owner/repo",
			},
			expectError: false,
		},
		{
			name: "missing URL",
			config: CloudsmithConfig{
				User:     "testuser",
				Password: "testpass",
				DestPath: "owner/repo",
			},
			expectError: true,
			errorMsg:    "cloudsmith URL is required",
		},
		{
			name: "missing user",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				Password: "testpass",
				DestPath: "owner/repo",
			},
			expectError: true,
			errorMsg:    "cloudsmith user is required",
		},
		{
			name: "missing password",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				User:     "testuser",
				DestPath: "owner/repo",
			},
			expectError: true,
			errorMsg:    "cloudsmith password is required",
		},
		{
			name: "missing dest path",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				User:     "testuser",
				Password: "testpass",
			},
			expectError: true,
			errorMsg:    "cloudsmith destination path is required",
		},
		{
			name: "invalid dest path format - single part",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				User:     "testuser",
				Password: "testpass",
				DestPath: "onlyowner",
			},
			expectError: true,
			errorMsg:    "cloudsmith destination path must be in the format 'owner/repo'",
		},
		{
			name: "invalid dest path format - too many parts",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				User:     "testuser",
				Password: "testpass",
				DestPath: "owner/repo/extra",
			},
			expectError: true,
			errorMsg:    "cloudsmith destination path must be in the format 'owner/repo'",
		},
		{
			name: "invalid dest path format - empty owner",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				User:     "testuser",
				Password: "testpass",
				DestPath: "/repo",
			},
			expectError: true,
			errorMsg:    "cloudsmith destination path must be in the format 'owner/repo'",
		},
		{
			name: "invalid dest path format - empty repo",
			config: CloudsmithConfig{
				URL:      "https://api.cloudsmith.io",
				User:     "testuser",
				Password: "testpass",
				DestPath: "owner/",
			},
			expectError: true,
			errorMsg:    "cloudsmith destination path must be in the format 'owner/repo'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := &CloudsmithDestination{
				config: tt.config,
				logger: logger,
			}

			err := dest.Validate()

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCloudsmithDestination_BuildDestinationPath(t *testing.T) {
	logger := logrus.New()

	tests := []struct {
		name         string
		config       CloudsmithConfig
		artifact     *core.Artifact
		raw          bool
		expectedPath string
	}{
		{
			name: "basic artifact path",
			config: CloudsmithConfig{
				DestPath: "owner/repo",
			},
			artifact: &core.Artifact{
				Name:     "test-artifact.zip",
				Location: "https://github.com/owner/repo/releases/download/v1.0.0/test-artifact.zip",
			},
			raw:          false,
			expectedPath: "owner/repo/test-artifact.zip",
		},
		{
			name: "artifact with source path strip",
			config: CloudsmithConfig{
				DestPath:        "owner/repo",
				SourcePathStrip: "github.com/owner/repo/releases/download/v1.0.0/",
			},
			artifact: &core.Artifact{
				Name:     "github.com/owner/repo/releases/download/v1.0.0/test-artifact.zip",
				Location: "https://github.com/owner/repo/releases/download/v1.0.0/test-artifact.zip",
			},
			raw:          false,
			expectedPath: "owner/repo/test-artifact.zip",
		},
		{
			name: "artifact with leading slash after strip",
			config: CloudsmithConfig{
				DestPath:        "owner/repo",
				SourcePathStrip: "staging/",
			},
			artifact: &core.Artifact{
				Name:     "staging/test-artifact.zip",
				Location: "https://example.com/staging/test-artifact.zip",
			},
			raw:          false,
			expectedPath: "owner/repo/test-artifact.zip",
		},
		{
			name: "no source path strip",
			config: CloudsmithConfig{
				DestPath: "owner/repo",
			},
			artifact: &core.Artifact{
				Name:     "path/to/test-artifact.zip",
				Location: "https://example.com/path/to/test-artifact.zip",
			},
			raw:          false,
			expectedPath: "owner/repo/path/to/test-artifact.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := &CloudsmithDestination{
				config: tt.config,
				logger: logger,
			}

			path, err := dest.BuildDestinationPath(tt.artifact, tt.raw)

			require.NoError(t, err)
			require.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestCloudsmithDestination_String(t *testing.T) {
	config := CloudsmithConfig{
		URL:      "https://api.cloudsmith.io",
		DestPath: "owner/repo",
	}

	dest := &CloudsmithDestination{
		config: config,
		logger: logrus.New(),
	}

	result := dest.String()
	expected := "CloudsmithDestination{URL: https://api.cloudsmith.io, DestPath: owner/repo}"

	require.Equal(t, expected, result)
}

func TestCloudsmithDestination_Put_InvalidDestPath(t *testing.T) {
	config := CloudsmithConfig{
		URL:      "https://api.cloudsmith.io",
		User:     "testuser",
		Password: "testpass",
		DestPath: "invalid-path",
	}

	dest := &CloudsmithDestination{
		config: config,
		logger: logrus.New(),
	}

	artifact := &core.Artifact{
		Name:     "test.zip",
		Location: "https://example.com/test.zip",
	}

	content := strings.NewReader("test content")
	ctx := context.Background()

	_, err := dest.Put(ctx, artifact, content, false)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid destination path format")
}

func TestCloudsmithDestination_Exists_InvalidDestPath(t *testing.T) {
	config := CloudsmithConfig{
		URL:      "https://api.cloudsmith.io",
		User:     "testuser",
		Password: "testpass",
		DestPath: "invalid-path",
	}

	dest := &CloudsmithDestination{
		config: config,
		logger: logrus.New(),
	}

	artifact := &core.Artifact{
		Name:     "test.zip",
		Location: "https://example.com/test.zip",
	}

	ctx := context.Background()

	_, err := dest.Exists(ctx, artifact, false)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid destination path format")
}

func TestCloudsmithDestination_uploadArtifact_ContentReading(t *testing.T) {
	config := CloudsmithConfig{
		URL:      "https://api.cloudsmith.io",
		User:     "testuser",
		Password: "testpass",
		DestPath: "owner/repo",
	}

	dest := &CloudsmithDestination{
		config: config,
		logger: logrus.New(),
	}

	artifact := &core.Artifact{
		Name:     "test.zip",
		Location: "https://example.com/test.zip",
		Metadata: map[string]string{
			"version": "2.0.0",
		},
	}

	// Test with failing reader
	failingReader := &failingReader{}
	ctx := context.Background()

	_, err := dest.uploadArtifact(ctx, "owner", "repo", artifact, failingReader, false)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read content")
}

// failingReader is a helper that always returns an error when reading.
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
