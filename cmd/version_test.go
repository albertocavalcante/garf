package cmd_test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/albertocavalcante/garf/cmd"
	"github.com/stretchr/testify/require"
)

// Global mutex to protect access to cmd.Version, cmd.CommitHash, and cmd.BuildDate.
var versionMutex sync.Mutex

type versionTestCase struct {
	name           string
	version        string
	commitHash     string
	buildDate      string
	expectedOutput string
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

	// Create a channel to ensure tests run sequentially while allowing Go test parallelism
	testChannel := make(chan struct{}, 1)
	testChannel <- struct{}{} // Initialize with one token

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Get token from channel to ensure only one test runs at a time
			token := <-testChannel
			defer func() { testChannel <- token }()

			// Save original values
			versionMutex.Lock()
			origVersion := cmd.Version
			origCommitHash := cmd.CommitHash
			origBuildDate := cmd.BuildDate

			// Set test values
			cmd.Version = tc.version
			cmd.CommitHash = tc.commitHash
			cmd.BuildDate = tc.buildDate
			versionMutex.Unlock()

			// Create a buffer to capture output
			var buf bytes.Buffer

			// Use the actual NewVersionCmd
			command := cmd.NewVersionCmd()
			command.SetOut(&buf)

			// Execute the command
			err := command.Execute()
			require.NoError(t, err)

			// Check output
			require.Equal(t, tc.expectedOutput, buf.String())

			// Restore original values
			versionMutex.Lock()
			cmd.Version = origVersion
			cmd.CommitHash = origCommitHash
			cmd.BuildDate = origBuildDate
			versionMutex.Unlock()
		})
	}
}
