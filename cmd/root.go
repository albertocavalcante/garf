package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	UnsuccessExitCode = 1
)

// NewRootCmd creates the root command for the CLI.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "garf",
		Short: "garf is a CLI to mirror artifacts",
		Long: `A CLI to mirror artifacts from places such as
				GitHub Releases to registries such as 
				JFrog Artifactory.`,
	}

	cmd.AddCommand(NewMirrorCmd())

	return cmd
}

// Execute runs the root command and exits the process with a non-zero status
// code if an error occurs.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(UnsuccessExitCode)
	}
}
