package core_test

import (
	"testing"

	"github.com/albertocavalcante/garf/core"
	"github.com/stretchr/testify/require"
)

func TestParseProperty(t *testing.T) {
	tests := []struct {
		name          string
		propertyStr   string
		expectedKey   string
		expectedValue string
		shouldErr     bool
	}{
		{
			name:          "Valid property",
			propertyStr:   "key=value",
			expectedKey:   "key",
			expectedValue: "value",
			shouldErr:     false,
		},
		{
			name:          "Property with equals sign in value",
			propertyStr:   "key=value=with=equals",
			expectedKey:   "key",
			expectedValue: "value=with=equals",
			shouldErr:     false,
		},
		{
			name:          "Empty value",
			propertyStr:   "key=",
			expectedKey:   "key",
			expectedValue: "",
			shouldErr:     false,
		},
		{
			name:        "No equals sign",
			propertyStr: "keyvalue",
			shouldErr:   true,
		},
		{
			name:        "Empty string",
			propertyStr: "",
			shouldErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, value, err := core.ParseProperty(tc.propertyStr)

			if tc.shouldErr {
				require.Error(t, err, "Expected error for invalid input")
			} else {
				require.NoError(t, err, "No error expected for valid input")
				require.Equal(t, tc.expectedKey, key, "Keys should match")
				require.Equal(t, tc.expectedValue, value, "Values should match")
			}
		})
	}
}

func TestCreateTargetProperties(t *testing.T) {
	tests := []struct {
		name           string
		properties     []string
		expectedResult map[string][]string
		shouldErr      bool
	}{
		{
			name:           "Empty properties list",
			properties:     []string{},
			expectedResult: map[string][]string{},
			shouldErr:      false,
		},
		{
			name:       "Single property",
			properties: []string{"key=value"},
			expectedResult: map[string][]string{
				"key": {"value"},
			},
			shouldErr: false,
		},
		{
			name:       "Multiple properties",
			properties: []string{"key1=value1", "key2=value2", "key3=value3"},
			expectedResult: map[string][]string{
				"key1": {"value1"},
				"key2": {"value2"},
				"key3": {"value3"},
			},
			shouldErr: false,
		},
		{
			name:       "Duplicate keys",
			properties: []string{"key1=value1", "key1=value2"},
			expectedResult: map[string][]string{
				"key1": {"value1", "value2"},
			},
			shouldErr: false,
		},
		{
			name:       "Invalid property",
			properties: []string{"key1=value1", "invalid-property"},
			shouldErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			props, err := core.CreateTargetProperties(tc.properties)

			if tc.shouldErr {
				require.Error(t, err, "Expected error for invalid input")
			} else {
				require.NoError(t, err, "No error expected for valid input")
				require.NotNil(t, props, "Properties should not be nil")

				// Verify all properties were correctly added
				propsMap := props.ToMap()
				require.Equal(t, tc.expectedResult, propsMap, "Property maps should match")
			}
		})
	}
}

// Note: We're not testing UploadGenericArtifact directly because it would require
// implementing the full ArtifactoryServicesManager interface with many methods.
// Instead, we're testing the property handling functions separately, which is where
// the main logic happens.
//
// In a real-world scenario, you might use a more advanced mocking framework or
// dependency injection to test UploadGenericArtifact fully.
