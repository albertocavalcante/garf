package artifact

import (
	"fmt"
	"net/url"
	"testing"
)

func TestExtractCoordinatesFromURL(t *testing.T) {
	type testCase struct {
		artifactURL         string
		expectedCoordinates *ArtifactCoordinates
		expectedError       error
	}

	testCases := []testCase{
		{
			artifactURL: "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe",
			expectedCoordinates: &ArtifactCoordinates{
				Host:     "github.com",
				Org:      "bazelbuild",
				Repo:     "bazel",
				Version:  "7.2.1",
				Artifact: "bazel_nojdk-7.2.1-windows-x86_64.exe",
				RawPath:  "bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe",
			},
			expectedError: nil,
		},
		{
			artifactURL: "https://example.com/path/to/artifact.zip",
			expectedCoordinates: &ArtifactCoordinates{
				Host:     "example.com",
				Artifact: "artifact.zip",
				RawPath:  "path/to/artifact.zip",
			},
			expectedError: nil,
		},
		{
			artifactURL:         "invalid-url",
			expectedCoordinates: nil,
			expectedError: fmt.Errorf("failed to parse URL: %w", &url.Error{
				Op:  "parse",
				URL: "invalid-url",
				Err: fmt.Errorf("invalid URI for request"),
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.artifactURL, func(t *testing.T) {
			coordinates, err := ExtractCoordinatesFromURL(tc.artifactURL)

			if tc.expectedError != nil {
				if err == nil || err.Error() != tc.expectedError.Error() {
					t.Errorf("Expected error %v, got %v", tc.expectedError, err)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)

				return
			}

			if coordinates == nil {
				t.Errorf("Expected coordinates, got nil")

				return
			}

			if coordinates.Host != tc.expectedCoordinates.Host {
				t.Errorf("Expected host %s, got %s", tc.expectedCoordinates.Host, coordinates.Host)
			}
			if coordinates.Org != tc.expectedCoordinates.Org {
				t.Errorf("Expected org %s, got %s", tc.expectedCoordinates.Org, coordinates.Org)
			}
			if coordinates.Repo != tc.expectedCoordinates.Repo {
				t.Errorf("Expected repo %s, got %s", tc.expectedCoordinates.Repo, coordinates.Repo)
			}
			if coordinates.Version != tc.expectedCoordinates.Version {
				t.Errorf("Expected version %s, got %s", tc.expectedCoordinates.Version, coordinates.Version)
			}
			if coordinates.Artifact != tc.expectedCoordinates.Artifact {
				t.Errorf("Expected artifact %s, got %s", tc.expectedCoordinates.Artifact, coordinates.Artifact)
			}
			if coordinates.RawPath != tc.expectedCoordinates.RawPath {
				t.Errorf("Expected raw path %s, got %s", tc.expectedCoordinates.RawPath, coordinates.RawPath)
			}
		})
	}
}
