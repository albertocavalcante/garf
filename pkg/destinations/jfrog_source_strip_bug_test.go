package destinations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestJFrogDestination_SourcePathStrippingBugFix tests the specific bug reported by the user
// where source path stripping was not preserving the GitHub structure correctly.
func TestJFrogDestination_SourcePathStrippingBugFix(t *testing.T) {
	// Create a logger with debug level to see all the enhanced logging
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test the exact scenario from the bug report
	tests := []struct {
		name            string
		sourceURL       string
		sourcePathStrip string
		destPath        string
		expectedPath    string
		description     string
	}{
		{
			name:            "Bug fix: JFrog staging to prod with GitHub URL",
			sourceURL:       "https://artifactory.example.com/staging-repo/path/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			sourcePathStrip: "artifactory.example.com/staging-repo/path/",
			destPath:        "prod-repo/binaries",
			expectedPath:    "/prod-repo/binaries/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			description:     "Should strip staging prefix but preserve github.com/owner/repo/version/filename structure",
		},
		{
			name:            "Standard GitHub release URL without stripping",
			sourceURL:       "https://github.com/owner/project/releases/download/1.0.0/app-1.0.0-linux.exe",
			sourcePathStrip: "",
			destPath:        "generic-local",
			expectedPath:    "/generic-local/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			description:     "Standard GitHub release URL should work as before",
		},
		{
			name:            "Another stripped GitHub URL pattern",
			sourceURL:       "https://artifactory.corp.net/staging/github.com/company/tool/v2.1.0/binary.zip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			destPath:        "prod-repo",
			expectedPath:    "/prod-repo/github.com/company/tool/v2.1.0/binary.zip",
			description:     "Should handle different staging patterns correctly",
		},
		{
			name:            "Generic URL with stripping",
			sourceURL:       "https://artifactory.corp.net/staging/some-host.com/path/to/file.zip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			destPath:        "generic-local",
			expectedPath:    "/generic-local/file.zip",
			description:     "Generic URLs should still work with stripping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track the request path for verification
			var requestPath string

			// Create a test server that captures the request path
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestPath = r.URL.Path
				// Remove matrix parameters for comparison
				if idx := strings.Index(requestPath, ";"); idx != -1 {
					requestPath = requestPath[:idx]
				}
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()

			// Create destination with source path strip configuration
			config := JFrogConfig{
				URL:             server.URL,
				User:            "testuser",
				Password:        "testpass",
				DestPath:        tt.destPath,
				SourcePathStrip: tt.sourcePathStrip,
			}

			dest := NewJFrogDestination(config, logger)

			// Create test artifact with the exact filename from the URL
			artifactName := "app-1.0.0-linux.exe"
			if tt.name == "Another stripped GitHub URL pattern" {
				artifactName = "binary.zip"
			} else if tt.name == "Generic URL with stripping" {
				artifactName = "file.zip"
			}

			artifact := &core.Artifact{
				Name:     artifactName,
				Location: tt.sourceURL,
				Metadata: map[string]string{
					"test": "value",
				},
			}

			// Perform the upload (this triggers the path building logic)
			err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
			require.NoError(t, err, tt.description)

			// Verify the path is correct
			require.Equal(t, tt.expectedPath, requestPath, tt.description)

			// Log for debugging
			t.Logf("✅ %s", tt.description)
			t.Logf("   Source URL: %s", tt.sourceURL)
			t.Logf("   Strip prefix: %s", tt.sourcePathStrip)
			t.Logf("   Dest path: %s", tt.destPath)
			t.Logf("   Expected path: %s", tt.expectedPath)
			t.Logf("   Actual path: %s", requestPath)
		})
	}
}

// TestJFrogDestination_BuildTargetURLWithBugFix tests the BuildTargetURL method specifically
// to ensure the bug fix works at the URL building level.
func TestJFrogDestination_BuildTargetURLWithBugFix(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name            string
		sourceURL       string
		sourcePathStrip string
		destPath        string
		expectedPath    string
		description     string
	}{
		{
			name:            "Exact bug scenario - URL building",
			sourceURL:       "https://artifactory.example.com/staging-repo/path/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			sourcePathStrip: "artifactory.example.com/staging-repo/path/",
			destPath:        "prod-repo/binaries",
			expectedPath:    "prod-repo/binaries/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			description:     "BuildTargetURL should generate correct path with GitHub structure preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := JFrogConfig{
				URL:             "https://jfrog.example.com",
				User:            "testuser",
				Password:        "testpass",
				DestPath:        tt.destPath,
				SourcePathStrip: tt.sourcePathStrip,
			}

			dest := NewJFrogDestination(config, logger)

			artifact := &core.Artifact{
				Name:     "app-1.0.0-linux.exe",
				Location: tt.sourceURL,
			}

			// Test BuildTargetURL method
			targetURL, err := dest.BuildTargetURL(artifact, false)
			require.NoError(t, err, "BuildTargetURL should not return error")

			require.Equal(t, tt.expectedPath, targetURL.Path, "BuildTargetURL should generate correct path with GitHub structure preserved")

			t.Logf("✅ %s", tt.description)
			t.Logf("   Target URL: %s", targetURL)
			t.Logf("   Expected path: %s", tt.expectedPath)
			t.Logf("   Actual path: %s", targetURL.Path)
		})
	}
}
