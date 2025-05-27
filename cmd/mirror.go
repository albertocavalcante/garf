package cmd

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/core/config"
	"github.com/albertocavalcante/garf/pkg/io"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/albertocavalcante/garf/pkg/netrc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultTimeout    = 30 * time.Minute
	defaultConcurrent = 4
)

// MirrorFlags holds the command-line flags for the mirror command.
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

// ValidateDryRunMode checks if the dry run mode is valid.
// This method is added for test compatibility.
func (f *MirrorFlags) ValidateDryRunMode() error {
	if f.DryRun {
		validModes := map[string]bool{"all": true, "upload": true}
		if !validModes[f.DryRunMode] {
			return fmt.Errorf("invalid dry run mode: %s. Valid modes are: all, upload", f.DryRunMode)
		}
	}

	return nil
}

// RunE is a compatibility method for tests.
func (f *MirrorFlags) RunE(cmd *cobra.Command, args []string) error {
	return runMirror(f)
}

// ValidateAndGetConfig is a legacy function for test compatibility.
// It's a wrapper around buildConfigFromFlags.
func ValidateAndGetConfig(flags *MirrorFlags) (*config.Config, error) {
	// Validate source and destination similar to validateFlags but for tests
	if flags.Source == "" {
		return nil, fmt.Errorf("source cannot be empty")
	}

	if flags.Destination == "" {
		return nil, fmt.Errorf("destination cannot be empty")
	}

	return buildConfigFromFlags(flags)
}

// NewMirrorCmd creates a new mirror command.
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
			return runMirror(flags)
		},
	}

	// Setup flags
	setupMirrorFlags(cmd, flags)

	return cmd
}

// setupMirrorFlags configures all flags for the mirror command.
func setupMirrorFlags(cmd *cobra.Command, flags *MirrorFlags) {
	cmd.Flags().StringVarP(&flags.ConfigFile, "config", "c", "", "Path to configuration file")
	cmd.Flags().StringVarP(&flags.Source, "source", "s", "", "GitHub Release URL to the artifact")
	cmd.Flags().StringVarP(
		&flags.Destination,
		"destination",
		"d",
		"",
		"Artifacts destination (e.g. sandbox-generic-local)",
	)
	cmd.Flags().StringVarP(
		&flags.FromFile,
		"from-file",
		"f",
		"",
		"Skip Download. Upload from file and use URL to infer coordinates",
	)
	cmd.Flags().BoolVar(&flags.Raw, "raw", false, "Raw Mirror. Keep the original URL structure in the destination path")
	cmd.Flags().StringArrayVar(
		&flags.Properties,
		"properties",
		[]string{},
		"Properties to attach to the artifact (e.g. type=toolchain platform=windows)",
	)
	cmd.Flags().BoolVar(
		&flags.Unzip,
		"unzip",
		false,
		"Unzip and upload content if source is a zip file with a single file inside",
	)
	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Perform a dry run without making actual changes")
	cmd.Flags().StringVar(
		&flags.DryRunMode,
		"dry-run-mode",
		"all",
		"Dry run mode: 'all' (skip all operations), 'upload' (skip only upload to Artifactory)",
	)
	cmd.Flags().StringVar(
		&flags.JFrogURL,
		"jfrog-url",
		"",
		"JFrog Artifactory URL (can also be set via JFROG_URL env var)",
	)
	cmd.Flags().StringVar(
		&flags.JFrogUser,
		"jfrog-user",
		"",
		"JFrog Artifactory username (can also be set via JFROG_USER env var)",
	)
	cmd.Flags().StringVar(
		&flags.JFrogPassword,
		"jfrog-password",
		"",
		"JFrog Artifactory password (can also be set via JFROG_PASSWORD env var)",
	)
	cmd.Flags().BoolVar(
		&flags.JFrogPasswordFromStdin,
		"jfrog-password-stdin",
		false,
		"Read JFrog Artifactory password from stdin (more secure than --jfrog-password)",
	)
}

