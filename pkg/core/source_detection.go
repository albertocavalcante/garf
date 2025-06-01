// Package core provides source type detection functionality.
package core

import (
	"strings"
)

// DetectSourceType determines the source type based on the URL.
// When sourcePathStrip is provided, it strips the prefix first to determine the actual source.
func DetectSourceType(sourceURL, sourcePathStrip string) string {
	// If source path strip is provided, apply it first to get the actual source URL
	urlToCheck := sourceURL

	if sourcePathStrip != "" {
		// Strip the prefix if it exists in the URL
		if strings.Contains(sourceURL, sourcePathStrip) {
			// Find the position after the strip prefix
			if idx := strings.Index(sourceURL, sourcePathStrip); idx != -1 {
				urlToCheck = sourceURL[idx+len(sourcePathStrip):]
				// Ensure it starts with a scheme
				if !strings.HasPrefix(urlToCheck, "http://") && !strings.HasPrefix(urlToCheck, "https://") {
					urlToCheck = "https://" + urlToCheck
				}
			}
		}
	}

	// Check if it's a GitHub URL
	if isGitHub, _ := IsGitHubURL(urlToCheck); isGitHub {
		return SourceTypeGitHub
	}

	// For non-GitHub URLs, use generic source type
	return SourceTypeGeneric
}
