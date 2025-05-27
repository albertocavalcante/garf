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
package garf

import (
	"context"
	"fmt"
	"net/url"
	"path"
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
	// JFrog Artifactory configuration
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
	return fmt.Sprintf("Config{JFrogURL: %s, JFrogUser: %s, JFrogPassword: [REDACTED], Timeout: %v, Concurrent: %d}",
		c.JFrogURL, c.JFrogUser, c.Timeout, c.Concurrent)
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
	Config Config
	logger *logrus.Logger
	mirror *mirror.DefaultMirror
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

	// Setup source once during client creation
	githubSource := sources.NewGitHubSource(config.Logger)
	if err := mirrorInstance.AddSource("github", githubSource); err != nil {
		return nil, fmt.Errorf("failed to add source: %w", err)
	}

	client := &Client{
		Config: config,
		logger: config.Logger,
		mirror: mirrorInstance,
	}

	return client, nil
}

// Mirror performs a single mirror operation.
func (c *Client) Mirror(ctx context.Context, request MirrorRequest) (*MirrorResult, error) {
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

	// Setup destination (needs to be per-request since destination path varies)
	jfrogConfig := destinations.JFrogConfig{
		URL:      c.Config.JFrogURL,
		User:     c.Config.JFrogUser,
		Password: c.Config.JFrogPassword,
		DestPath: request.Destination,
	}

	jfrogDest := destinations.NewJFrogDestination(jfrogConfig, c.logger)
	// Use a unique destination name for each request to avoid conflicts
	destName := fmt.Sprintf("jfrog-%d", time.Now().UnixNano())
	if err := c.mirror.AddDestination(destName, jfrogDest); err != nil {
		return nil, fmt.Errorf("failed to add destination: %w", err)
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
	if config.JFrogURL == "" {
		return fmt.Errorf("JFrogURL is required")
	}

	if config.JFrogUser == "" {
		return fmt.Errorf("JFrogUser is required")
	}

	if config.JFrogPassword == "" {
		return fmt.Errorf("JFrogPassword is required")
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
	if request.DryRunMode != "" {
		validModes := map[string]bool{"all": true, "upload": true}
		if !validModes[request.DryRunMode] {
			return fmt.Errorf("invalid dry run mode: %s. Valid modes are: all, upload", request.DryRunMode)
		}
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
