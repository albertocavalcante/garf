package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectSourceType(t *testing.T) {
	tests := getSourceTypeDetectionTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSourceTypeDetectionTest(t, tt)
		})
	}
}

func getSourceTypeDetectionTestCases() []struct {
	name               string
	sourceURL          string
	sourcePathStrip    string
	expectedSourceType string
} {
	return []struct {
		name               string
		sourceURL          string
		sourcePathStrip    string
		expectedSourceType string
	}{
		{
			name:               "GitHub URL",
			sourceURL:          "https://github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip:    "",
			expectedSourceType: SourceTypeGitHub,
		},
		{
			name:               "Generic URL",
			sourceURL:          "https://example.com/file.zip",
			sourcePathStrip:    "",
			expectedSourceType: SourceTypeGeneric,
		},
		{
			name:               "JFrog URL without strip",
			sourceURL:          "https://artifactory.corp.net/staging/file.zip",
			sourcePathStrip:    "",
			expectedSourceType: SourceTypeGeneric,
		},
		{
			name:               "JFrog URL with strip revealing GitHub",
			sourceURL:          "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip:    "artifactory.corp.net/staging/",
			expectedSourceType: SourceTypeGitHub,
		},
		{
			name:               "JFrog URL with strip revealing non-GitHub",
			sourceURL:          "https://artifactory.corp.net/staging/some-other-host.com/file.zip",
			sourcePathStrip:    "artifactory.corp.net/staging/",
			expectedSourceType: SourceTypeGeneric,
		},
		{
			name:               "URL with strip prefix not found",
			sourceURL:          "https://github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip:    "not-found-prefix/",
			expectedSourceType: SourceTypeGitHub,
		},
		{
			name:               "Empty source URL",
			sourceURL:          "",
			sourcePathStrip:    "",
			expectedSourceType: SourceTypeGeneric,
		},
		{
			name:               "Strip with scheme reconstruction",
			sourceURL:          "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/file.zip",
			sourcePathStrip:    "artifactory.corp.net/staging/",
			expectedSourceType: SourceTypeGitHub,
		},
	}
}

func runSourceTypeDetectionTest(t *testing.T, tt struct {
	name               string
	sourceURL          string
	sourcePathStrip    string
	expectedSourceType string
},
) {
	result := DetectSourceType(tt.sourceURL, tt.sourcePathStrip)
	require.Equal(t, tt.expectedSourceType, result)
}
