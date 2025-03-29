package cmd

import (
	"bufio"
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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	propertyKeyValueParts = 2
	defaultTimeout        = 30 * time.Minute
	defaultConcurrent     = 4
	percentageMultiplier  = 100
)

// MirrorFlags represents the flags for the mirror command.
type MirrorFlags struct {
	ConfigFile             string
	Source                 string
	Destination            string
	FromFile               string
	Raw                    bool
	Properties             []string
	Unzip                  bool
	DryRun                 bool
	DryRunMode             string
	JFrogURL               string
	JFrogUser              string
	JFrogPassword          string
	JFrogPasswordFromStdin bool
	TestMirror             *mirror.DefaultMirror // Used for testing only
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
	flags.BoolVar(
		&f.DryRun,
		"dry-run",
		false,
		"Perform a dry run without making actual changes",
	)
	flags.StringVar(
		&f.DryRunMode,
		"dry-run-mode",
		"all",
		"Dry run mode: 'all' (skip all operations), 'upload' (skip only upload to Artifactory)",
	)
	flags.StringVar(
		&f.JFrogURL,
		"jfrog-url",
		"",
		"JFrog Artifactory URL (can also be set via JFROG_URL env var)",
	)
	flags.StringVar(
		&f.JFrogUser,
		"jfrog-user",
		"",
		"JFrog Artifactory username (can also be set via JFROG_USER env var)",
	)
	flags.StringVar(
		&f.JFrogPassword,
		"jfrog-password",
		"",
		"JFrog Artifactory password (can also be set via JFROG_PASSWORD env var)",
	)
	flags.BoolVar(
		&f.JFrogPasswordFromStdin,
		"jfrog-password-stdin",
		false,
		"Read JFrog Artifactory password from stdin (more secure than --jfrog-password)",
	)
}

