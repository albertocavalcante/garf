package garf_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/albertocavalcante/garf"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Test helper functions to reduce code duplication

// createTestClient creates a standard test client with default configuration.
func createTestClient(t *testing.T) *garf.Client {
	t.Helper()

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
	})
	require.NoError(t, err)

	return client
}

// createDryRunRequest creates a standard dry run mirror request.
func createDryRunRequest(source, destination string) garf.MirrorRequest {
	return garf.MirrorRequest{
		Source:      source,
		Destination: destination,
		DryRun:      true,
		DryRunMode:  "all",
	}
}

// createDryRunRequestWithStrip creates a dry run mirror request with source path stripping.
func createDryRunRequestWithStrip(source, destination, sourcePathStrip string) garf.MirrorRequest {
	return garf.MirrorRequest{
		Source:          source,
		Destination:     destination,
		SourcePathStrip: sourcePathStrip,
		DryRun:          true,
		DryRunMode:      "all",
	}
}

// runConcurrentMirrorTest runs a concurrent test with the given number of goroutines and iterations.
func runConcurrentMirrorTest(t *testing.T, client *garf.Client, numGoroutines, numIterations int, requestFunc func(int, int) garf.MirrorRequest) {
	t.Helper()

	ctx := context.Background()
	start := make(chan struct{})

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()
			<-start // Wait for signal to start

			for j := 0; j < numIterations; j++ {
				request := requestFunc(id, j)
				_, err := client.Mirror(ctx, request)
				require.NoError(t, err)
			}
		}(i)
	}

	// Start all goroutines at once
	close(start)
	wg.Wait()
}

func TestNewClient(t *testing.T) {
	tests := getNewClientTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runNewClientTest(t, tt)
		})
	}
}

func getNewClientTestCases() []struct {
	name        string
	config      garf.Config
	expectError bool
	errorMsg    string
} {
	var tests []struct {
		name        string
		config      garf.Config
		expectError bool
		errorMsg    string
	}

	tests = append(tests, getValidConfigTestCases()...)
	tests = append(tests, getInvalidConfigTestCases()...)
	tests = append(tests, getCustomConfigTestCases()...)

	return tests
}

func getValidConfigTestCases() []struct {
	name        string
	config      garf.Config
	expectError bool
	errorMsg    string
} {
	return []struct {
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
	}
}

func getInvalidConfigTestCases() []struct {
	name        string
	config      garf.Config
	expectError bool
	errorMsg    string
} {
	return []struct {
		name        string
		config      garf.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing JFrogURL",
			config: garf.Config{
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
			},
			expectError: true,
			errorMsg:    "registry URL is required",
		},
		{
			name: "missing JFrogUser",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogPassword: "testpass",
			},
			expectError: true,
			errorMsg:    "registry user is required",
		},
		{
			name: "missing JFrogPassword",
			config: garf.Config{
				JFrogURL:  "https://test.jfrog.io/artifactory",
				JFrogUser: "testuser",
			},
			expectError: true,
			errorMsg:    "registry password is required",
		},
	}
}

