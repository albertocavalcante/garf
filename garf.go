// Package garf provides a high-level API for mirroring artifacts from sources
// like GitHub to destinations like JFrog Artifactory.
//
// This package offers a simple, stable API for Go programs that want to mirror artifacts programmatically.
// It handles authentication, URL processing, and artifact uploading automatically.
//
// Basic usage:
//
//	client, err := garf.NewClient(garf.Config{
//		JFrogURL:      "https://mycompany.jfrog.io/artifactory",
//		JFrogUser:     "username",
//		JFrogPassword: "password",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	result, err := client.Mirror(ctx, garf.MirrorRequest{
//		Source:      "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
//		Destination: "my-generic-repo",
//		Properties:  map[string]string{"type": "binary", "platform": "linux"},
//		Unzip:       true, // Extract single files from zip archives
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
// For JFrog-to-JFrog mirroring with source path stripping:
//
//	result, err := client.Mirror(ctx, garf.MirrorRequest{
//		Source:          "https://artifactory.corp.net/staging/github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
//		Destination:     "prod-repo",
//		SourcePathStrip: "artifactory.corp.net/staging/", // Strip staging prefix
//		Properties:      map[string]string{"type": "binary"},
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
package garf

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/destinations"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/albertocavalcante/garf/pkg/processor"
	"github.com/albertocavalcante/garf/pkg/sources"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultTimeout is the default timeout for mirror operations.
	DefaultTimeout = 30 * time.Minute
	// DefaultConcurrent is the default number of concurrent operations.
	DefaultConcurrent = 4
	// UnknownArtifactName is returned when artifact name cannot be determined.
	UnknownArtifactName = "unknown"
)

// Config holds the configuration for the garf client.
type Config struct {
	// Generic registry credentials (PREFERRED - supports both JFrog and Cloudsmith)
	RegistryURL      string // Replaces JFrogURL but supports both
	RegistryUser     string // Replaces JFrogUser but supports both
	RegistryPassword string // Replaces JFrogPassword but supports both

	// Registry type (defaults to "jfrog" for backward compatibility)
	RegistryType string

	// DEPRECATED: JFrog-specific fields (kept for backward compatibility)
	// These will be mapped to Registry* fields if Registry* fields are empty
	JFrogURL      string
	JFrogUser     string
	JFrogPassword string

	// Optional: Custom logger (if nil, a default logger will be used)
	Logger *logrus.Logger

	// Optional: Request timeout (default: 30 minutes)
	Timeout time.Duration

	// Optional: Number of concurrent operations (default: 4)
	Concurrent int
}

// String implements fmt.Stringer to prevent accidental credential logging.
func (c Config) String() string {
	return fmt.Sprintf("Config{RegistryType: %s, RegistryURL: %s, RegistryUser: %s, RegistryPassword: [REDACTED], Timeout: %v, Concurrent: %d}",
		c.RegistryType, c.RegistryURL, c.RegistryUser, c.Timeout, c.Concurrent)
}

// MirrorRequest represents a single mirror operation request.
type MirrorRequest struct {
	// Source URL of the artifact to mirror
	Source string

	// Destination repository name in JFrog Artifactory
	Destination string

	// Optional: Properties to attach to the artifact
	Properties map[string]string

	// Optional: Whether to preserve the original URL structure (default: false)
	Raw bool

	// Optional: Whether to extract single files from zip archives (default: false)
	Unzip bool

	// Optional: Local file path to upload instead of downloading from source
	// The source URL is still used for coordinate extraction
	// TODO: This feature is not yet implemented
	FromFile string

	// Optional: Dry run mode - "all" skips everything, "upload" skips only upload
	DryRun     bool
	DryRunMode string

	// Optional: Source path prefix to strip from source URLs before processing.
	// This is useful for JFrog-to-JFrog mirroring where you want to remove the source
	// repository path. For example, setting this to "artifactory.corp.net/staging/"
	// will strip that prefix from source URLs before generating the destination path.
	// This enables clean mirroring from staging to production repositories.
	SourcePathStrip string
}

// MirrorResult represents the result of a mirror operation.
type MirrorResult struct {
	// Source URL that was mirrored
	Source string

	// Destination path where the artifact was stored
	DestinationPath string

	// Error if the operation failed (nil on success)
	Error error
}

// Client provides the main interface for mirroring artifacts.
type Client struct {
	Config       Config
	logger       *logrus.Logger
	mirror       *mirror.DefaultMirror
	destinations map[string]core.Destination
	mu           sync.RWMutex // Protects destinations map
}

