package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/albertocavalcante/garf/pkg/archive"
	"github.com/albertocavalcante/garf/pkg/config"
	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/destinations"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/albertocavalcante/garf/pkg/sources"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	propertyKeyValueParts = 2
	defaultTimeout        = 30 * time.Minute
	defaultConcurrent     = 4
)

// MirrorFlags represents the flags for the mirror command.
type MirrorFlags struct {
	ConfigFile  string
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

// ParseProperties converts the properties array into a map.
func ParseProperties(props []string) map[string]string {
	result := make(map[string]string)

	for _, prop := range props {
		parts := strings.SplitN(prop, "=", propertyKeyValueParts)
		if len(parts) == propertyKeyValueParts {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	return result
}

// ValidateAndGetConfig validates required flags and environment variables and returns a JFrog config.
func ValidateAndGetConfig(source, destination string) (*destinations.JFrogConfig, error) {
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

	return &destinations.JFrogConfig{
		URL:      jfrogUrl,
		User:     jfrogUser,
		Password: jfrogPassword,
	}, nil
}

// ZipExtractionParams holds the parameters for zip extraction.
type ZipExtractionParams struct {
	ctx      context.Context
	logger   *logrus.Logger
	mirror   *mirror.DefaultMirror
	artifact *core.Artifact
	opts     *core.MirrorOptions
}

// handleZipExtraction handles the extraction of a zip file and mirrors the extracted content.
func handleZipExtraction(params ZipExtractionParams) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "garf-unzip-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	extractOpts := archive.ExtractOptions{
		DestinationDir:       tempDir,
		PreserveOriginalName: true,
	}

	extractedPath, err := archive.ExtractSingleFile(params.artifact.Location, extractOpts)
	if err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	// Update the artifact name and location for the extracted file
	params.artifact.Name = filepath.Base(extractedPath)
	params.artifact.Location = extractedPath

	// Mirror the extracted file
	extractResults := params.mirror.Mirror(params.ctx, []*core.Artifact{params.artifact}, params.opts)
	for extractResult := range extractResults {
		if extractResult.Error != nil {
			return extractResult.Error
		}
	}

	return nil
}

// createMirror creates a new mirror instance with the given logger.
func createMirror(logger *logrus.Logger) *mirror.DefaultMirror {
	return mirror.NewDefaultMirror(logger)
}

// setupSource creates and adds a GitHub source to the mirror.
func setupSource(mirror *mirror.DefaultMirror, logger *logrus.Logger) (*sources.GitHubSource, error) {
	source := sources.NewGitHubSource(logger)
	if err := mirror.AddSource("github", source); err != nil {
		return nil, fmt.Errorf("failed to add source: %w", err)
	}

	return source, nil
}

// setupDestination creates and adds a JFrog destination to the mirror.
func setupDestination(mirror *mirror.DefaultMirror, logger *logrus.Logger, config *destinations.JFrogConfig) error {
	dest := destinations.NewJFrogDestination(*config, logger)
	if err := mirror.AddDestination("jfrog", dest); err != nil {
		return fmt.Errorf("failed to add destination: %w", err)
	}

	return nil
}

// createArtifact creates a new artifact from the given flags.
func createArtifact(flags *MirrorFlags) *core.Artifact {
	artifact := &core.Artifact{
		Name:     filepath.Base(flags.Source),
		Location: flags.Source,
		Metadata: ParseProperties(flags.Properties),
	}

	if flags.FromFile != "" {
		artifact.Location = flags.FromFile
	}

	return artifact
}

// createMirrorOptions creates mirror options from the given flags.
func createMirrorOptions(ctx context.Context, flags *MirrorFlags) *core.MirrorOptions {
	return &core.MirrorOptions{
		PreserveStructure: !flags.Raw,
		Context:           ctx,
		Concurrent:        defaultConcurrent,
	}
}

// MirrorResultsParams contains parameters for processing mirror results.
type MirrorResultsParams struct {
	results  <-chan mirror.MirrorResult
	logger   *logrus.Logger
	flags    *MirrorFlags
	mirror   *mirror.DefaultMirror
	artifact *core.Artifact
	opts     *core.MirrorOptions
}

// processMirrorResults processes the results from the mirror operation.
func processMirrorResults(params MirrorResultsParams) error {
	var lastErr error

	for result := range params.results {
		if result.Error != nil {
			lastErr = result.Error
			params.logger.WithError(result.Error).Errorf("Failed to mirror %s", result.Artifact.Name)

			continue
		}

		// Handle unzipping if needed
		if params.flags.Unzip && archive.IsZipFile(result.DestinationPath) {
			extractParams := ZipExtractionParams{
				ctx:      params.opts.Context,
				logger:   params.logger,
				mirror:   params.mirror,
				artifact: params.artifact,
				opts:     params.opts,
			}

			if err := handleZipExtraction(extractParams); err != nil {
				lastErr = err
				params.logger.WithError(err).Error("Failed to extract zip file")
			}
		}
	}

	return lastErr
}

// processAndUploadArtifact processes and uploads a single artifact.
func processAndUploadArtifact(flags *MirrorFlags, jfrogConfig *destinations.JFrogConfig) error {
	ctx := context.Background()

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Create mirror instance
	mirror := createMirror(logger)

	// Create source
	source, err := setupSource(mirror, logger)
	if err != nil {
		return err
	}
	defer source.Close()

	// Create destination
	if err := setupDestination(mirror, logger, jfrogConfig); err != nil {
		return err
	}

	// Create artifact
	artifact := createArtifact(flags)

	// Create mirror options
	opts := createMirrorOptions(ctx, flags)

	// Start mirroring
	results := mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)

	// Process results
	params := MirrorResultsParams{
		results:  results,
		logger:   logger,
		flags:    flags,
		mirror:   mirror,
		artifact: artifact,
		opts:     opts,
	}

	return processMirrorResults(params)
}

