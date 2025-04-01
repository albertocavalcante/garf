// Package urlprocessor provides URL processing capabilities for transforming
// repository URLs into structured paths for artifact storage.
package urlprocessor

import (
	"net/url"
	"path"
	"strings"
)

// PathBuilder transforms URLs into structured storage paths.
// It provides both raw path preservation and cleaner structured paths.
type PathBuilder struct {
	processors []processor
}

// processor is an internal type for handling different URL formats.
type processor struct {
	canHandle func(*url.URL) bool
	process   func(*url.URL, bool) string
}

// New creates a new PathBuilder with default processors.
func New() *PathBuilder {
	return &PathBuilder{
		processors: []processor{
			// GitHub URLs processor
			{
				canHandle: func(u *url.URL) bool {
					return strings.Contains(u.Host, "github.com")
				},
				process: processGitHubURL,
			},
			// Default fallback processor
			{
				canHandle: func(u *url.URL) bool { return true },
				process:   func(u *url.URL, _ bool) string { return path.Base(u.Path) },
			},
		},
	}
}

// AddProcessor adds a custom URL processor to the builder.
func (p *PathBuilder) AddProcessor(canHandle func(*url.URL) bool, process func(*url.URL, bool) string) {
	// Insert before the default processor (which is always last)
	if len(p.processors) > 0 {
		p.processors = append(p.processors[:len(p.processors)-1],
			processor{canHandle: canHandle, process: process},
			p.processors[len(p.processors)-1])
	} else {
		p.processors = append(p.processors, processor{canHandle: canHandle, process: process})
	}
}

// ProcessURL finds the appropriate processor for the URL and returns the resulting path.
func (p *PathBuilder) ProcessURL(sourceURL *url.URL, raw bool) string {
	for _, proc := range p.processors {
		if proc.canHandle(sourceURL) {
			return proc.process(sourceURL, raw)
		}
	}

	return path.Base(sourceURL.Path) // Fallback in case processors list is empty
}

// ProcessURLString parses a URL string and processes it.
func (p *PathBuilder) ProcessURLString(urlStr string, raw bool) (string, error) {
	sourceURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	return p.ProcessURL(sourceURL, raw), nil
}

// processGitHubURL handles GitHub-specific URL processing.
func processGitHubURL(sourceURL *url.URL, raw bool) string {
	// Extract filename
	filename := path.Base(sourceURL.Path)

	// Parse the GitHub path components
	pathParts := strings.Split(strings.TrimPrefix(sourceURL.Path, "/"), "/")

	// Check if this is a GitHub release URL
	if len(pathParts) >= 5 && pathParts[2] == "releases" && pathParts[3] == "download" {
		if raw {
			// Raw mode: Keep the full GitHub path
			return path.Join("github.com", strings.TrimPrefix(sourceURL.Path, "/"))
		} else {
			// Clean mode: Create a structured path
			owner := pathParts[0]
			repo := pathParts[1]
			version := pathParts[4]

			// Build path: github.com/owner/repo/version/filename
			return path.Join("github.com", owner, repo, version, filename)
		}
	}

	// For other GitHub URLs, use just the filename
	return filename
}
