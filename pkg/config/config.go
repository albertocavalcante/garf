package config

import (
	"context"
	"io"

	"github.com/albertocavalcante/garf/pkg/core"
)

// Artifact is an alias for core.Artifact.
type Artifact = core.Artifact

// Config represents the configuration for the mirror process.
type Config struct {
	Source      SourceConfig      `yaml:"source"`
	Destination DestinationConfig `yaml:"destination"`
	LogLevel    string            `yaml:"logLevel"`
	Concurrent  int               `yaml:"concurrent"`
}

// SourceConfig represents the configuration for a source.
type SourceConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

// DestinationConfig represents the configuration for a destination.
type DestinationConfig struct {
	Type     string `yaml:"type"`
	URL      string `yaml:"url"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
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