func getCustomConfigTestCases() []struct {
	name        string
	config      garf.Config
	expectError bool
	errorMsg    string
} {
	return []struct {
		name        string
		config      garf.Config
		expectError bool
		errorMsg    string
	}{
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
}

func runNewClientTest(t *testing.T, tt struct {
	name        string
	config      garf.Config
	expectError bool
	errorMsg    string
},
) {
	client, err := garf.NewClient(tt.config)

	if tt.expectError {
		require.Error(t, err)
		require.Contains(t, err.Error(), tt.errorMsg)
		require.Nil(t, client)
	} else {
		require.NoError(t, err)
		require.NotNil(t, client)
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
			errorMsg:    "registry URL is required",
		},
		{
			name: "missing user",
			config: garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogPassword: "testpass",
			},
			expectError: true,
			errorMsg:    "registry user is required",
		},
		{
			name: "missing password",
			config: garf.Config{
				JFrogURL:  "https://test.jfrog.io/artifactory",
				JFrogUser: "testuser",
			},
			expectError: true,
			errorMsg:    "registry password is required",
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

// TestMirrorRequest_Validation tests the MirrorRequest struct validation.
func TestMirrorRequest_Validation(t *testing.T) {
	client := createTestClient(t)

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
	client := createTestClient(t)
	ctx := context.Background()

	// First call to the same destination
	_, err := client.Mirror(ctx, createDryRunRequest(
		"https://github.com/test/repo/releases/download/v1.0.0/file1.txt",
		"test-repo",
	))
	require.NoError(t, err)

	// Second call to the same destination - should reuse cached destination
	_, err = client.Mirror(ctx, createDryRunRequest(
		"https://github.com/test/repo/releases/download/v1.0.0/file2.txt",
		"test-repo", // Same destination as above
	))
	require.NoError(t, err)

	// Third call to a different destination - should create new destination
	_, err = client.Mirror(ctx, createDryRunRequest(
		"https://github.com/test/repo/releases/download/v1.0.0/file3.txt",
		"different-repo", // Different destination
	))
	require.NoError(t, err)

	// Verify that we have exactly 2 destinations cached
	require.Equal(t, 2, client.GetCachedDestinationsCount())
	require.True(t, client.IsCachedDestination("test-repo"))
	require.True(t, client.IsCachedDestination("different-repo"))
}

func TestClient_SourcePathStripping(t *testing.T) {
	client := createTestClient(t)
	ctx := context.Background()

	tests := getSourcePathStrippingTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSourcePathStrippingTest(t, client, ctx, tt)
		})
	}
}

func getSourcePathStrippingTestCases() []struct {
	name            string
	sourceURL       string
	sourcePathStrip string
	destination     string
	expectError     bool
} {
	return []struct {
		name            string
		sourceURL       string
		sourcePathStrip string
		destination     string
		expectError     bool
	}{
		{
			name:            "JFrog to JFrog mirroring with staging prefix strip",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			destination:     "prod-repo",
			expectError:     false,
		},
		{
			name:            "JFrog to JFrog mirroring with host strip",
			sourceURL:       "https://artifactory.corp.net/repo/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net",
			destination:     "prod-repo",
			expectError:     false,
		},
		{
			name:            "No stripping when prefix not found",
			sourceURL:       "https://github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
			sourcePathStrip: "artifactory.corp.net/staging/",
			destination:     "prod-repo",
			expectError:     false,
		},
		{
			name:            "Empty strip prefix",
			sourceURL:       "https://github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
			sourcePathStrip: "",
			destination:     "prod-repo",
			expectError:     false,
		},
		{
			name:            "Valid request with source path stripping",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			destination:     "my-repo",
			expectError:     false,
		},
	}
}

func runSourcePathStrippingTest(t *testing.T, client *garf.Client, ctx context.Context, tt struct {
	name            string
	sourceURL       string
	sourcePathStrip string
	destination     string
	expectError     bool
},
) {
	request := createDryRunRequestWithStrip(tt.sourceURL, tt.destination, tt.sourcePathStrip)

	result, err := client.Mirror(ctx, request)

	if tt.expectError {
		require.Error(t, err)
		require.Nil(t, result)
	} else {
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, tt.sourceURL, result.Source)
		require.NoError(t, result.Error)
	}
}

func TestClient_DestinationCachingWithSourcePathStrip(t *testing.T) {
	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// First call with source path stripping
	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:          "https://artifactory.corp.net/staging/github.com/test/repo/releases/download/v1.0.0/file1.txt",
		Destination:     "test-repo",
		SourcePathStrip: "artifactory.corp.net/staging/",
		DryRun:          true,
		DryRunMode:      "all",
	})
	require.NoError(t, err)

	// Second call to the same destination with same stripping - should reuse cached destination
	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:          "https://artifactory.corp.net/staging/github.com/test/repo/releases/download/v1.0.0/file2.txt",
		Destination:     "test-repo",
		SourcePathStrip: "artifactory.corp.net/staging/", // Same stripping as above
		DryRun:          true,
		DryRunMode:      "all",
	})
	require.NoError(t, err)

	// Third call to the same destination with different stripping - should create new destination
	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:          "https://artifactory.corp.net/other/github.com/test/repo/releases/download/v1.0.0/file3.txt",
		Destination:     "test-repo",
		SourcePathStrip: "artifactory.corp.net/other/", // Different stripping
		DryRun:          true,
		DryRunMode:      "all",
	})
	require.NoError(t, err)

	// Fourth call to the same destination with no stripping - should create another new destination
	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:      "https://github.com/test/repo/releases/download/v1.0.0/file4.txt",
		Destination: "test-repo",
		// No SourcePathStrip
		DryRun:     true,
		DryRunMode: "all",
	})
	require.NoError(t, err)

	// Verify that we have exactly 3 destinations cached (same repo name but different stripping configs)
	require.Equal(t, 3, client.GetCachedDestinationsCount())
	require.True(t, client.IsCachedDestination("test-repo|strip:artifactory.corp.net/staging/"))
	require.True(t, client.IsCachedDestination("test-repo|strip:artifactory.corp.net/other/"))
	require.True(t, client.IsCachedDestination("test-repo"))
}

