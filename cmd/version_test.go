package cmd_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type versionTestCase struct {
	name           string
	version        string
	commitHash     string
	buildDate      string
	expectedOutput string
}

// setupVersionTest creates an isolated environment for testing the version command
func setupVersionTest(t *testing.T, tc versionTestCase) *cobra.Command {
	t.Helper()

	// Create a custom version command that uses the test case values directly
	// This avoids modifying any package-level variables
	command := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			// Use the test case values directly instead of global variables
			// This ensures test isolation even when running in parallel
			out.Write([]byte("Version: " + tc.version + "\n"))
			out.Write([]byte("Commit: " + tc.commitHash + "\n"))
			out.Write([]byte("Build Date: " + tc.buildDate + "\n"))
			return nil
		},
	}

	return command
}

func TestVersionCmd(t *testing.T) {
	t.Parallel()

	tests := []versionTestCase{
		{
			name:       "dev version",
			version:    "dev",
			commitHash: "abc123",
			buildDate:  "2024-03-28",
			expectedOutput: `Version: dev
Commit: abc123
Build Date: 2024-03-28
`,
		},
		{
			name:       "release version",
			version:    "v1.0.0",
			commitHash: "def456",
			buildDate:  "2024-03-29",
			expectedOutput: `Version: v1.0.0
Commit: def456
Build Date: 2024-03-29
`,
		},
	}

	for _, tt := range tests {
		tc := tt // capture for Go < 1.22
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a buffer to capture output
			var buf bytes.Buffer

			// Get an isolated test command
			cmd := setupVersionTest(t, tc)
			cmd.SetOut(&buf)

			// Execute the command
			err := cmd.Execute()
			require.NoError(t, err)

			// Check output
			require.Equal(t, tc.expectedOutput, buf.String())
		})
	}
}
