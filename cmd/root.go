package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/albertocavalcante/garf/artifact"
	"github.com/albertocavalcante/garf/core"
	"github.com/spf13/cobra"
)

const (
	UnsuccessExitCode = 1
)

var (
	source      string
	destination string
	fromFile    string

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
			fmt.Printf("Artifact coordinates: %+v\n", artifactCoordinates)

			fmt.Printf("Mirroring %s to %s\n", source, destination)

			var location string
			if fromFile != "" {
				location = fromFile
			} else {
				defer os.RemoveAll(filepath.Dir(location))

				location, err = core.DownloadArtifact(source)
				if err != nil {
					return err
				}
				fmt.Printf("Artifact downloaded to %s\n", location)
			}

			err = core.UploadGenericArtifact(location, destination, artifactCoordinates)
			if err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&source, "source", "s", "", "Source of the artifacts")
	rootCmd.Flags().StringVarP(&destination, "destination", "d", "", "Destination of the artifacts")
	rootCmd.Flags().StringVarP(&fromFile, "from-file", "f", "", "Use the source URL only for coordinates and upload the file at the specified path")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(UnsuccessExitCode)
	}
}
