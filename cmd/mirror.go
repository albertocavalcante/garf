package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/albertocavalcante/garf/artifact"
	"github.com/albertocavalcante/garf/core"
	"github.com/spf13/cobra"
)

type MirrorFlags struct {
	Source      string
	Destination string
	FromFile    string
	Raw         bool
	Properties  []string
}

func (f *MirrorFlags) addFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVarP(&f.Source, "source", "s", "", "GitHub Release URL to the artifact")
	flags.StringVarP(&f.Destination, "destination", "d", "", "Artifacts destination (e.g. sandbox-generic-local)")
	flags.StringVarP(&f.FromFile, "from-file", "f", "", "Skip Download. Upload from file and use URL to infer coordinates")
	flags.BoolVar(
		&f.Raw,
		"raw",
		false,
		"Raw Mirror. Don't upload the artifact with the parsed coordinates but the full URL path",
	)
	flags.StringArrayVar(
		&f.Properties,
		"properties",
		[]string{},
		"Properties to attach to the artifact (e.g. type=toolchain platform=windows)",
	)
}

// ValidateAndGetConfig validates required flags and environment variables and returns a JFrog config.
func ValidateAndGetConfig(source, destination string) (*core.JFrogConfig, error) {
	if source == "" || destination == "" {
		return nil, fmt.Errorf("--source and --destination flags are required")
	}

	jfrogUrl, ok := os.LookupEnv("JFROG_URL")
	if !ok {
		return nil, fmt.Errorf("JFROG_URL environment variable is required")
	}

	jfrogUser, ok := os.LookupEnv("JFROG_USER")
	if !ok {
		return nil, fmt.Errorf("JFROG_USER environment variable is required")
	}

	jfrogPassword, ok := os.LookupEnv("JFROG_PASSWORD")
	if !ok {
		return nil, fmt.Errorf("JFROG_PASSWORD environment variable is required")
	}

	return &core.JFrogConfig{
		Url:      jfrogUrl,
		User:     jfrogUser,
		Password: jfrogPassword,
	}, nil
}

// processAndUploadArtifact handles the downloading and uploading of an artifact.
func processAndUploadArtifact(flags *MirrorFlags, jfrogConfig *core.JFrogConfig) error {
	coordinates, err := artifact.ExtractCoordinatesFromURL(flags.Source)
	if err != nil {
		return err
	}

	var location string

	var tempDir string

	if flags.FromFile != "" {
		location = flags.FromFile
	} else {
		location, err = core.DownloadArtifact(flags.Source)
		if err != nil {
			return err
		}

		tempDir = filepath.Dir(location)
		defer os.RemoveAll(tempDir)
	}

	jfrogClient, err := core.NewJFrogClient(jfrogConfig)
	if err != nil {
		return err
	}

	targetPath := ConstructTargetPath(flags.Destination, coordinates, flags.Raw)

	return jfrogClient.UploadGenericArtifact(location, targetPath, flags.Properties)
}

// NewMirrorCmd creates a new cobra.Command for the "mirror" subcommand.

// This subcommand will download an artifact from a source URL and upload it
// to a destination URL. The source URL should be a GitHub Releases URL and
// the destination URL should be a JFrog Generic Repository URL.

// The "--from-file" flag can be used to skip the download step and upload a
// local file instead. The file path should be specified as the value for this
// flag.

// This subcommand requires the following environment variables to be set:

// - JFROG_URL: the URL of the JFrog Artifactory instance
// - JFROG_USER: the username to use for authentication
// - JFROG_PASSWORD: the password to use for authentication

// This subcommand will also require the "--source" and "--destination" flags
// to be set.
func NewMirrorCmd() *cobra.Command {
	flags := &MirrorFlags{}

	var jfrogConfig *core.JFrogConfig

	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Mirror artifacts from places such as GitHub Releases to registries such as JFrog Artifactory.",
		Long: `Mirror will simply download artifacts from a source URL and upload them to a destination URL, 
		preserving their path.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := ValidateAndGetConfig(flags.Source, flags.Destination)
			if err != nil {
				return err
			}

			jfrogConfig = config

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return processAndUploadArtifact(flags, jfrogConfig)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

// ConstructTargetPath constructs the target path for uploading the artifact.
func ConstructTargetPath(repoKey string, coordinates *artifact.ArtifactCoordinates, raw bool) string {
	mirrorPath := coordinates.UrlPath()
	if raw {
		mirrorPath = coordinates.RawUrlPath()
	}

	return fmt.Sprintf("%s/%s", repoKey, mirrorPath)
}
