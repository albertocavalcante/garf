// Package urlprocessor provides URL processing capabilities for transforming
// repository URLs into structured paths for artifact storage.
package urlprocessor

import (
	"net/url"
	"path"
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
	githubProc := &GitHubProcessor{}

	return &PathBuilder{
		processors: []processor{
			// GitHub URLs processor
			{
				canHandle: githubProc.CanProcess,
				process:   githubProc.Process,
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
