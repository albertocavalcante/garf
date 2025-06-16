// Package config provides configuration management functionality.
package config

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/albertocavalcante/garf/pkg/core"
	yaml "gopkg.in/yaml.v3"
)

// Artifact is an alias for core.Artifact.
type Artifact = core.Artifact

// Config represents the configuration for the mirror process.
type Config struct {
	Source      SourceConfig      `yaml:"source"`
	Destination DestinationConfig `yaml:"destination"`
	LogLevel    string            `yaml:"log_level"`
	Concurrent  int               `yaml:"concurrent"`
}

// SourceConfig represents the configuration for a source.
type SourceConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

// DestinationConfig represents the configuration for a destination.
type DestinationConfig struct {
	Type            string `yaml:"type"`
	URL             string `yaml:"url"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	DestPath        string `yaml:"dest_path"`
	SourcePathStrip string `yaml:"source_path_strip"`
}

// Source represents a source from which artifacts can be retrieved.
type Source interface {
	// List returns all available artifacts from the source
	List(ctx context.Context) ([]*Artifact, error)

	// Get retrieves a specific artifact by its coordinates
	Get(ctx context.Context, artifact *Artifact) (io.ReadCloser, error)

	// Validate checks if the source configuration is valid
	Validate() error
}

// Destination represents a location where artifacts can be stored.
type Destination interface {
	// Put stores an artifact in the destination
	Put(ctx context.Context, artifact *Artifact, content io.Reader) error

	// Exists checks if an artifact already exists in the destination
	Exists(ctx context.Context, artifact *Artifact) (bool, error)

	// Validate checks if the destination configuration is valid
	Validate() error
}

// Load loads configuration from a file.
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// validateSource validates the source configuration.
func (c *Config) validateSource() error {
	if c.Source.Type == "" {
		return fmt.Errorf("source type cannot be empty")
	}

	if c.Source.Type != core.SourceTypeGitHub && c.Source.Type != core.SourceTypeGeneric {
		return fmt.Errorf("invalid source type: %s", c.Source.Type)
	}

	if c.Source.URL == "" {
		return fmt.Errorf("source URL cannot be empty")
	}

	if _, err := url.Parse(c.Source.URL); err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	}

	return nil
}

// validateDestination validates the destination configuration.
func (c *Config) validateDestination() error {
	if c.Destination.Type == "" {
		return fmt.Errorf("destination type cannot be empty")
	}

	if c.Destination.Type != "jfrog" && c.Destination.Type != "cloudsmith" {
		return fmt.Errorf("invalid destination type: %s", c.Destination.Type)
	}

	if c.Destination.URL == "" {
		return fmt.Errorf("destination URL cannot be empty")
	}

	if _, err := url.Parse(c.Destination.URL); err != nil {
		return fmt.Errorf("invalid destination URL: %w", err)
	}

	if c.Destination.User == "" {
		return fmt.Errorf("destination user cannot be empty")
	}

	if c.Destination.Password == "" {
		return fmt.Errorf("destination password cannot be empty")
	}

	// Validate SourcePathStrip if specified using centralized validation
	if err := core.ValidateSourcePathStrip(c.Destination.SourcePathStrip); err != nil {
		return err
	}

	return nil
}

// validateLogLevel validates the log level configuration.
func (c *Config) validateLogLevel() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if err := c.validateSource(); err != nil {
		return err
	}

	if err := c.validateDestination(); err != nil {
		return err
	}

	if err := c.validateLogLevel(); err != nil {
		return err
	}

	if c.Concurrent <= 0 {
		return fmt.Errorf("concurrent must be greater than 0")
	}

	return nil
}