// NewClient creates a new garf client with the provided configuration.
func NewClient(config Config) (*Client, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.Logger == nil {
		config.Logger = logrus.New()
		config.Logger.SetLevel(logrus.InfoLevel)
	}

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	if config.Concurrent == 0 {
		config.Concurrent = DefaultConcurrent
	}

	// Create mirror instance
	mirrorInstance := mirror.NewDefaultMirror(config.Logger)

	// Setup both GitHub and generic sources
	githubSource := sources.NewGitHubSource(config.Logger)
	if err := mirrorInstance.AddSource(core.SourceTypeGitHub, githubSource); err != nil {
		return nil, fmt.Errorf("failed to add GitHub source: %w", err)
	}

	genericSource := sources.NewGenericSource(config.Logger)
	if err := mirrorInstance.AddSource(core.SourceTypeGeneric, genericSource); err != nil {
		return nil, fmt.Errorf("failed to add generic source: %w", err)
	}

	client := &Client{
		Config:       config,
		logger:       config.Logger,
		mirror:       mirrorInstance,
		destinations: make(map[string]core.Destination),
	}

	return client, nil
}

// Mirror performs a single mirror operation.
func (c *Client) Mirror(ctx context.Context, request MirrorRequest) (*MirrorResult, error) {
	logger := c.logger.WithFields(logrus.Fields{
		"source":      request.Source,
		"destination": request.Destination,
		"dry_run":     request.DryRun,
	})

	logger.Info("Starting mirror operation")

	if err := c.ValidateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create context with timeout if not already set or if our timeout is shorter
	if c.Config.Timeout > 0 {
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > c.Config.Timeout {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.Config.Timeout)
			defer cancel()
		}
	}

	// Detect source type and ensure the appropriate source is available
	sourceType := c.DetectSourceType(request.Source, request.SourcePathStrip)
	if err := c.EnsureSourceAvailable(sourceType); err != nil {
		return nil, fmt.Errorf("failed to setup source: %w", err)
	}

	// Create destination key for caching (includes source path strip config)
	destKey := request.Destination
	if request.SourcePathStrip != "" {
		destKey = fmt.Sprintf("%s|strip:%s", request.Destination, request.SourcePathStrip)
	}

	// Get or create destination with proper locking
	c.mu.Lock()

	dest, exists := c.destinations[destKey]
	if !exists {
		// Create new destination
		var err error

		dest, err = c.createDestination(request.Destination, request.SourcePathStrip)
		if err != nil {
			c.mu.Unlock()
			logger.WithError(err).Error("Failed to create destination")

			return nil, fmt.Errorf("failed to create destination: %w", err)
		}

		c.destinations[destKey] = dest
	}
	c.mu.Unlock()

	// Register destination with mirror (outside of lock to minimize lock duration)
	if !exists {
		if err := c.mirror.AddDestination(destKey, dest); err != nil {
			// If registration fails, remove from cache
			c.mu.Lock()
			delete(c.destinations, destKey)
			c.mu.Unlock()
			logger.WithError(err).Error("Failed to register destination with mirror")

			return nil, fmt.Errorf("failed to add destination: %w", err)
		}
	}

	// Create artifact
	artifact := &core.Artifact{
		Name:     ExtractArtifactName(request.Source),
		Location: request.Source,
		Metadata: request.Properties,
	}

	// Create mirror options
	opts := &core.MirrorOptions{
		Context:    ctx,
		Raw:        request.Raw,
		Concurrent: c.Config.Concurrent,
		DryRun:     request.DryRun,
		DryRunMode: request.DryRunMode,
		Unzip:      request.Unzip,
	}

	// Perform mirror operation
	results := c.mirror.Mirror(ctx, []*core.Artifact{artifact}, opts)

	// Wait for result
	select {
	case result, ok := <-results:
		if !ok {
			return nil, fmt.Errorf("mirror operation completed without result")
		}

		// If there was an error during mirroring, return it
		if result.Error != nil {
			return &MirrorResult{
				Source:          request.Source,
				DestinationPath: result.DestinationPath,
				Error:           result.Error,
			}, nil
		}

		// If unzip is enabled, process the artifact
		if request.Unzip {
			if err := processor.ProcessArtifact(ctx, c.logger, c.mirror, result.Artifact, opts); err != nil {
				return &MirrorResult{
					Source:          request.Source,
					DestinationPath: result.DestinationPath,
					Error:           fmt.Errorf("failed to process artifact: %w", err),
				}, nil
			}
		}

		return &MirrorResult{
			Source:          request.Source,
			DestinationPath: result.DestinationPath,
			Error:           result.Error,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ValidateConfig validates the client configuration.
func ValidateConfig(config Config) error {
	// Validate registry type if specified
	if config.RegistryType != "" &&
		config.RegistryType != core.RegistryTypeJFrog &&
		config.RegistryType != core.RegistryTypeCloudsmith {
		return fmt.Errorf("unsupported registry type: %s", config.RegistryType)
	}

	// Check for registry credentials (prefer new generic fields, fallback to legacy)
	url := config.RegistryURL
	if url == "" {
		url = config.JFrogURL // Backward compatibility
	}

	if url == "" {
		return fmt.Errorf("registry URL is required (use RegistryURL or JFrogURL)")
	}

	user := config.RegistryUser
	if user == "" {
		user = config.JFrogUser // Backward compatibility
	}

	if user == "" {
		return fmt.Errorf("registry user is required (use RegistryUser or JFrogUser)")
	}

	password := config.RegistryPassword
	if password == "" {
		password = config.JFrogPassword // Backward compatibility
	}

	if password == "" {
		return fmt.Errorf("registry password is required (use RegistryPassword or JFrogPassword)")
	}

	if config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	if config.Concurrent < 0 {
		return fmt.Errorf("concurrent cannot be negative")
	}

	return nil
}

// ValidateRequest validates a mirror request.
func (c *Client) ValidateRequest(request MirrorRequest) error {
	if request.Source == "" {
		return fmt.Errorf("source is required")
	}

	if request.Destination == "" {
		return fmt.Errorf("destination is required")
	}

	// Validate DryRunMode if specified, regardless of DryRun flag
	if err := core.ValidateDryRunMode(request.DryRunMode); err != nil {
		return err
	}

	// Validate SourcePathStrip if specified
	if err := core.ValidateSourcePathStrip(request.SourcePathStrip); err != nil {
		return err
	}

	return nil
}

// ExtractArtifactName extracts the artifact name from a URL.
func ExtractArtifactName(urlStr string) string {
	// Handle empty URL
	if urlStr == "" {
		return UnknownArtifactName
	}

	// Parse the URL to handle edge cases properly
	u, err := url.Parse(urlStr)
	if err != nil {
		return UnknownArtifactName
	}

	// Extract the base name from the path
	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return UnknownArtifactName
	}

	return name
}

// GetCachedDestinationsCount returns the number of cached destinations (for testing).
func (c *Client) GetCachedDestinationsCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.destinations)
}

