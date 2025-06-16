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

	tests := getVersionTestCases()
	testChannel := make(chan struct{}, 1)
	testChannel <- struct{}{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runVersionTest(t, tc, testChannel)
		})
	}
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

func runVersionTest(t *testing.T, tc versionTestCase, testChannel chan struct{}) {
	t.Parallel()

	token := <-testChannel
	defer func() { testChannel <- token }()

	origVersion, origCommitHash, origBuildDate := saveOriginalVersionValues()
	defer restoreOriginalVersionValues(origVersion, origCommitHash, origBuildDate)

	setTestVersionValues(tc)

	var buf bytes.Buffer

	command := cmd.NewVersionCmd()
	command.SetOut(&buf)

	err := command.Execute()
	require.NoError(t, err)
	require.Equal(t, tc.expectedOutput, buf.String())
}

func saveOriginalVersionValues() (string, string, string) {
	versionMutex.Lock()
	defer versionMutex.Unlock()

	return cmd.Version, cmd.CommitHash, cmd.BuildDate
}

func restoreOriginalVersionValues(version, commitHash, buildDate string) {
	versionMutex.Lock()
	defer versionMutex.Unlock()

	cmd.Version, cmd.CommitHash, cmd.BuildDate = version, commitHash, buildDate
}

func setTestVersionValues(tc versionTestCase) {
	versionMutex.Lock()
	defer versionMutex.Unlock()

	cmd.Version = tc.version
	cmd.CommitHash = tc.commitHash
	cmd.BuildDate = tc.buildDate
}
