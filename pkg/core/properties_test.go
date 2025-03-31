package core_test

import (
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/stretchr/testify/require"
)

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
			result := core.ParseProperties(tc.props)
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
			result := core.ParseProperties(tc.props)
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
			result := core.ParseProperties(tc.props)
			require.Equal(t, tc.expected, result)
		})
	}
}
