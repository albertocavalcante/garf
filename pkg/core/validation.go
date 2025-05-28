// Package core provides validation functionality for garf parameters.
package core

import (
	"fmt"
	"strings"
)

// ValidateSourcePathStrip validates the source path strip parameter for security and correctness.
func ValidateSourcePathStrip(sourcePathStrip string) error {
	if sourcePathStrip == "" {
		return nil // Empty is valid
	}

	// Trim whitespace to handle edge cases
	sourcePathStrip = strings.TrimSpace(sourcePathStrip)

	// Check for path traversal attacks
	if strings.Contains(sourcePathStrip, "..") {
		return fmt.Errorf("source path strip cannot contain '..' for security reasons")
	}

	// Ensure it doesn't start with a scheme (should be a path/host component)
	if strings.HasPrefix(sourcePathStrip, "http://") || strings.HasPrefix(sourcePathStrip, "https://") {
		return fmt.Errorf("source path strip should not include the URL scheme (http:// or https://)")
	}

	// Validate against other potentially dangerous patterns
	if strings.Contains(sourcePathStrip, "\\") {
		return fmt.Errorf("source path strip should not contain backslashes")
	}

	return nil
}

// ValidateDryRunMode validates the dry run mode parameter.
func ValidateDryRunMode(dryRunMode string) error {
	if dryRunMode == "" {
		return nil // Empty is valid
	}

	validModes := map[string]bool{"all": true, "upload": true}
	if !validModes[dryRunMode] {
		return fmt.Errorf("invalid dry run mode: %s. Valid modes are: all, upload", dryRunMode)
	}

	return nil
}
