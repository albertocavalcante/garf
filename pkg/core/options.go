package core

import (
	"context"
	"fmt"
)

// MirrorOptions configures how artifacts are mirrored.
type MirrorOptions struct {
	// PreserveStructure determines if the original directory structure should be maintained
	PreserveStructure bool

	// VerifyChecksum enables checksum verification during mirroring
	VerifyChecksum bool

	// Concurrent specifies the number of concurrent mirror operations
	// If set to 0, a sensible default will be used
	Concurrent int

	// ProgressFunc is called to report progress during mirroring
	ProgressFunc func(current, total int64)

	// Context allows for cancellation and timeouts
	Context context.Context
}

// Validate validates the mirror options.
func (o *MirrorOptions) Validate() error {
	if o.Context == nil {
		return fmt.Errorf("context is required")
	}

	if o.Concurrent <= 0 {
		return fmt.Errorf("concurrent must be greater than 0")
	}

	return nil
}