func TestMirrorRequest_WithSourcePathStrip(t *testing.T) {
	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
	})
	require.NoError(t, err)

	// Test that a request with SourcePathStrip is valid
	request := garf.MirrorRequest{
		Source:          "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
		Destination:     "my-repo",
		SourcePathStrip: "artifactory.corp.net/staging/",
		Properties:      map[string]string{"type": "binary"},
		Raw:             false,
		Unzip:           false,
		DryRun:          true,
		DryRunMode:      "all",
	}

	err = client.ValidateRequest(request)
	require.NoError(t, err)

	// Test that the request can be processed
	ctx := context.Background()
	result, err := client.Mirror(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, request.Source, result.Source)
	require.NoError(t, result.Error)
}

func TestClient_RaceConditionInDestinationCaching(t *testing.T) {
	client := createTestClient(t)

	numGoroutines := 100
	numIterations := 10

	// Test concurrent access to the same destination
	runConcurrentMirrorTest(t, client, numGoroutines, numIterations, func(id, j int) garf.MirrorRequest {
		return createDryRunRequest(
			fmt.Sprintf("https://github.com/test/repo/releases/download/v1.0.0/file-%d-%d.txt", id, j),
			"test-repo", // Same destination for all
		)
	})

	// Verify that we have exactly 1 destination cached (no race condition corruption)
	require.Equal(t, 1, client.GetCachedDestinationsCount())
	require.True(t, client.IsCachedDestination("test-repo"))
}

func TestClient_RaceConditionWithDifferentSourcePathStrip(t *testing.T) {
	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "testuser",
		JFrogPassword: "testpass",
	})
	require.NoError(t, err)

	ctx := context.Background()
	numGoroutines := 50

	// Use a channel to synchronize goroutine starts
	start := make(chan struct{})

	var wg sync.WaitGroup

	// Test concurrent access with different source path strip configurations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()
			<-start // Wait for signal to start

			// Alternate between different strip configurations
			var sourcePathStrip string

			switch id % 3 {
			case 0:
				sourcePathStrip = "artifactory.corp.net/staging/"
			case 1:
				sourcePathStrip = "artifactory.corp.net/other/"
			case 2:
				sourcePathStrip = "" // No stripping
			}

			_, err := client.Mirror(ctx, garf.MirrorRequest{
				Source:          fmt.Sprintf("https://artifactory.corp.net/staging/github.com/test/repo/releases/download/v1.0.0/file-%d.txt", id),
				Destination:     "test-repo",
				SourcePathStrip: sourcePathStrip,
				DryRun:          true,
				DryRunMode:      "all",
			})
			require.NoError(t, err)
		}(i)
	}

	// Start all goroutines at once
	close(start)
	wg.Wait()

	// Verify that we have exactly 3 destinations cached (one for each strip config)
	require.Equal(t, 3, client.GetCachedDestinationsCount())
	require.True(t, client.IsCachedDestination("test-repo|strip:artifactory.corp.net/staging/"))
	require.True(t, client.IsCachedDestination("test-repo|strip:artifactory.corp.net/other/"))
	require.True(t, client.IsCachedDestination("test-repo"))
}

type validateRequestTestCase struct {
	name        string
	request     garf.MirrorRequest
	expectError bool
	errorMsg    string
}

func TestClient_ValidateRequest(t *testing.T) {
	client := createTestClient(t)
	tests := getValidateRequestTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidateRequestTestCase(t, client, tt)
		})
	}
}

