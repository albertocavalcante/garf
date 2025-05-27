package core

import (
	"context"
	"fmt"
)

// MirrorOptions configures how artifacts are mirrored.
type MirrorOptions struct {
	// Context allows for cancellation and timeouts
	Context context.Context

	// Raw keeps the original URL structure instead of creating a cleaner path
	Raw bool

	// Concurrent specifies the number of concurrent mirror operations
	// If set to 0, a sensible default will be used
	Concurrent int

	// ProgressFunc is called to report progress during mirroring
	ProgressFunc func(current, total int64, message string)

	// DryRun indicates if this is a dry run operation
	DryRun bool

	// DryRunMode specifies the dry run mode: "all" or "upload"
	DryRunMode string

	// Unzip indicates if ZIP files should be extracted during mirroring
	Unzip bool

	// PreserveZipName indicates if the ZIP filename should be preserved when extracting,
	// replacing the ZIP extension with the extracted file's extension
	PreserveZipName bool
}

// Validate validates the mirror options.
func (o *MirrorOptions) Validate() error {
	if o.Context == nil {
		return fmt.Errorf("context is required")
	}

	if o.Concurrent <= 0 {
		return fmt.Errorf("concurrent must be greater than 0")
	}

	if o.DryRun {
		validModes := map[string]bool{
			"all":    true,
			"upload": true,
		}

		if !validModes[o.DryRunMode] {
			return fmt.Errorf("invalid dry run mode: %s. Valid modes are: all, upload", o.DryRunMode)
		}
	}

	return nil
}
