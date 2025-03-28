package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		commitHash     string
		buildDate      string
		expectedOutput string
	}{
		{
			name:       "dev version",
			version:    "dev",
			commitHash: "abc123",
			buildDate:  "2024-03-28",
			expectedOutput: fmt.Sprintf(`garf version dev
commit: abc123
build date: 2024-03-28
`),
		},
		{
			name:       "release version",
			version:    "v1.0.0",
			commitHash: "def456",
			buildDate:  "2024-03-29",
			expectedOutput: fmt.Sprintf(`garf version v1.0.0
commit: def456
build date: 2024-03-29
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origVersion := Version
			origCommitHash := CommitHash
			origBuildDate := BuildDate

			// Set test values
			Version = tt.version
			CommitHash = tt.commitHash
			BuildDate = tt.buildDate

			// Restore original values after test
			defer func() {
				Version = origVersion
				CommitHash = origCommitHash
				BuildDate = origBuildDate
			}()

			// Create a buffer to capture output
			var buf bytes.Buffer
			cmd := NewVersionCmd()
			cmd.SetOut(&buf)

			// Execute the command
			err := cmd.Execute()
			require.NoError(t, err)

			// Check output
			require.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}
