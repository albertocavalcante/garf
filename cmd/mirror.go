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
	Unzip       bool
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
	flags.BoolVar(
		&f.Unzip,
		"unzip",
		false,
		"Unzip and upload content if source is a zip file with a single file inside",
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

// processFromLocalFile handles the case when the user specified a local file with --from-file.
func processFromLocalFile(flags *MirrorFlags, location string) (string, string, bool, error) {
	// Validate the file exists
	if _, err := os.Stat(location); os.IsNotExist(err) {
		return "", "", false, fmt.Errorf("file '%s' not found", location)
	}

	// Skip creating temp directory if unzipping not needed
	if !flags.Unzip {
		return location, "", false, nil
	}

	// Skip creating temp directory if not a zip file
	if !core.IsZipFile(location) {
		return location, "", false, nil
	}

	// Create temp directory for zip extraction
	tempDir, err := os.MkdirTemp("", "garf-unzip-")
	if err != nil {
		return location, "", false, fmt.Errorf("failed to create temporary directory for unzipping: %w", err)
	}

	return location, tempDir, true, nil
}

// processFromRemoteURL handles downloading an artifact from a remote URL.
func processFromRemoteURL(flags *MirrorFlags) (string, string, bool, error) {
	location, err := core.DownloadArtifact(flags.Source)
	if err != nil {
		return "", "", false, fmt.Errorf("failed to download artifact: %w", err)
	}

	tempDir := filepath.Dir(location)
	if tempDir == "" {
		return location, "", false, fmt.Errorf("unable to determine temporary directory for downloaded file")
	}

	return location, tempDir, true, nil
}

// handleZipExtraction handles the extraction of a zip file if the --unzip flag is specified.
func handleZipExtraction(
	flags *MirrorFlags,
	location string,
	tempDir string,
	coordinates *artifact.ArtifactCoordinates,
) (string, error) {
	// Skip extraction if unzip flag is not set
	if !flags.Unzip {
		return location, nil
	}

	// Skip extraction if file is not a zip
	if !core.IsZipFile(location) {
		return location, nil
	}

	// Configure extraction options
	extractOptions := &core.ExtractOptions{}

	// Set extraction destination if temp directory is available
	if tempDir != "" {
		extractOptions.DestinationDir = tempDir
	}

	extractedPath, extracted, err := core.ExtractSingleFileFromZip(location, extractOptions)
	if err != nil {
		return location, fmt.Errorf("failed to unzip file: %w", err)
	}

	// No extraction happened
	if !extracted {
		return location, nil
	}

	// Update the artifact name in coordinates to match the extracted file
	originalName := coordinates.Artifact
	coordinates.Artifact = filepath.Base(extractedPath)

	fmt.Printf("Extracted %s from %s\n", coordinates.Artifact, originalName)

	return extractedPath, nil
}

// processAndUploadArtifact handles the downloading and uploading of an artifact.
func processAndUploadArtifact(flags *MirrorFlags, jfrogConfig *core.JFrogConfig) error {
	coordinates, err := artifact.ExtractCoordinatesFromURL(flags.Source)
	if err != nil {
		return err
	}

	// Process the artifact from local file or remote URL
	var location string

	var tempDir string

	var needsCleanup bool

	if flags.FromFile != "" {
		location, tempDir, needsCleanup, err = processFromLocalFile(flags, flags.FromFile)
	} else {
		location, tempDir, needsCleanup, err = processFromRemoteURL(flags)
	}

	if err != nil {
		return err
	}

	// Set up cleanup of temporary directories
	if needsCleanup && tempDir != "" {
		defer func() {
			os.RemoveAll(tempDir)
		}()
	}

	// Handle unzipping if needed
	location, err = handleZipExtraction(flags, location, tempDir, coordinates)
	if err != nil {
		return err
	}

	// Upload the artifact
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
	// Use raw path if requested
	if raw {
		return fmt.Sprintf("%s/%s", repoKey, coordinates.RawUrlPath())
	}

	// Otherwise use parsed path
	return fmt.Sprintf("%s/%s", repoKey, coordinates.UrlPath())
}

// HandleZipExtractionForTest is an exported version of handleZipExtraction for testing.
// It's a simple pass-through to the private function to enable unit testing.
func HandleZipExtractionForTest(
	flags *MirrorFlags,
	location string,
	tempDir string,
	coordinates *artifact.ArtifactCoordinates,
) (string, error) {
	return handleZipExtraction(flags, location, tempDir, coordinates)
}

type FileNotFoundError struct {
	Path string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("file not found: %s", e.Path)
}