// ParseProperties converts a slice of property strings formatted as "key=value" into a map. It splits each string at the first "=" character, trimming any surrounding whitespace from both the key and value. Only properties that result in exactly two parts are included in the returned map.
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
// Password priority: command-line flag > environment variable
// ValidateAndGetConfig validates that the mirror source, destination, and JFrog Artifactory credentials (URL, user, and password) are provided via command-line flags or environment variables, and returns a new JFrog configuration. It prioritizes flag values over environment variables and supports supplying the password via stdin using the --jfrog-password-stdin flag. An error is returned if any required value is missing.
func ValidateAndGetConfig(
	source, destination, jfrogURL, jfrogUser, jfrogPassword string,
) (*destinations.JFrogConfig, error) {
	if source == "" || destination == "" {
		return nil, fmt.Errorf("required flag(s) \"destination\", \"source\" not set")
	}

	// Priority: command-line flags > environment variables
	url := jfrogURL
	if url == "" {
		var ok bool

		url, ok = os.LookupEnv("JFROG_URL")
		if !ok {
			return nil, fmt.Errorf("JFrog URL is required via --jfrog-url flag or JFROG_URL environment variable")
		}
	}

	user := jfrogUser
	if user == "" {
		var ok bool

		user, ok = os.LookupEnv("JFROG_USER")
		if !ok {
			return nil, fmt.Errorf("JFrog user is required via --jfrog-user flag or JFROG_USER environment variable")
		}
	}

	password := jfrogPassword
	if password == "" {
		var ok bool

		password, ok = os.LookupEnv("JFROG_PASSWORD")
		if !ok {
			return nil, fmt.Errorf(
				"JFrog password is required via --jfrog-password flag, --jfrog-password-stdin flag, or JFROG_PASSWORD env var",
			)
		}
	}

	return &destinations.JFrogConfig{
		URL:      url,
		User:     user,
		Password: password,
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

// HandleZipExtraction extracts a single file from the zip archive specified in the artifact's location,
// updates the artifact with the extracted file's name and location, and mirrors the extracted content using
// the provided mirror, options, and context. It returns an error if any step of the extraction or mirroring process fails.
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

// setupSource configures and adds a source to the mirror.
func (f *MirrorFlags) setupSource(m *mirror.DefaultMirror, logger *logrus.Logger, config *config.Config) error {
	source, err := m.SetupSource(logger, config)
	if err != nil {
		return fmt.Errorf("failed to setup source: %w", err)
	}

	if err := m.AddSource("default", source); err != nil {
		return fmt.Errorf("failed to add source: %w", err)
	}

	return nil
}

// setupDestination configures and adds a destination to the mirror.
func (f *MirrorFlags) setupDestination(m *mirror.DefaultMirror, logger *logrus.Logger, config *config.Config) error {
	dest, err := m.SetupDestination(logger, config)
	if err != nil {
		return fmt.Errorf("failed to setup destination: %w", err)
	}

	if err := m.AddDestination("default", dest); err != nil {
		return fmt.Errorf("failed to add destination: %w", err)
	}

	return nil
}

// createArtifact constructs a new artifact based on the provided mirror flags.
// It sets the artifact's name using the base name of the source path and initially assigns the source as its location.
// If a file path is specified via the FromFile flag, that value overrides the source for the location.
// Additionally, it parses the Properties flag to populate the artifact's metadata.
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

// createMirrorOptions constructs a new MirrorOptions instance configured from command-line flags and context.
// It inverts the Raw flag to determine whether to preserve the original structure,
// applies a default concurrency level, and sets dry-run options based on the DryRun and DryRunMode flags.
// The provided context supports cancellation of the mirroring operation.
func createMirrorOptions(ctx context.Context, flags *MirrorFlags) *core.MirrorOptions {
	return &core.MirrorOptions{
		PreserveStructure: !flags.Raw,
		Context:           ctx,
		Concurrent:        defaultConcurrent,
		DryRun:            flags.DryRun,
		DryRunMode:        flags.DryRunMode,
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

// processMirrorResults iterates over mirror operation results, logging any errors encountered during mirroring.
// If the Unzip flag is enabled and a destination file is identified as a zip archive, it attempts extraction using the provided parameters.
// It returns the last error encountered during processing, or nil if all operations complete successfully.
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

// ValidateDryRunMode validates the dry run mode.
func (f *MirrorFlags) ValidateDryRunMode() error {
	if !f.DryRun {
		return nil
	}

	validModes := map[string]bool{
		"all":    true,
		"upload": true,
	}

	if !validModes[f.DryRunMode] {
		return fmt.Errorf("invalid dry run mode: %s. Valid modes are: all, upload", f.DryRunMode)
	}

	return nil
}

// getConfig reads and validates the configuration file.
func (f *MirrorFlags) getConfig() (*config.Config, error) {
	if f.ConfigFile != "" {
		// Read config from file
		cfg, err := config.Load(f.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}

		// Validate config
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		return cfg, nil
	}

	// Return default config
	return &config.Config{
		Source: config.SourceConfig{
			Type: "github",
			URL:  f.Source,
		},
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			URL:      f.Destination,
			User:     "",
			Password: "",
		},
		LogLevel:   "info",
		Concurrent: defaultConcurrent,
	}, nil
}

// setupLogger creates and configures a logger instance.
func (f *MirrorFlags) setupLogger(config *config.Config) (*logrus.Logger, error) {
	logger := logrus.New()

	if config.LogLevel != "" {
		level, err := logrus.ParseLevel(config.LogLevel)
		if err != nil {
			return nil, fmt.Errorf("invalid log level: %w", err)
		}

		logger.SetLevel(level)
	}

	// Configure colorful and structured logging
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		DisableColors:          false,
		DisableLevelTruncation: true,
		PadLevelText:           true,
		ForceColors:            true,
	})

	// Log dry run mode
	if f.DryRun {
		logger.WithField("mode", f.DryRunMode).Info("Running in dry run mode")
	}

	return logger, nil
}

// setupTestMirror creates a new mirror instance with source and destination.
func (f *MirrorFlags) setupTestMirror(logger *logrus.Logger, config *config.Config) (*mirror.DefaultMirror, error) {
	m := mirror.NewDefaultMirror(logger)

	// Setup source
	if err := f.setupSource(m, logger, config); err != nil {
		return nil, err
	}

	// Setup destination only if not in dry run mode or if in upload-only dry run mode
	if !f.DryRun || f.DryRunMode == "upload" {
		if err := f.setupDestination(m, logger, config); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// setupMirror creates and configures a mirror instance.
func (f *MirrorFlags) setupMirror(logger *logrus.Logger, config *config.Config) (*mirror.DefaultMirror, error) {
	// Create or use test mirror
	if f.TestMirror != nil {
		return f.TestMirror, nil
	}

	return f.setupTestMirror(logger, config)
}

// validateSourceAndDestination validates if source and destination are set when not using a config file.
func (f *MirrorFlags) validateSourceAndDestination(config *config.Config) error {
	if f.ConfigFile == "" && (config.Source.URL == "" || config.Destination.URL == "") {
		return fmt.Errorf("required flag(s) \"destination\", \"source\" not set")
	}

	return nil
}

// prepareArtifact creates an artifact from config source URL and prepares mirror options.
func (f *MirrorFlags) prepareArtifact(ctx context.Context) (*core.Artifact, *core.MirrorOptions) {
	artifact := createArtifact(f)
	opts := createMirrorOptions(ctx, f)

	return artifact, opts
}

// readPasswordFromStdin reads a password from the standard input (stdin).
//
// It returns the password if successfully read and non-empty. If reading from stdin fails,
// if no input is provided, or if the trimmed password is empty, it returns an error.
func readPasswordFromStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read password from stdin: %w", err)
		}

		return "", fmt.Errorf("no password provided via stdin")
	}

	password := scanner.Text()
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	return password, nil
}

