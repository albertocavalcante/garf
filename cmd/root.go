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

type GarfFlags struct {
	Source      string
	Destination string
	FromFile    string
}

func (f *GarfFlags) addFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVarP(&f.Source, "source", "s", "", "GitHub Release URL to the artifact")
	flags.StringVarP(&f.Destination, "destination", "d", "", "Artifacts destination (e.g. sandbox-generic-local)")
	flags.StringVarP(&f.FromFile, "from-file", "f", "", "Skip Download. Upload from file and use URL to infer coordinates")
}

// NewRootCmd creates the root command for the CLI.
func NewRootCmd() *cobra.Command {
	f := &GarfFlags{}

	var jfConfig *core.JFrogConfig

	cmd := &cobra.Command{
		Use:   "garf",
		Short: "garf is a CLI to mirror artifacts",
		Long: `A CLI to mirror artifacts from places such as
				GitHub Releases to registries such as 
				JFrog Artifactory.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if f.Source == "" || f.Destination == "" {
				return fmt.Errorf("--source and --destination flags are required")
			}

			// Get required environment variables
			jfrogUrl := os.Getenv("JFROG_URL")
			jfrogUser := os.Getenv("JFROG_USER")
			jfrogPassword := os.Getenv("JFROG_PASSWORD")

			// Validate required environment variables
			if jfrogUrl == "" || jfrogUser == "" || jfrogPassword == "" {
				return fmt.Errorf("JFROG_URL, JFROG_USER and JFROG_PASSWORD environment variables are required")
			}

			jfConfig = &core.JFrogConfig{
				Url:      jfrogUrl,
				User:     jfrogUser,
				Password: jfrogPassword,
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactCoordinates, err := artifact.ExtractCoordinatesFromURL(f.Source)
			if err != nil {
				return err
			}
			fmt.Printf("Artifact coordinates: %+v\n", artifactCoordinates)

			fmt.Printf("Mirroring %s to %s\n", f.Source, f.Destination)

			var location string
			if f.FromFile != "" {
				location = f.FromFile
			} else {
				defer os.RemoveAll(filepath.Dir(location))

				location, err = core.DownloadArtifact(f.Source)
				if err != nil {
					return err
				}
				fmt.Printf("Artifact downloaded to %s\n", location)
			}

			jfClient, err := core.NewJFrogClient(jfConfig)
			if err != nil {
				return err
			}

			err = jfClient.UploadGenericArtifact(location, f.Destination, artifactCoordinates)
			if err != nil {
				return err
			}

			return nil
		},
	}

	f.addFlags(cmd)

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(UnsuccessExitCode)
	}
}
