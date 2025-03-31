package cmd_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/albertocavalcante/garf/cmd"
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
			expectedOutput: fmt.Sprintf(`Version: dev
Commit: abc123
Build Date: 2024-03-28
`),
		},
		{
			name:       "release version",
			version:    "v1.0.0",
			commitHash: "def456",
			buildDate:  "2024-03-29",
			expectedOutput: fmt.Sprintf(`Version: v1.0.0
Commit: def456
Build Date: 2024-03-29
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origVersion := cmd.Version
			origCommitHash := cmd.CommitHash
			origBuildDate := cmd.BuildDate

			// Set test values
			cmd.Version = tt.version
			cmd.CommitHash = tt.commitHash
			cmd.BuildDate = tt.buildDate

			// Restore original values after test
			defer func() {
				cmd.Version = origVersion
				cmd.CommitHash = origCommitHash
				cmd.BuildDate = origBuildDate
			}()

			// Create a buffer to capture output
			var buf bytes.Buffer

			cmd := cmd.NewVersionCmd()
			cmd.SetOut(&buf)

			// Execute the command
			err := cmd.Execute()
			require.NoError(t, err)

			// Check output
			require.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}
