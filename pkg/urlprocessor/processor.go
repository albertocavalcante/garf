// Package urlprocessor provides interfaces and implementations for processing URLs
// to create structured paths for artifacts.
package urlprocessor

import (
	"net/url"
	"path"
)

// Processor defines the interface for URL processors that can recognize and transform URLs
// into structured paths for storage.
type Processor interface {
	// CanProcess checks if this processor can handle the given URL
	CanProcess(sourceURL *url.URL) bool

	// Process transforms a source URL into a structured path
	// If raw is true, it preserves the complete URL structure
	// If raw is false, it creates a clean, structured path
	Process(sourceURL *url.URL, raw bool) string
}

// DefaultProcessor is a fallback processor that simply extracts the filename.
type DefaultProcessor struct{}

// CanProcess returns true for any URL, as this is the fallback processor.
func (p *DefaultProcessor) CanProcess(sourceURL *url.URL) bool {
	return true
}

// Process simply returns the base filename of the URL path.
func (p *DefaultProcessor) Process(sourceURL *url.URL, raw bool) string {
	return path.Base(sourceURL.Path)
}
