package garf_test

import (
	"context"
	"testing"
	"time"

	"github.com/albertocavalcante/garf"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      garf.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
			},
			expectError: false,
		},
		{
			name: "missing JFrogURL",
			config: garf.Config{
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
			},
			expectError: true,
			errorMsg:    "JFrogURL is required",
		},
		{
			name: "missing JFrogUser",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogPassword: "testpass",
			},
			expectError: true,
			errorMsg:    "JFrogUser is required",
		},
		{
			name: "missing JFrogPassword",
			config: garf.Config{
				JFrogURL:  "https://test.jfrog.io/artifactory",
				JFrogUser: "testuser",
			},
			expectError: true,
			errorMsg:    "JFrogPassword is required",
		},
		{
			name: "config with custom logger",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
				Logger:        logrus.New(),
			},
			expectError: false,
		},
		{
			name: "config with custom timeout and concurrent",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
				Timeout:       10 * time.Minute,
				Concurrent:    8,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := garf.NewClient(tt.config)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
			}
		})
	}
}

func TestExtractArtifactName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "github release url",
			url:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
			expected: "artifact.zip",
		},
		{
			name:     "simple filename",
			url:      "https://example.com/file.tar.gz",
			expected: "file.tar.gz",
		},
		{
			name:     "url with query params",
			url:      "https://example.com/path/file.exe?version=1.0",
			expected: "file.exe",
		},
		{
			name:     "empty url",
			url:      "",
			expected: garf.UnknownArtifactName,
		},
		{
			name:     "url without filename",
			url:      "https://example.com/",
			expected: garf.UnknownArtifactName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := garf.ExtractArtifactName(tt.url)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      garf.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
			},
			expectError: false,
		},
		{
			name:        "empty config",
			config:      garf.Config{},
			expectError: true,
			errorMsg:    "JFrogURL is required",
		},
		{
			name: "missing user",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogPassword: "testpass",
			},
			expectError: true,
			errorMsg:    "JFrogUser is required",
		},
		{
			name: "missing password",
			config: garf.Config{
				JFrogURL:  "https://test.jfrog.io/artifactory",
				JFrogUser: "testuser",
			},
			expectError: true,
			errorMsg:    "JFrogPassword is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := garf.ValidateConfig(tt.config)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_validateRequest(t *testing.T) {
	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     garf.MirrorRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: garf.MirrorRequest{
				Source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
				Destination: "my-repo",
			},
			expectError: false,
		},
		{
			name: "missing source",
			request: garf.MirrorRequest{
				Destination: "my-repo",
			},
			expectError: true,
			errorMsg:    "source is required",
		},
		{
			name: "missing destination",
			request: garf.MirrorRequest{
				Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
			},
			expectError: true,
			errorMsg:    "destination is required",
		},
		{
			name: "valid dry run with mode",
			request: garf.MirrorRequest{
				Source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
				Destination: "my-repo",
				DryRun:      true,
				DryRunMode:  "upload",
			},
			expectError: false,
		},
		{
			name: "invalid dry run mode",
			request: garf.MirrorRequest{
				Source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
				Destination: "my-repo",
				DryRun:      true,
				DryRunMode:  "invalid",
			},
			expectError: true,
			errorMsg:    "invalid dry run mode",
		},
		{
			name: "request with properties",
			request: garf.MirrorRequest{
				Source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
				Destination: "my-repo",
				Properties:  map[string]string{"type": "binary", "platform": "linux"},
			},
			expectError: false,
		},
		{
			name: "request with all options",
			request: garf.MirrorRequest{
				Source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
				Destination: "my-repo",
				Properties:  map[string]string{"type": "binary"},
				Raw:         true,
				Unzip:       true,
				FromFile:    "/path/to/local/file",
				DryRun:      true,
				DryRunMode:  "all",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.ValidateRequest(tt.request)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMirrorRequest_Validation tests the MirrorRequest struct validation.
func TestMirrorRequest_Validation(t *testing.T) {
	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
	})
	require.NoError(t, err)

	// Test that a context timeout is properly handled
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	request := garf.MirrorRequest{
		Source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
		Destination: "my-repo",
	}

	// This should fail due to context timeout, not validation
	result, err := client.Mirror(ctx, request)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "context deadline exceeded")
}

// TestConfig_Defaults tests that default values are properly set.
func TestConfig_Defaults(t *testing.T) {
	config := garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
		// Not setting Timeout, Concurrent, or Logger to test defaults
	}

	client, err := garf.NewClient(config)
	require.NoError(t, err)

	// Check that defaults were set
	require.Equal(t, 30*time.Minute, client.Config.Timeout)
	require.Equal(t, 4, client.Config.Concurrent)
	require.NotNil(t, client.Config.Logger)
	require.Equal(t, logrus.InfoLevel, client.Config.Logger.Level)
}

// TestMirrorResult_Structure tests the MirrorResult struct.
func TestMirrorResult_Structure(t *testing.T) {
	result := &garf.MirrorResult{
		Source:          "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
		DestinationPath: "my-repo/github.com/owner/repo/v1.0.0/artifact.zip",
		Error:           nil,
	}

	require.Equal(t, "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", result.Source)
	require.Equal(t, "my-repo/github.com/owner/repo/v1.0.0/artifact.zip", result.DestinationPath)
	require.NoError(t, result.Error)
}

func TestClient_DestinationCaching(t *testing.T) {
	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// First call to the same destination
	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:      "https://github.com/test/repo/releases/download/v1.0.0/file1.txt",
		Destination: "test-repo",
		DryRun:      true,
		DryRunMode:  "all",
	})
	require.NoError(t, err)

	// Second call to the same destination - should reuse cached destination
	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:      "https://github.com/test/repo/releases/download/v1.0.0/file2.txt",
		Destination: "test-repo", // Same destination as above
		DryRun:      true,
		DryRunMode:  "all",
	})
	require.NoError(t, err)

	// Third call to a different destination - should create new destination
	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:      "https://github.com/test/repo/releases/download/v1.0.0/file3.txt",
		Destination: "different-repo", // Different destination
		DryRun:      true,
		DryRunMode:  "all",
	})
	require.NoError(t, err)

	// Verify that we have exactly 2 destinations cached
	require.Equal(t, 2, client.GetCachedDestinationsCount())
	require.True(t, client.IsCachedDestination("test-repo"))
	require.True(t, client.IsCachedDestination("different-repo"))
}