// runMirror executes the mirror operation.
func runMirror(flags *MirrorFlags) error {
	// Validate flags
	if err := validateFlags(flags); err != nil {
		return err
	}

	// Get configuration
	cfg, err := getConfig(flags)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Setup logger
	logger := setupLogger(cfg, flags)

	// Log configuration (without sensitive data)
	logger.WithFields(logrus.Fields{
		"source_url":   flags.Source,
		"destination":  flags.Destination,
		"jfrog_url":    cfg.Destination.URL,
		"jfrog_user":   cfg.Destination.User,
		"raw_mode":     flags.Raw,
		"unzip":        flags.Unzip,
		"dry_run":      flags.DryRun,
		"dry_run_mode": flags.DryRunMode,
	}).Info("Starting mirror operation with configuration")

	// Create mirror with sources and destinations
	m, err := setupMirror(logger, cfg, flags)
	if err != nil {
		logger.WithError(err).Error("Failed to setup mirror")

		return err
	}

	logger.Debug("Mirror setup completed successfully")

	// Create artifact and options
	artifact, opts := createArtifactAndOptions(flags, cfg)

	logger.WithFields(logrus.Fields{
		"artifact_name":     artifact.Name,
		"artifact_location": artifact.Location,
		"metadata":          artifact.Metadata,
	}).Info("Created artifact for mirroring")

	// Execute mirroring
	logger.Info("Starting mirror execution")

	results := m.Mirror(opts.Context, []*core.Artifact{artifact}, opts)

	// Process results
	var lastErr error

	for result := range results {
		if result.Error != nil {
			lastErr = result.Error
			logger.WithError(result.Error).Errorf("Failed to mirror %s", result.Artifact.Name)

			continue
		}

		logger.WithFields(logrus.Fields{
			"artifact_name":    result.Artifact.Name,
			"destination_path": result.DestinationPath,
		}).Info("Successfully mirrored artifact")
	}

	if lastErr == nil {
		logger.Info("Mirror operation completed successfully")
	} else {
		logger.WithError(lastErr).Error("Mirror operation completed with errors")
	}

	return lastErr
}

// validateFlags checks if flags are valid.
func validateFlags(flags *MirrorFlags) error {
	// Validate dry run mode
	if err := flags.ValidateDryRunMode(); err != nil {
		return err
	}

	// If not using config file, source and destination are required
	if flags.ConfigFile == "" && (flags.Source == "" || flags.Destination == "") {
		return fmt.Errorf("required flag(s) \"destination\", \"source\" not set")
	}

	return nil
}

// getConfig loads configuration from file or builds it from flags.
func getConfig(flags *MirrorFlags) (*config.Config, error) {
	// If config file provided, load it
	if flags.ConfigFile != "" {
		return loadConfigFromFile(flags.ConfigFile)
	}

	// Otherwise, build config from flags and environment
	return buildConfigFromFlags(flags)
}

// loadConfigFromFile loads configuration from a file.
func loadConfigFromFile(configFile string) (*config.Config, error) {
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// buildConfigFromFlags creates configuration from flags and environment variables.
func buildConfigFromFlags(flags *MirrorFlags) (*config.Config, error) {
	// Setup viper for environment variables
	v := viper.New()
	v.SetEnvPrefix("JFROG")
	v.AutomaticEnv()

	// Get JFrog credentials
	jfrogURL, err := getJFrogURL(flags, v)
	if err != nil {
		return nil, err
	}

	jfrogUser, jfrogPassword, err := GetJFrogCredentials(jfrogURL, flags, v)
	if err != nil {
		return nil, err
	}

	// Create config
	return &config.Config{
		Source: config.SourceConfig{
			Type: "github",
			URL:  flags.Source,
		},
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			URL:      jfrogURL,
			User:     jfrogUser,
			Password: jfrogPassword,
			DestPath: flags.Destination,
		},
		LogLevel:   "info",
		Concurrent: defaultConcurrent,
	}, nil
}

// getJFrogURL retrieves and normalizes the JFrog URL from flags or environment.
func getJFrogURL(flags *MirrorFlags, v *viper.Viper) (string, error) {
	// Get JFrog URL (from flag or environment)
	jfrogURL := flags.JFrogURL
	if jfrogURL == "" {
		jfrogURL = v.GetString("URL")
	}

	// Normalize URL
	if jfrogURL != "" {
		// Add scheme if missing
		if !strings.HasPrefix(jfrogURL, "http://") && !strings.HasPrefix(jfrogURL, "https://") {
			jfrogURL = "http://" + jfrogURL
		}

		// Ensure /artifactory path
		if !strings.Contains(jfrogURL, "/artifactory") {
			jfrogURL = strings.TrimSuffix(jfrogURL, "/") + "/artifactory"
		}
	}

	// Validate
	if jfrogURL == "" {
		return "", fmt.Errorf("JFrog URL is required")
	}

	return jfrogURL, nil
}