// mirrorCmd represents the mirror command.
var mirrorCmd = &cobra.Command{
	Use:   "mirror",
	Short: "Mirror artifacts from a source to a destination",
	Long: `Mirror artifacts from a source to a destination.

This command will:
1. Download artifacts from the configured source
2. Process them according to their type (e.g., unzip if needed)
3. Upload them to all configured destinations

Examples:
  # Using configuration file
  garf mirror --config config.yaml

  # Using command line flags
  garf mirror \
    --source https://github.com/example/repo/releases/download/v1.0.0/artifact.zip \
    --destination sandbox-generic-local \
    --unzip`,
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := &MirrorFlags{}
		flags.addFlags(cmd)

		// Get configuration
		config, err := getConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		// If no config file is provided, validate required flags
		if flags.ConfigFile == "" {
			if flags.Source == "" {
				return fmt.Errorf("--source flag is required when not using --config")
			}
			if flags.Destination == "" {
				return fmt.Errorf("--destination flag is required when not using --config")
			}
		}

		// If source and destination are provided via flags, use them instead of config
		if flags.Source != "" {
			config.Source.URL = flags.Source
		}
		if flags.Destination != "" {
			config.Destination.URL = flags.Destination
		}

		// Setup logger
		logger := logrus.New()
		if config.LogLevel != "" {
			level, err := logrus.ParseLevel(config.LogLevel)
			if err != nil {
				return fmt.Errorf("invalid log level: %w", err)
			}
			logger.SetLevel(level)
		}

		// Create mirror instance
		mirror := mirror.NewDefaultMirror(logger)

		// Setup source
		source, err := mirror.SetupSource(logger, config)
		if err != nil {
			return fmt.Errorf("failed to setup source: %w", err)
		}
		if err := mirror.AddSource("default", source); err != nil {
			return fmt.Errorf("failed to add source: %w", err)
		}

		// Setup destination
		dest, err := mirror.SetupDestination(logger, config)
		if err != nil {
			return fmt.Errorf("failed to setup destination: %w", err)
		}
		if err := mirror.AddDestination("default", dest); err != nil {
			return fmt.Errorf("failed to add destination: %w", err)
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		// Create artifact from flags
		artifact := createArtifact(flags)

		// Create mirror options
		opts := createMirrorOptions(ctx, flags)

		// Start mirroring process
		results := mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)

		// Process results
		params := MirrorResultsParams{
			results:  results,
			logger:   logger,
			flags:    flags,
			mirror:   mirror,
			artifact: artifact,
			opts:     opts,
		}

		return processMirrorResults(params)
	},
}

// NewMirrorCmd creates a new mirror command.
func NewMirrorCmd() *cobra.Command {
	flags := &MirrorFlags{}
	mirrorCmd.Flags().StringVarP(&flags.ConfigFile, "config", "c", "", "Path to configuration file")
	// Remove the required flag since we now support both config and direct flags
	// mirrorCmd.MarkFlagRequired("config")

	return mirrorCmd
}

// getConfig reads and validates the configuration file.
func getConfig() (*config.Config, error) {
	// Load configuration from file
	return &config.Config{
		Source: config.SourceConfig{
			Type: "github",
			URL:  "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip",
		},
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			URL:      "https://artifactory.example.com/artifactory",
			User:     os.Getenv("JFROG_USER"),
			Password: os.Getenv("JFROG_PASSWORD"),
		},
		LogLevel:   "info",
		Concurrent: defaultConcurrent,
	}, nil
}