// IsCachedDestination checks if a destination is cached (for testing).
func (c *Client) IsCachedDestination(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.destinations[key]

	return exists
}

// createDestination creates a destination based on the registry type with the given configuration.
func (c *Client) createDestination(destPath, sourcePathStrip string) (core.Destination, error) {
	// Validate sourcePathStrip parameter using centralized validation
	if err := core.ValidateSourcePathStrip(sourcePathStrip); err != nil {
		return nil, err
	}

	// Default to JFrog for backward compatibility
	registryType := c.Config.RegistryType
	if registryType == "" {
		registryType = core.RegistryTypeJFrog
	}

	switch registryType {
	case core.RegistryTypeJFrog:
		return c.createJFrogDestination(destPath, sourcePathStrip)
	case core.RegistryTypeCloudsmith:
		return c.createCloudsmithDestination(destPath, sourcePathStrip)
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", registryType)
	}
}

// createJFrogDestination creates a new JFrog destination with the given configuration.
func (c *Client) createJFrogDestination(destPath, sourcePathStrip string) (core.Destination, error) {
	// Support both new Registry* fields and legacy JFrog* fields for backward compatibility
	url := c.Config.RegistryURL
	if url == "" {
		url = c.Config.JFrogURL // Backward compatibility
	}

	user := c.Config.RegistryUser
	if user == "" {
		user = c.Config.JFrogUser // Backward compatibility
	}

	password := c.Config.RegistryPassword
	if password == "" {
		password = c.Config.JFrogPassword // Backward compatibility
	}

	jfrogConfig := destinations.JFrogConfig{
		URL:             url,
		User:            user,
		Password:        password,
		DestPath:        destPath,
		SourcePathStrip: sourcePathStrip,
	}

	return destinations.NewJFrogDestination(jfrogConfig, c.logger), nil
}

// createCloudsmithDestination creates a new Cloudsmith destination with the given configuration.
func (c *Client) createCloudsmithDestination(destPath, sourcePathStrip string) (core.Destination, error) {
	// Support both new Registry* fields and legacy JFrog* fields for backward compatibility
	url := c.Config.RegistryURL
	if url == "" {
		url = c.Config.JFrogURL // Backward compatibility fallback
	}

	user := c.Config.RegistryUser
	if user == "" {
		user = c.Config.JFrogUser // Backward compatibility fallback
	}

	password := c.Config.RegistryPassword
	if password == "" {
		password = c.Config.JFrogPassword // Backward compatibility fallback
	}

	cloudsmithConfig := destinations.CloudsmithConfig{
		URL:             url,
		User:            user,
		Password:        password,
		DestPath:        destPath,
		SourcePathStrip: sourcePathStrip,
	}

	return destinations.NewCloudsmithDestination(cloudsmithConfig, c.logger), nil
}

// DetectSourceType determines the source type based on the URL.
// When SourcePathStrip is provided, it strips the prefix first to determine the actual source.
func (c *Client) DetectSourceType(sourceURL, sourcePathStrip string) string {
	return core.DetectSourceType(sourceURL, sourcePathStrip)
}

// EnsureSourceAvailable ensures that the appropriate source is available in the mirror.
func (c *Client) EnsureSourceAvailable(sourceType string) error {
	// Check if the source is already available
	_, err := c.mirror.GetSource(sourceType)
	if err == nil {
		// Source is already available
		return nil
	}

	// Source not found, this shouldn't happen since we set up both sources in NewClient
	return fmt.Errorf("source type %s is not available", sourceType)
}
