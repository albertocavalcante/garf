package cmd_test

import (
	"os"
	"testing"

	"github.com/albertocavalcante/garf/artifact"
	"github.com/albertocavalcante/garf/cmd"
	"github.com/stretchr/testify/require"
)

// configTestCase defines a test case for ValidateAndGetConfig testing.
type configTestCase struct {
	name        string
	source      string
	destination string
	envVars     map[string]string
	shouldErr   bool
}

// validateConfigTestCases contains all test cases for ValidateAndGetConfig.
var validateConfigTestCases = []configTestCase{
	{
		name:        "Valid config with all required params",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: false,
	},
	{
		name:        "Missing source",
		source:      "",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing destination",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing JFROG_URL",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_USER":     "user",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing JFROG_USER",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":      "https://example.jfrog.io/artifactory",
			"JFROG_PASSWORD": "password",
		},
		shouldErr: true,
	},
	{
		name:        "Missing JFROG_PASSWORD",
		source:      "https://github.com/example/repo/releases/download/v1.0/file.zip",
		destination: "repo-local",
		envVars: map[string]string{
			"JFROG_URL":  "https://example.jfrog.io/artifactory",
			"JFROG_USER": "user",
		},
		shouldErr: true,
	},
}

// TestValidateAndGetConfig tests the ValidateAndGetConfig function with various
// combinations of valid and invalid inputs.
func TestValidateAndGetConfig(t *testing.T) {
	// Save original environment
	origEnv := saveEnvironment([]string{"JFROG_URL", "JFROG_USER", "JFROG_PASSWORD"})

	// Restore environment after tests
	defer restoreEnvironment(origEnv)

	for _, tc := range validateConfigTestCases {
		t.Run(tc.name, func(t *testing.T) {
			runValidateConfigTest(t, tc)
		})
	}
}

// runValidateConfigTest runs a single test case for ValidateAndGetConfig.
func runValidateConfigTest(t *testing.T, tc configTestCase) {
	// Clear environment
	os.Unsetenv("JFROG_URL")
	os.Unsetenv("JFROG_USER")
	os.Unsetenv("JFROG_PASSWORD")

	// Set test environment variables
	for k, v := range tc.envVars {
		os.Setenv(k, v)
	}

	// Run the function
	config, err := cmd.ValidateAndGetConfig(tc.source, tc.destination)

	// Verify results
	if tc.shouldErr {
		require.Error(t, err, "Expected error for invalid input")
		require.Nil(t, config, "Config should be nil when error occurs")
	} else {
		require.NoError(t, err, "No error expected for valid input")
		require.NotNil(t, config, "Config should not be nil")
		require.Equal(t, tc.envVars["JFROG_URL"], config.Url)
		require.Equal(t, tc.envVars["JFROG_USER"], config.User)
		require.Equal(t, tc.envVars["JFROG_PASSWORD"], config.Password)
	}
}

// TestConstructTargetPath tests the ConstructTargetPath function with different
// coordinate types and raw options.
func TestConstructTargetPath(t *testing.T) {
	tests := []struct {
		name        string
		repoKey     string
		coordinates *artifact.ArtifactCoordinates
		raw         bool
		expected    string
	}{
		{
			name:    "Regular parsed coordinates",
			repoKey: "repo-local",
			coordinates: &artifact.ArtifactCoordinates{
				Host:     "github.com",
				Org:      "org",
				Repo:     "repo",
				Version:  "v1.0",
				Artifact: "file.zip",
				RawPath:  "org/repo/releases/download/v1.0/file.zip",
			},
			raw:      false,
			expected: "repo-local/github.com/org/repo/v1.0/file.zip",
		},
		{
			name:    "Raw path coordinates",
			repoKey: "repo-local",
			coordinates: &artifact.ArtifactCoordinates{
				Host:     "github.com",
				Org:      "org",
				Repo:     "repo",
				Version:  "v1.0",
				Artifact: "file.zip",
				RawPath:  "org/repo/releases/download/v1.0/file.zip",
			},
			raw:      true,
			expected: "repo-local/github.com/org/repo/releases/download/v1.0/file.zip",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cmd.ConstructTargetPath(tc.repoKey, tc.coordinates, tc.raw)
			require.Equal(t, tc.expected, result)
		})
	}
}

// saveEnvironment preserves the current environment variables for restoration later.
func saveEnvironment(keys []string) map[string]string {
	env := make(map[string]string)

	for _, key := range keys {
		env[key] = os.Getenv(key)
	}

	return env
}

// restoreEnvironment sets environment variables back to their original values.
func restoreEnvironment(env map[string]string) {
	for k, v := range env {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
}

// Note: We're not testing processAndUploadArtifact or the full NewMirrorCmd directly
// because they have external dependencies that would require more complex mocking.
// In a more complete test suite, you would mock the core.DownloadArtifact and
// core.NewJFrogClient functions to isolate the test from external dependencies.
