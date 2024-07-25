package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	UnsuccessExitCode = 1
)

var rootCmd = &cobra.Command{
	Use:   "alb",
	Short: "alb is a CLI for myself :)",
	Long: `A CLI project template to play with
			Go, Cobra, Bazel and Bzlmod`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("alb")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(UnsuccessExitCode)
	}
}
