package cmd

import (
	"fmt"
	"os"

	"github.com/albertocavalcante/garf/artifact"
	"github.com/spf13/cobra"
)

const (
	UnsuccessExitCode = 1
)

var (
	source      string
	destination string

	rootCmd = &cobra.Command{
		Use:   "garf",
		Short: "garf is a CLI to mirror artifacts",
		Long: `A CLI to mirror artifacts from places such as
				GitHub Releases to registries such as 
				JFrog Artifactory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" || destination == "" {
				return fmt.Errorf("--source and --destination flags are required")
			}

			artifactCoordinates, err := artifact.ExtractCoordinatesFromURL(source)
			if err != nil {
				return err
			}
			fmt.Printf("Mirroring %s to %s\n", source, destination)
			fmt.Printf("Artifact coordinates: %+v\n", artifactCoordinates)
			return nil
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&source, "source", "s", "", "Source of the artifacts")
	rootCmd.Flags().StringVarP(&destination, "destination", "d", "", "Destination of the artifacts")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(UnsuccessExitCode)
	}
}
