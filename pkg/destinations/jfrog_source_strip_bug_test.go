package destinations

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJFrogDestination_SourcePathStrippingBugFix tests the specific bug reported by the user
// where source path stripping was not preserving the GitHub structure correctly.
func TestJFrogDestination_SourcePathStrippingBugFix(t *testing.T) {
	logger := NewTestLogger()

	tests := []JFrogTestCase{
		{
			Name:            "Bug fix: JFrog staging to prod with GitHub URL",
			SourceURL:       "https://artifactory.example.com/staging-repo/path/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			SourcePathStrip: "artifactory.example.com/staging-repo/path/",
			DestPath:        "prod-repo/binaries",
			ExpectedPath:    "/prod-repo/binaries/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			Description:     "Should strip staging prefix but preserve github.com/owner/repo/version/filename structure",
		},
		{
			Name:            "Standard GitHub release URL without stripping",
			SourceURL:       StandardGitHubArtURL,
			SourcePathStrip: "",
			DestPath:        "generic-local",
			ExpectedPath:    "/generic-local/github.com/org/repo/v1.0.0/artifact.zip",
			Description:     "Standard GitHub release URL should work as before",
		},
		{
			Name:            "Another stripped GitHub URL pattern",
			SourceURL:       "https://artifactory.corp.net/staging/github.com/company/tool/v2.1.0/binary.zip",
			SourcePathStrip: "artifactory.corp.net/staging/",
			DestPath:        "prod-repo",
			ExpectedPath:    "/prod-repo/github.com/company/tool/v2.1.0/binary.zip",
			Description:     "Should handle different staging patterns correctly",
		},
		{
			Name:            "Generic URL with stripping",
			SourceURL:       "https://artifactory.corp.net/staging/some-host.com/path/to/file.zip",
			SourcePathStrip: "artifactory.corp.net/staging/",
			DestPath:        "generic-local",
			ExpectedPath:    "/generic-local/some-host.com/path/to/file.zip",
			Description:     "Generic URLs should preserve structure after stripping",
		},
		{
			Name:            "BCR URL without stripping (original bug scenario)",
			SourceURL:       StandardBCRArtURL,
			SourcePathStrip: "",
			DestPath:        "staging",
			ExpectedPath:    "/staging/bcr.bazel.build/modules/lib/v1.2.3/source.json",
			Description:     "BCR URLs should preserve full structure when no stripping is applied (this was the original bug)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			baseConfig := DefaultJFrogConfig()
			baseConfig.DestPath = tt.DestPath
			baseConfig.SourcePathStrip = tt.SourcePathStrip

			serverEnv := SetupTestServer(t, baseConfig, nil)
			defer serverEnv.Server.Close()

			dest := NewJFrogDestination(serverEnv.Config, logger)

			// Determine artifact name based on URL or test case specifics
			var artifactName string
			switch tt.Name {
			case "Another stripped GitHub URL pattern":
				artifactName = "binary.zip"
			case "Generic URL with stripping":
				artifactName = "file.zip"
			case "BCR URL without stripping (original bug scenario)":
				artifactName = "source.json"
			case "Standard GitHub release URL without stripping":
				artifactName = "artifact.zip"
			default:
				// Extract filename from SourceURL for the default case
				if slash := strings.LastIndex(tt.SourceURL, "/"); slash >= 0 {
					artifactName = tt.SourceURL[slash+1:]
				} else {
					artifactName = "app-1.0.0-linux.exe" // Fallback default
				}
			}

			artifact := DefaultArtifact(t, artifactName, tt.SourceURL, map[string]string{"test": "value"})

			// Perform the upload (this triggers the path building logic)
			destinationPath, err := dest.Put(context.Background(), artifact, strings.NewReader("test content"), false)
			require.NoError(t, err, tt.Description)

			// Verify the path is correct
			require.Equal(t, tt.ExpectedPath, serverEnv.RequestPath, tt.Description)

			// Log for debugging
			t.Logf("✅ %s", tt.Description)
			t.Logf("   Source URL: %s", tt.SourceURL)
			t.Logf("   Strip prefix: %s", tt.SourcePathStrip)
			t.Logf("   Dest path: %s", tt.DestPath)
			t.Logf("   Expected path: %s", tt.ExpectedPath)
			t.Logf("   Actual path: %s", serverEnv.RequestPath)
			t.Logf("   Destination path returned: %s", destinationPath)
		})
	}
}

// TestJFrogDestination_BuildTargetURLWithBugFix tests the BuildTargetURL method specifically
// to ensure the bug fix works at the URL building level.
func TestJFrogDestination_BuildTargetURLWithBugFix(t *testing.T) {
	logger := NewTestLogger()

	tests := []JFrogTestCase{
		{
			Name:            "Exact bug scenario - URL building",
			SourceURL:       "https://artifactory.example.com/staging-repo/path/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			SourcePathStrip: "artifactory.example.com/staging-repo/path/",
			DestPath:        "prod-repo/binaries",
			ExpectedPath:    "/prod-repo/binaries/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			Description:     "BuildTargetURL should generate correct path with GitHub structure preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			config := DefaultJFrogConfig()
			config.DestPath = tt.DestPath
			config.SourcePathStrip = tt.SourcePathStrip

			dest := NewJFrogDestination(config, logger)

			// Extract filename from SourceURL for artifact creation
			var artifactName string
			if slash := strings.LastIndex(tt.SourceURL, "/"); slash >= 0 {
				artifactName = tt.SourceURL[slash+1:]
			} else {
				artifactName = "app-1.0.0-linux.exe" // Fallback
			}
			artifact := DefaultArtifact(t, artifactName, tt.SourceURL, nil)

			// Test BuildTargetURL method
			targetURL, err := dest.BuildTargetURL(artifact, false)
			require.NoError(t, err, "BuildTargetURL should not return error")

			// targetURL.Path will have a leading slash if the base URL has a scheme/host (which DefaultJFrogConfig().URL does)
			actualPath := targetURL.Path
			// tt.ExpectedPath now also includes the leading slash for direct comparison
			require.Equal(t, tt.ExpectedPath, actualPath, "BuildTargetURL should generate correct path with GitHub structure preserved")

			t.Logf("✅ %s", tt.Description)
			t.Logf("   Target URL: %s", targetURL.String())
			t.Logf("   Expected path: %s", tt.ExpectedPath)
			t.Logf("   Actual path: %s", actualPath)
		})
	}
}
