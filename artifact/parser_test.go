package artifact_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/albertocavalcante/garf/artifact"
	"github.com/stretchr/testify/require"
)

// verifyCoordinates compares the expected and actual coordinates.
func verifyCoordinates(t *testing.T, expected, actual *artifact.ArtifactCoordinates) {
	if actual == nil {
		t.Errorf("Expected coordinates, got nil")

		return
	}

	require.Equal(t, expected.Host, actual.Host, "Host should match")
	require.Equal(t, expected.Org, actual.Org, "Org should match")
	require.Equal(t, expected.Repo, actual.Repo, "Repo should match")
	require.Equal(t, expected.Version, actual.Version, "Version should match")
	require.Equal(t, expected.Artifact, actual.Artifact, "Artifact should match")
	require.Equal(t, expected.RawPath, actual.RawPath, "RawPath should match")
}

func TestExtractCoordinatesFromURL(t *testing.T) {
	type testCase struct {
		artifactURL         string
		expectedCoordinates *artifact.ArtifactCoordinates
		expectedError       error
	}

	testCases := []testCase{
		{
			artifactURL: "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe",
			expectedCoordinates: &artifact.ArtifactCoordinates{
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
			expectedCoordinates: &artifact.ArtifactCoordinates{
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
			coordinates, err := artifact.ExtractCoordinatesFromURL(tc.artifactURL)

			// Check error cases first
			if tc.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedError.Error(), err.Error())

				return
			}

			// Check success cases
			require.NoError(t, err)
			verifyCoordinates(t, tc.expectedCoordinates, coordinates)
		})
	}
}
