package cmd_test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/albertocavalcante/garf/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var versionMutex sync.Mutex

type versionTestCase struct {
	name           string
	version        string
	commitHash     string
	buildDate      string
	expectedOutput string
}

func getVersionTestCases() []versionTestCase {
	return []versionTestCase{
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
}

func setupVersionTest(t *testing.T, tc versionTestCase) (*cobra.Command, *bytes.Buffer, func()) {
	t.Helper()

	// Lock mutex before modifying global variables
	versionMutex.Lock()

	// Save original values
	origVersion := cmd.Version
	origCommitHash := cmd.CommitHash
	origBuildDate := cmd.BuildDate

	// Set test values
	cmd.Version = tc.version
	cmd.CommitHash = tc.commitHash
	cmd.BuildDate = tc.buildDate

	// Create a buffer to capture output
	var buf bytes.Buffer

	command := cmd.NewVersionCmd()
	command.SetOut(&buf)

	cleanup := func() {
		versionMutex.Lock()
		cmd.Version = origVersion
		cmd.CommitHash = origCommitHash
		cmd.BuildDate = origBuildDate
		versionMutex.Unlock()
	}

	// Unlock mutex after setting values
	versionMutex.Unlock()

	return command, &buf, cleanup
}

func TestVersionCmd(t *testing.T) {
	t.Parallel()

	tests := getVersionTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			command, buf, cleanup := setupVersionTest(t, tt)
			t.Cleanup(cleanup)

			// Execute the command
			err := command.Execute()
			require.NoError(t, err)

			// Check output
			require.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}
