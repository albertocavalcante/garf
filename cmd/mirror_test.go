package cmd_test

import (
	"os"
	"testing"

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
		require.Equal(t, tc.envVars["JFROG_URL"], config.URL)
		require.Equal(t, tc.envVars["JFROG_USER"], config.User)
		require.Equal(t, tc.envVars["JFROG_PASSWORD"], config.Password)
	}
}

// TestParsePropertiesBasic tests basic property parsing functionality.
func TestParsePropertiesBasic(t *testing.T) {
	tests := []struct {
		name     string
		props    []string
		expected map[string]string
	}{
		{
			name:     "Empty properties",
			props:    []string{},
			expected: map[string]string{},
		},
		{
			name: "Single property",
			props: []string{
				"type=toolchain",
			},
			expected: map[string]string{
				"type": "toolchain",
			},
		},
		{
			name: "Multiple properties",
			props: []string{
				"type=toolchain",
				"platform=windows",
				"version=1.0.0",
			},
			expected: map[string]string{
				"type":     "toolchain",
				"platform": "windows",
				"version":  "1.0.0",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cmd.ParseProperties(tc.props)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestParsePropertiesInvalid tests invalid property parsing.
func TestParsePropertiesInvalid(t *testing.T) {
	tests := []struct {
		name     string
		props    []string
		expected map[string]string
	}{
		{
			name: "Invalid property format",
			props: []string{
				"type=toolchain",
				"invalid-property",
				"platform=windows",
			},
			expected: map[string]string{
				"type":     "toolchain",
				"platform": "windows",
			},
		},
		{
			name: "Empty property value",
			props: []string{
				"type=",
			},
			expected: map[string]string{
				"type": "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cmd.ParseProperties(tc.props)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestParsePropertiesSpecial tests property parsing with special cases.
func TestParsePropertiesSpecial(t *testing.T) {
	tests := []struct {
		name     string
		props    []string
		expected map[string]string
	}{
		{
			name: "Property with equals in value",
			props: []string{
				"type=toolchain",
				"path=dir1=dir2=dir3",
			},
			expected: map[string]string{
				"type": "toolchain",
				"path": "dir1=dir2=dir3",
			},
		},
		{
			name: "Multiple equals signs",
			props: []string{
				"path=dir1=dir2=dir3",
			},
			expected: map[string]string{
				"path": "dir1=dir2=dir3",
			},
		},
		{
			name: "Whitespace in key and value",
			props: []string{
				" type = toolchain ",
			},
			expected: map[string]string{
				"type": "toolchain",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cmd.ParseProperties(tc.props)
			require.Equal(t, tc.expected, result)
		})
	}
}

// Helper functions for environment management.
func saveEnvironment(keys []string) map[string]string {
	env := make(map[string]string)
	for _, key := range keys {
		env[key] = os.Getenv(key)
	}

	return env
}

func restoreEnvironment(env map[string]string) {
	for key, value := range env {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}
