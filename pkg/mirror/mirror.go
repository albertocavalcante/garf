// Package mirror provides the core interfaces and types for artifact mirroring.
package mirror

import (
	"context"
	"io"

	"github.com/albertocavalcante/garf/pkg/core"
)

// MirrorResult contains the result of a mirror operation.
type MirrorResult struct {
	// Artifact is the original artifact that was mirrored
	Artifact *core.Artifact

	// DestinationPath is where the artifact was mirrored to
	DestinationPath string

	// Error contains any error that occurred during mirroring
	// If nil, the operation was successful
	Error error
}

// Mirror represents the core mirroring functionality.
type Mirror interface {
	// Mirror copies artifacts from a source to a destination
	Mirror(ctx context.Context, artifacts []*core.Artifact, opts *core.MirrorOptions) <-chan MirrorResult

	// AddSource adds a new source to the mirror
	AddSource(name string, source core.Source) error

	// AddDestination adds a new destination to the mirror
	AddDestination(name string, destination core.Destination) error

	// GetSource retrieves a source by name
	GetSource(name string) (core.Source, error)

	// GetDestination retrieves a destination by name
	GetDestination(name string) (core.Destination, error)
}

// ProgressTracker provides progress tracking for mirror operations.
type ProgressTracker interface {
	// Start initializes the progress tracking
	Start(total int64)

	// Update updates the current progress
	Update(current int64)

	// Complete marks the operation as complete
	Complete()
}

// Validator provides validation functionality for artifacts.
type Validator interface {
	// ValidateArtifact checks if an artifact is valid
	ValidateArtifact(artifact *core.Artifact) error

	// ValidateChecksum verifies the checksum of an artifact
	ValidateChecksum(artifact *core.Artifact, content io.Reader) error
}
