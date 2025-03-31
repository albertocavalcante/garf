package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the current version of the CLI.
var Version = "dev"

// CommitHash is the git commit hash.
var CommitHash = "unknown"

// BuildDate is the date the binary was built.
var BuildDate = "unknown"

// NewVersionCmd returns a new Cobra command configured to display the CLI's version information.
// When executed, the command prints the version number, git commit hash, and build date to the standard output.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long: `Print the version information of garf CLI including:
				- Version number
				- Git commit hash
				- Build date`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Version: %s\n", Version)
			fmt.Fprintf(out, "Commit: %s\n", CommitHash)
			fmt.Fprintf(out, "Build Date: %s\n", BuildDate)

			return nil
		},
	}

	return cmd
}
