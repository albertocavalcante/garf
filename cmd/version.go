package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is the current version of the CLI
	Version = "dev"
	// CommitHash is the git commit hash
	CommitHash = "unknown"
	// BuildDate is the date the binary was built
	BuildDate = "unknown"
)

// NewVersionCmd creates the version command for the CLI.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long: `Print the version information of garf CLI including:
				- Version number
				- Git commit hash
				- Build date`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("garf version %s\n", Version)
			fmt.Printf("commit: %s\n", CommitHash)
			fmt.Printf("build date: %s\n", BuildDate)
		},
	}

	return cmd
}
