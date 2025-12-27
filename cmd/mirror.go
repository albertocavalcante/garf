package cmd

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/core/config"
	"github.com/albertocavalcante/garf/pkg/io"
	"github.com/albertocavalcante/garf/pkg/mirror"
	netrc "github.com/albertocavalcante/netrcgo"
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
	ConfigFile      string
	Source          string
	Destination     string
	FromFile        string
	Raw             bool
	Properties      []string
	Unzip           bool
	PreserveZipName bool
	DryRun          bool
	DryRunMode      string
	SourcePathStrip string
	Checksum        string

	// Legacy JFrog-specific flags (backward compatibility, will be removed in future versions)
	JFrogURL               string
	JFrogUser              string
	JFrogPassword          string
	JFrogPasswordFromStdin bool

	// New generic registry flags (preferred for new usage)
	RegistryType              string
	RegistryURL               string
	RegistryUser              string
	RegistryPassword          string
	RegistryPasswordFromStdin bool

	TestMirror *mirror.DefaultMirror // Used for testing only
}

// ValidateDryRunMode checks if the dry run mode is valid.
// This method is added for test compatibility.
func (f *MirrorFlags) ValidateDryRunMode() error {
	if f.DryRun {
		return core.ValidateDryRunMode(f.DryRunMode)
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
	setupBasicFlags(cmd, flags)
	setupProcessingFlags(cmd, flags)
	setupJFrogFlags(cmd, flags)
	setupRegistryFlags(cmd, flags)
}

// setupBasicFlags configures basic mirror flags.
func setupBasicFlags(cmd *cobra.Command, flags *MirrorFlags) {
	cmd.Flags().StringVarP(&flags.ConfigFile, "config", "c", "", "Path to configuration file")
	cmd.Flags().StringVarP(&flags.Source, "source", "s", "", "GitHub Release URL to the artifact")
	cmd.Flags().StringVarP(&flags.Destination, "destination", "d", "", "Artifacts destination (e.g. sandbox-generic-local)")
	cmd.Flags().StringVarP(&flags.FromFile, "from-file", "f", "", "Skip Download. Upload from file and use URL to infer coordinates")
	cmd.Flags().StringVar(&flags.SourcePathStrip, "source-path-strip", "", "Strip path prefixes from source URLs before processing")
}

// setupProcessingFlags configures artifact processing flags.
func setupProcessingFlags(cmd *cobra.Command, flags *MirrorFlags) {
	cmd.Flags().BoolVar(&flags.Raw, "raw", false, "Raw Mirror. Keep the original URL structure in the destination path")
	cmd.Flags().StringArrayVar(&flags.Properties, "properties", []string{}, "Properties to attach to the artifact")
	cmd.Flags().BoolVar(&flags.Unzip, "unzip", false, "Unzip and upload content if source is a zip file with a single file inside")
	cmd.Flags().BoolVar(&flags.PreserveZipName, "preserve-zip-name", false, "Preserve the ZIP filename when extracting, replacing the ZIP extension with the extracted file's extension")
	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Perform a dry run without making actual changes")
	cmd.Flags().StringVar(&flags.DryRunMode, "dry-run-mode", "all", "Dry run mode: 'all' (skip all operations), 'upload' (skip only upload to Artifactory)")
	cmd.Flags().StringVar(&flags.Checksum, "checksum", "", "SHA256 checksum to verify the artifact against")
	cmd.Flags().StringVar(&flags.Checksum, "sha256", "", "Alias for --checksum")
	cmd.Flags().MarkHidden("sha256")
}

// setupJFrogFlags configures JFrog-specific flags.
func setupJFrogFlags(cmd *cobra.Command, flags *MirrorFlags) {
	cmd.Flags().StringVar(&flags.JFrogURL, "jfrog-url", "", "JFrog Artifactory URL (can also be set via JFROG_URL env var)")
	cmd.Flags().StringVar(&flags.JFrogUser, "jfrog-user", "", "JFrog Artifactory username (can also be set via JFROG_USER env var)")
	cmd.Flags().StringVar(&flags.JFrogPassword, "jfrog-password", "", "JFrog Artifactory password (can also be set via JFROG_PASSWORD env var)")
	cmd.Flags().BoolVar(&flags.JFrogPasswordFromStdin, "jfrog-password-stdin", false, "Read JFrog password from stdin")
}

// setupRegistryFlags configures generic registry flags.
func setupRegistryFlags(cmd *cobra.Command, flags *MirrorFlags) {
	cmd.Flags().StringVar(&flags.RegistryType, "registry-type", "jfrog", "Registry type: 'jfrog' or 'cloudsmith'")
	cmd.Flags().StringVar(&flags.RegistryURL, "registry-url", "", "Registry URL (can also be set via REGISTRY_URL env var)")
	cmd.Flags().StringVar(&flags.RegistryUser, "registry-user", "", "Registry username (can also be set via REGISTRY_USER env var)")
	cmd.Flags().StringVar(&flags.RegistryPassword, "registry-password", "", "Registry password (can also be set via REGISTRY_PASSWORD env var)")
	cmd.Flags().BoolVar(&flags.RegistryPasswordFromStdin, "registry-password-stdin", false, "Read registry password from stdin")
}

// runMirror executes the mirror operation.
func runMirror(flags *MirrorFlags) error {
	// Setup and validation
	cfg, logger, err := initializeMirrorOperation(flags)
	if err != nil {
		return err
	}

	// Setup mirror
	m, artifact, opts, err := prepareMirrorExecution(logger, cfg, flags)
	if err != nil {
		return err
	}

	// Execute mirroring and handle results
	return executeMirroring(logger, m, artifact, opts)
}

// initializeMirrorOperation handles validation, config loading, and logger setup.
func initializeMirrorOperation(flags *MirrorFlags) (*config.Config, *logrus.Logger, error) {
	if err := validateFlags(flags); err != nil {
		return nil, nil, err
	}

	cfg, err := getConfig(flags)
	if err != nil {
		return nil, nil, fmt.Errorf("configuration error: %w", err)
	}

	logger := setupLogger(cfg, flags)
	logConfigurationInfo(logger, flags, cfg)

	return cfg, logger, nil
}

// prepareMirrorExecution sets up the mirror, artifact, and options.
func prepareMirrorExecution(logger *logrus.Logger, cfg *config.Config, flags *MirrorFlags) (*mirror.DefaultMirror, *core.Artifact, *core.MirrorOptions, error) {
	m, err := setupMirror(logger, cfg, flags)
	if err != nil {
		logger.WithError(err).Error("Failed to setup mirror")

		return nil, nil, nil, err
	}

	logger.Debug("Mirror setup completed successfully")

	artifact, opts := createArtifactAndOptions(flags, cfg)
	logArtifactInfo(logger, artifact)

	return m, artifact, opts, nil
}

// executeMirroring runs the mirror operation and processes results.
func executeMirroring(logger *logrus.Logger, m *mirror.DefaultMirror, artifact *core.Artifact, opts *core.MirrorOptions) error {
	logger.Info("Starting mirror execution")

	results := m.Mirror(opts.Context, []*core.Artifact{artifact}, opts)

	var lastErr error

	for result := range results {
		if result.Error != nil {
			lastErr = result.Error
			logger.WithError(result.Error).Errorf("Failed to mirror %s", result.Artifact.Name)

			continue
		}

		logSuccessfulMirror(logger, result)
	}

	logMirrorCompletion(logger, lastErr)

	return lastErr
}

// logConfigurationInfo logs the mirror operation configuration.
func logConfigurationInfo(logger *logrus.Logger, flags *MirrorFlags, cfg *config.Config) {
	logger.WithFields(logrus.Fields{
		"source_url":    flags.Source,
		"destination":   flags.Destination,
		"registry_url":  cfg.Destination.URL,
		"registry_user": cfg.Destination.User,
		"raw_mode":      flags.Raw,
		"unzip":         flags.Unzip,
		"dry_run":       flags.DryRun,
		"dry_run_mode":  flags.DryRunMode,
	}).Info("Starting mirror operation with configuration")
}

// logArtifactInfo logs artifact information.
func logArtifactInfo(logger *logrus.Logger, artifact *core.Artifact) {
	logger.WithFields(logrus.Fields{
		"artifact_name":     artifact.Name,
		"artifact_location": artifact.Location,
		"metadata":          artifact.Metadata,
	}).Info("Created artifact for mirroring")
}

// logSuccessfulMirror logs successful mirror results.
func logSuccessfulMirror(logger *logrus.Logger, result mirror.MirrorResult) {
	logger.WithFields(logrus.Fields{
		"artifact_name":    result.Artifact.Name,
		"destination_path": result.DestinationPath,
	}).Info("Successfully mirrored artifact")

	if result.DestinationPath != "" {
		logger.Infof("✓ Mirrored to: %s", result.DestinationPath)
	}
}

// logMirrorCompletion logs the final result of the mirror operation.
func logMirrorCompletion(logger *logrus.Logger, lastErr error) {
	if lastErr == nil {
		logger.Info("Mirror operation completed successfully")
	} else {
		logger.WithError(lastErr).Error("Mirror operation completed with errors")
	}
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
	// Handle passwords from stdin
	registryPassword, jfrogPassword, err := resolvePasswordsFromStdin(flags)
	if err != nil {
		return nil, err
	}

	// Resolve credentials with potential netrc fallback
	creds, err := resolveCredentialsWithNetrc(flags, registryPassword, jfrogPassword)
	if err != nil {
		return nil, err
	}

	// Build configuration using builder pattern
	return config.NewBuilder().
		WithSource(flags.Source, flags.SourcePathStrip).
		WithRegistryCredentials(creds).
		WithDestination(flags.Destination, flags.SourcePathStrip).
		Build()
}

// resolvePasswordsFromStdin handles reading passwords from stdin.
func resolvePasswordsFromStdin(flags *MirrorFlags) (string, string, error) {
	registryPassword := flags.RegistryPassword

	if flags.RegistryPasswordFromStdin {
		password, err := io.ReadPasswordFromStdin()
		if err != nil {
			return "", "", fmt.Errorf("failed to read registry password from stdin: %w", err)
		}

		registryPassword = password
	}

	jfrogPassword := flags.JFrogPassword

	if flags.JFrogPasswordFromStdin {
		password, err := io.ReadPasswordFromStdin()
		if err != nil {
			return "", "", fmt.Errorf("failed to read JFrog password from stdin: %w", err)
		}

		jfrogPassword = password
	}

	return registryPassword, jfrogPassword, nil
}

// resolveCredentialsWithNetrc resolves credentials and applies netrc fallback if needed.
func resolveCredentialsWithNetrc(flags *MirrorFlags, registryPassword, jfrogPassword string) (*config.RegistryCredentials, error) {
	resolver := config.NewCredentialsResolver()

	registryConfig := &config.RegistryConfig{
		URL:              flags.RegistryURL,
		User:             flags.RegistryUser,
		Password:         registryPassword,
		Type:             flags.RegistryType,
		PasswordFromStdin: false, // passwordFromStdin already handled
	}

	jfrogConfig := &config.JFrogConfig{
		URL:              flags.JFrogURL,
		User:             flags.JFrogUser,
		Password:         jfrogPassword,
		PasswordFromStdin: false, // passwordFromStdin already handled
	}

	creds, err := resolver.ResolveCredentials(registryConfig, jfrogConfig)

	if err != nil && creds != nil && creds.Type == "jfrog" && (creds.User == "" || creds.Password == "") {
		return tryNetrcFallback(creds, err)
	}

	return creds, err
}

// tryNetrcFallback attempts to fill missing JFrog credentials from netrc.
func tryNetrcFallback(creds *config.RegistryCredentials, originalErr error) (*config.RegistryCredentials, error) {
	user, password, netrcErr := tryNetrcCredentials(creds.URL, creds.User, creds.Password)
	if netrcErr != nil {
		return nil, originalErr
	}

	creds.User = user
	creds.Password = password

	if creds.User == "" || creds.Password == "" {
		_, _, validateErr := validateCredentials(user, password, netrcErr)

		return nil, validateErr
	}

	return creds, nil
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
		Context:         ctx,
		Raw:             flags.Raw,
		Concurrent:      defaultConcurrent,
		DryRun:          flags.DryRun,
		DryRunMode:      flags.DryRunMode,
		Unzip:           flags.Unzip,
		PreserveZipName: flags.PreserveZipName,
		Checksum:        flags.Checksum,
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