func getValidateRequestTestCases() []validateRequestTestCase {
	return []validateRequestTestCase{
		{
			name: "valid request", request: garf.MirrorRequest{
				Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			},
			expectError: false,
		},
		{name: "missing source", request: garf.MirrorRequest{Destination: "my-repo"}, expectError: true, errorMsg: "source is required"},
		{name: "missing destination", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
		}, expectError: true, errorMsg: "destination is required"},
		{name: "valid dry run with mode", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			DryRun: true, DryRunMode: "upload",
		}, expectError: false},
		{name: "invalid dry run mode", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			DryRun: true, DryRunMode: "invalid",
		}, expectError: true, errorMsg: "invalid dry run mode"},
		{name: "request with properties", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			Properties: map[string]string{"type": "binary", "platform": "linux"},
		}, expectError: false},
		{name: "request with all options", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			Properties: map[string]string{"type": "binary"}, Raw: true, Unzip: true,
			FromFile: "/path/to/local/file", DryRun: true, DryRunMode: "all",
		}, expectError: false},
		{name: "request with source path stripping", request: garf.MirrorRequest{
			Source:      "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
			Destination: "my-repo", SourcePathStrip: "artifactory.corp.net/staging/",
			Properties: map[string]string{"type": "binary"},
		}, expectError: false},
		{name: "valid source path strip", request: garf.MirrorRequest{
			Source:      "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
			Destination: "my-repo", SourcePathStrip: "artifactory.corp.net/staging/",
		}, expectError: false},
		{name: "source path strip with path traversal", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			SourcePathStrip: "../../malicious/path",
		}, expectError: true, errorMsg: "source path strip cannot contain '..' for security reasons"},
		{name: "source path strip with http scheme", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			SourcePathStrip: "http://artifactory.corp.net/staging/",
		}, expectError: true, errorMsg: "source path strip should not include the URL scheme"},
		{name: "source path strip with https scheme", request: garf.MirrorRequest{
			Source: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip", Destination: "my-repo",
			SourcePathStrip: "https://artifactory.corp.net/staging/",
		}, expectError: true, errorMsg: "source path strip should not include the URL scheme"},
	}
}

func runValidateRequestTestCase(t *testing.T, client *garf.Client, tt validateRequestTestCase) {
	err := client.ValidateRequest(tt.request)
	if tt.expectError {
		require.Error(t, err)
		require.Contains(t, err.Error(), tt.errorMsg)
	} else {
		require.NoError(t, err)
	}
}

func TestClient_SourceTypeDetection(t *testing.T) {
	config := garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "test",
		JFrogPassword: "test",
	}

	client, err := garf.NewClient(config)
	require.NoError(t, err)

	tests := []struct {
		name               string
		sourceURL          string
		sourcePathStrip    string
		expectedSourceType string
	}{
		{
			name:               "GitHub URL",
			sourceURL:          "https://github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip:    "",
			expectedSourceType: "github",
		},
		{
			name:               "JFrog URL without strip",
			sourceURL:          "https://artifactory.corp.net/staging/file.zip",
			sourcePathStrip:    "",
			expectedSourceType: "generic",
		},
		{
			name:               "JFrog URL with strip revealing GitHub",
			sourceURL:          "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip:    "artifactory.corp.net/staging/",
			expectedSourceType: "github",
		},
		{
			name:               "JFrog URL with strip revealing non-GitHub",
			sourceURL:          "https://artifactory.corp.net/staging/some-other-host.com/file.zip",
			sourcePathStrip:    "artifactory.corp.net/staging/",
			expectedSourceType: "generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceType := client.DetectSourceType(tt.sourceURL, tt.sourcePathStrip)
			require.Equal(t, tt.expectedSourceType, sourceType)

			// Verify the source is available
			err := client.EnsureSourceAvailable(sourceType)
			require.NoError(t, err)
		})
	}
}