// updateConfigFromFlags updates the configuration with values from command-line flags.
func (f *MirrorFlags) updateConfigFromFlags(config *config.Config) error {
	// If source and destination are provided via flags, use them instead of config
	if f.Source != "" {
		config.Source.URL = f.Source
	}

	if f.Destination != "" {
		config.Destination.URL = f.Destination
	}

	// If JFrog credentials are provided via flags, use them
	if f.JFrogURL != "" {
		config.Destination.URL = f.JFrogURL
	}

	if f.JFrogUser != "" {
		config.Destination.User = f.JFrogUser
	}

	// Handle password from stdin if requested
	if f.JFrogPasswordFromStdin {
		password, err := readPasswordFromStdin()
		if err != nil {
			return err
		}

		config.Destination.Password = password
	} else if f.JFrogPassword != "" {
		config.Destination.Password = f.JFrogPassword
	}

	return nil
}

// validateConfig validates the configuration based on whether a config file is provided.
func (f *MirrorFlags) validateConfig(config *config.Config) error {
	if f.ConfigFile != "" {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}
	} else if err := f.validateSourceAndDestination(config); err != nil {
		return err
	}

	return nil
}

// RunE executes the mirror command.
func (f *MirrorFlags) RunE(cmd *cobra.Command, args []string) error {
	// Validate dry run mode first
	if err := f.ValidateDryRunMode(); err != nil {
		return err
	}

	// Get configuration
	config, err := f.getConfig()
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Update config from flags
	if err := f.updateConfigFromFlags(config); err != nil {
		return err
	}

	// Validate configuration
	if err := f.validateConfig(config); err != nil {
		return err
	}

	// Setup logger
	logger, err := f.setupLogger(config)
	if err != nil {
		return err
	}

	// Setup mirror
	m, err := f.setupMirror(logger, config)
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Create artifact from config source URL
	f.Source = config.Source.URL // Set the source URL from config
	artifact, opts := f.prepareArtifact(ctx)

	// Start mirroring process
	results := m.Mirror(ctx, []*core.Artifact{artifact}, opts)

	// Process results
	params := MirrorResultsParams{
		results:  results,
		logger:   logger,
		flags:    f,
		mirror:   m,
		artifact: artifact,
		opts:     opts,
	}

	return processMirrorResults(params)
}

// NewMirrorCmd returns a new Cobra command configured to mirror artifacts from a source to a destination.
// It sets up the command's usage, description, examples, and flags—enabling users to mirror artifacts using either a configuration file or direct command-line flags.
func NewMirrorCmd() *cobra.Command {
	flags := &MirrorFlags{}
	cmd := &cobra.Command{
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
    --unzip

  # Dry run modes
  garf mirror --source <url> --destination <repo> --dry-run  # Skip all operations
  garf mirror --source <url> --destination <repo> --dry-run --dry-run-mode upload  # Skip only upload`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flags.RunE(cmd, args)
		},
	}
	cmd.Flags().StringVarP(&flags.ConfigFile, "config", "c", "", "Path to configuration file")
	flags.addFlags(cmd)

	return cmd
}
