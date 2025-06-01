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
	testCases := []JFrogTestCase{
		{
			Name:            "Standard GitHub release URL without stripping",
			ArtifactName:    "artifact.zip",
			SourceURL:       "https://artifactory.example.com/staging-repo/path/github.com/owner/project/1.0.0/app-1.0.0-linux.exe",
			SourcePathStrip: "",
			DestPath:        "generic-local",
			ExpectedPath:    "/generic-local/artifactory.example.com/staging-repo/path/github.com/owner/project/1.0.0/artifact.zip",
		},
		{
			Name:            "Another stripped GitHub URL pattern",
			ArtifactName:    "binary.zip",
			SourceURL:       StandardGitHubArtURL,
			SourcePathStrip: "github.com/org/repo",
			DestPath:        "generic-local",
			ExpectedPath:    "/generic-local/releases/download/v1.0.0/binary.zip",
		},
		{
			Name:            "Generic URL with stripping",
			ArtifactName:    "file.zip",
			SourceURL:       StandardGenericArtURL,
			SourcePathStrip: "myget.org",
			DestPath:        "generic-local",
			ExpectedPath:    "/generic-local/F/feed/package/file.zip",
		},
		{
			Name:            "BCR URL without stripping (original bug scenario)",
			ArtifactName:    "source.json",
			SourceURL:       StandardBCRArtURL,
			SourcePathStrip: "",
			DestPath:        "generic-local",
			ExpectedPath:    "/generic-local/bcr.bazel.build/modules/lib/v1.2.3/source.json",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			env := SetupTestServer(t, DefaultJFrogConfig(), nil)
			defer env.Server.Close()

			env.Config.DestPath = tt.DestPath
			env.Config.SourcePathStrip = tt.SourcePathStrip

			destination := NewJFrogDestination(env.Config, env.Logger)

			// Use the helper function to get artifact name
			artifactName := GetArtifactNameFromTestCase(tt)
			artifact := DefaultArtifact(t, artifactName, tt.SourceURL, nil)

			// Perform the upload (this triggers the path building logic)
			destinationPath, err := destination.Put(context.Background(), artifact, strings.NewReader("test content"), false)
			require.NoError(t, err, tt.Description)

			// Verify the path is correct
			require.Equal(t, tt.ExpectedPath, env.RequestPath, tt.Description)

			// Log for debugging
			t.Logf("✅ %s", tt.Description)
			t.Logf("   Source URL: %s", tt.SourceURL)
			t.Logf("   Strip prefix: %s", tt.SourcePathStrip)
			t.Logf("   Dest path: %s", tt.DestPath)
			t.Logf("   Expected path: %s", tt.ExpectedPath)
			t.Logf("   Actual path: %s", env.RequestPath)
			t.Logf("   Destination path returned: %s", destinationPath)
		})
	}
}

// TestJFrogDestination_BuildTargetURLWithBugFix tests the BuildTargetURL method specifically
// to ensure the bug fix works at the URL building level.
func TestJFrogDestination_BuildTargetURLWithBugFix(t *testing.T) {
	tests := []JFrogTestCase{
		{
			Name:            "Exact bug scenario - URL building",
			ArtifactName:    "app-1.0.0-linux.exe",
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

			dest := NewJFrogDestination(config, NewTestLogger())

			// Use the helper function to get artifact name
			artifactName := GetArtifactNameFromTestCase(tt)
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