func TestClient_SourcePathStripValidation(t *testing.T) {
	config := garf.Config{
		JFrogURL:      "https://test.jfrog.io/artifactory",
		JFrogUser:     "test",
		JFrogPassword: "test",
	}

	client, err := garf.NewClient(config)
	require.NoError(t, err)

	tests := []struct {
		name            string
		sourcePathStrip string
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "valid path strip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			wantErr:         false,
		},
		{
			name:            "empty path strip",
			sourcePathStrip: "",
			wantErr:         false,
		},
		{
			name:            "path traversal attack",
			sourcePathStrip: "../../malicious",
			wantErr:         true,
			errMsg:          "source path strip cannot contain '..' for security reasons",
		},
		{
			name:            "https scheme",
			sourcePathStrip: "https://malicious.com/",
			wantErr:         true,
			errMsg:          "source path strip should not include the URL scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := garf.MirrorRequest{
				Source:          "https://github.com/example/repo/releases/download/v1.0/file.zip",
				Destination:     "test-repo",
				SourcePathStrip: tt.sourcePathStrip,
			}

			err := client.ValidateRequest(request)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRegistryTypeValidation tests the new registry type validation functionality.
func TestRegistryTypeValidation(t *testing.T) {
	tests := []struct {
		name         string
		registryType string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "empty registry type should be valid (backward compatibility)",
			registryType: "",
			expectError:  false,
		},
		{
			name:         "jfrog registry type should be valid",
			registryType: "jfrog",
			expectError:  false,
		},
		{
			name:         "cloudsmith registry type should be valid",
			registryType: "cloudsmith",
			expectError:  false,
		},
		{
			name:         "invalid registry type should fail",
			registryType: "invalid",
			expectError:  true,
			errorMsg:     "unsupported registry type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
				RegistryType:  tt.registryType,
			}

			err := garf.ValidateConfig(config)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRegistryTypeWithNewClient tests that clients can be created with registry types.
func TestRegistryTypeWithNewClient(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name         string
		registryType string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "empty registry type works (backward compatibility)",
			registryType: "",
			expectError:  false,
		},
		{
			name:         "explicit jfrog registry type works",
			registryType: "jfrog",
			expectError:  false,
		},
		{
			name:         "cloudsmith registry type works in validation",
			registryType: "cloudsmith",
			expectError:  false,
		},
		{
			name:         "invalid registry type fails in validation",
			registryType: "invalid",
			expectError:  true,
			errorMsg:     "unsupported registry type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := garf.Config{
				JFrogURL:      "https://test.jfrog.io/artifactory",
				JFrogUser:     "testuser",
				JFrogPassword: "testpass",
				RegistryType:  tt.registryType,
				Logger:        logger,
			}

			client, err := garf.NewClient(config)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				require.Equal(t, tt.registryType, client.Config.RegistryType)
			}
		})
	}
}

// TestCloudsmithRegistryImplementation tests that Cloudsmith registry type properly
// creates destinations and validates configuration, verifying the implementation works.
func TestCloudsmithRegistryImplementation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a client with Cloudsmith registry type
	config := garf.Config{
		RegistryURL:      "https://api.cloudsmith.io",
		RegistryUser:     "testuser",
		RegistryPassword: "testpass",
		RegistryType:     "cloudsmith",
		Logger:           logger,
	}

	client, err := garf.NewClient(config)
	require.NoError(t, err, "Client creation should succeed with cloudsmith registry type")
	require.NotNil(t, client)
	require.Equal(t, "cloudsmith", client.Config.RegistryType)

	// Test that the client successfully handles Cloudsmith registry type
	// We'll test this indirectly through a dry-run mirror operation
	request := garf.MirrorRequest{
		Source:      "https://github.com/example/repo/releases/download/v1.0.0/test-artifact.zip",
		Destination: "owner/repo",
		DryRun:      true,  // Use dry run to test destination creation without network calls
		DryRunMode:  "all", // Ensure we skip all operations including download
	}

	// Attempt to mirror in dry-run mode - this should succeed and create the destination
	ctx := context.Background()
	result, err := client.Mirror(ctx, request)

	// In dry-run mode, this should succeed without actual network calls
	require.NoError(t, err, "Mirror should succeed for Cloudsmith registry in dry-run mode")
	require.NotNil(t, result, "Result should not be nil in dry-run mode")
	require.Equal(t, request.Source, result.Source, "Source should match")
	require.Contains(t, result.DestinationPath, "owner/repo", "Destination path should contain owner/repo")
	require.Contains(t, result.DestinationPath, "test-artifact.zip", "Destination path should contain artifact name")

	t.Logf("Successfully verified Cloudsmith registry implementation with destination: %s", result.DestinationPath)
}