// GetJFrogCredentials retrieves JFrog credentials from flags, environment variables, or .netrc.
func GetJFrogCredentials(jfrogURL string, flags *MirrorFlags, v *viper.Viper) (string, string, error) {
	jfrogUser := flags.JFrogUser
	if jfrogUser == "" {
		jfrogUser = v.GetString("USER")
	}

	jfrogPassword := flags.JFrogPassword
	if jfrogPassword == "" {
		jfrogPassword = v.GetString("PASSWORD")
	}

	if flags.JFrogPasswordFromStdin {
		password, err := io.ReadPasswordFromStdin()
		if err != nil {
			return "", "", fmt.Errorf("failed to read password from stdin: %w", err)
		}

		jfrogPassword = password
	}

	var netrcErr error
	if jfrogUser == "" || jfrogPassword == "" {
		jfrogUser, jfrogPassword, netrcErr = tryNetrcCredentials(jfrogURL, jfrogUser, jfrogPassword)
	}

	return validateCredentials(jfrogUser, jfrogPassword, netrcErr)
}

// tryNetrcCredentials attempts to get missing credentials from .netrc.
func tryNetrcCredentials(jfrogURL, currentUser, currentPassword string) (string, string, error) {
	host, err := ExtractHostFromURL(jfrogURL)
	if err != nil {
		return currentUser, currentPassword, err
	}

	creds, found, err := netrc.GetHostCredentials(host)
	if err != nil {
		return currentUser, currentPassword, err
	}

	if found {
		if currentUser == "" {
			currentUser = creds.Login
		}

		if currentPassword == "" {
			currentPassword = creds.Password
		}
	}

	return currentUser, currentPassword, nil
}

// validateCredentials checks if credentials are complete and returns appropriate errors.
func validateCredentials(user, password string, netrcErr error) (string, string, error) {
	if user == "" {
		if netrcErr != nil {
			return "", "", fmt.Errorf("JFrog user is required and .netrc lookup failed: %w", netrcErr)
		}

		return "", "", fmt.Errorf("JFrog user is required")
	}

	if password == "" {
		if netrcErr != nil {
			return "", "", fmt.Errorf("JFrog password is required and .netrc lookup failed: %w", netrcErr)
		}

		return "", "", fmt.Errorf("JFrog password is required")
	}

	return user, password, nil
}

// setupLogger creates and configures a logger.
func setupLogger(cfg *config.Config, flags *MirrorFlags) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	if cfg.LogLevel != "" {
		if level, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
			logger.SetLevel(level)
		}
	}

	// Configure formatter
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		DisableColors:          false,
		DisableLevelTruncation: true,
		PadLevelText:           true,
		ForceColors:            true,
	})

	// Log mode information
	if flags.DryRun {
		logger.WithField("mode", flags.DryRunMode).Info("Running in dry run mode")
	}

	logger.WithField("raw", flags.Raw).Debug("Raw flag status")

	return logger
}

// setupMirror creates and configures a mirror.
func setupMirror(logger *logrus.Logger, cfg *config.Config, flags *MirrorFlags) (*mirror.DefaultMirror, error) {
	// Use test mirror if provided (for testing)
	if flags.TestMirror != nil {
		return flags.TestMirror, nil
	}

	// Create new mirror
	m := mirror.NewDefaultMirror(logger)

	// Setup source
	source, err := mirror.SetupSource(logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to setup source: %w", err)
	}

	if err := m.AddSource("default", source); err != nil {
		return nil, fmt.Errorf("failed to add source: %w", err)
	}

	// Setup destination if not in dry run mode or in upload-only dry run mode
	if !flags.DryRun || flags.DryRunMode == "upload" {
		dest, err := mirror.SetupDestination(logger, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to setup destination: %w", err)
		}

		if err := m.AddDestination("default", dest); err != nil {
			return nil, fmt.Errorf("failed to add destination: %w", err)
		}
	}

	return m, nil
}

// createArtifactAndOptions creates an artifact and mirror options.
func createArtifactAndOptions(flags *MirrorFlags, cfg *config.Config) (*core.Artifact, *core.MirrorOptions) {
	// Create artifact
	artifact := &core.Artifact{
		Name:     filepath.Base(cfg.Source.URL),
		Location: cfg.Source.URL,
		Metadata: core.ParseProperties(flags.Properties),
	}

	// Override location if fromFile is specified
	if flags.FromFile != "" {
		artifact.Location = flags.FromFile
	}

	// Create options
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

	// We can't defer cancel() here because the context needs to outlive this function
	// The caller should ensure it's properly canceled when no longer needed
	_ = cancel // Prevent linter warnings about unused variables

	opts := &core.MirrorOptions{
		Context:    ctx,
		Raw:        flags.Raw,
		Concurrent: defaultConcurrent,
		DryRun:     flags.DryRun,
		DryRunMode: flags.DryRunMode,
		Unzip:      flags.Unzip,
	}

	return artifact, opts
}

// ExtractHostFromURL parses the provided URL string and returns the host without scheme.
func ExtractHostFromURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse JFrog URL %q: %w", rawURL, err)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("URL %q does not contain a valid host", rawURL)
	}

	return parsed.Host, nil
}
