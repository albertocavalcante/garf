package core

import (
	"context"
	"fmt"
	"io"
)

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

// Artifact represents a single artifact that can be mirrored.
type Artifact struct {
	// Name is the unique identifier for the artifact
	Name string

	// Version represents the artifact version
	Version string

	// Location is the source location of the artifact
	Location string

	// Metadata contains additional artifact information
	Metadata map[string]string
}

// Validate validates the artifact fields.
func (a *Artifact) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("name is required")
	}

	if a.Location == "" {
		return fmt.Errorf("location is required")
	}

	return nil
}
