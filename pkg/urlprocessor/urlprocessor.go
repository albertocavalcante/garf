// Package urlprocessor provides URL processing capabilities for transforming
// repository URLs into structured paths for artifact storage.
package urlprocessor

import (
	"net/url"
	"path"

	"github.com/sirupsen/logrus"
)

// PathBuilder transforms URLs into structured storage paths.
// It provides both raw path preservation and cleaner structured paths.
type PathBuilder struct {
	processors []processor
	logger     *logrus.Logger
}

// processor is an internal type for handling different URL formats.
type processor struct {
	canHandle func(*url.URL) bool
	process   func(*url.URL, bool) string
}

// New creates a new PathBuilder with default processors.
func New() *PathBuilder {
	return NewWithLogger(nil)
}

// NewWithLogger creates a new PathBuilder with a logger and default processors.
func NewWithLogger(logger *logrus.Logger) *PathBuilder {
	if logger == nil {
		logger = logrus.New()
	}

	githubProc := NewGitHubProcessor(logger)

	return &PathBuilder{
		logger: logger,
		processors: []processor{
			// GitHub URLs processor
			{
				canHandle: githubProc.CanProcess,
				process:   githubProc.Process,
			},
			// Default fallback processor for non-GitHub URLs
			// Always preserves full URL structure (host + path)
			{
				canHandle: func(u *url.URL) bool { return true },
				process: func(u *url.URL, raw bool) string {
					// The raw flag doesn't affect non-GitHub URLs - they always preserve structure
					// This ensures that URLs like bcr.bazel.build/modules/... maintain their path structure
					return u.Host + u.Path
				},
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
	p.logger.WithFields(logrus.Fields{
		"url":      sourceURL.String(),
		"raw_mode": raw,
	}).Debug("Processing URL with PathBuilder")

	for i, proc := range p.processors {
		if proc.canHandle(sourceURL) {
			result := proc.process(sourceURL, raw)
			p.logger.WithFields(logrus.Fields{
				"processor_index": i,
				"result_path":     result,
			}).Debug("URL processed successfully")

			return result
		}
	}

	fallback := path.Base(sourceURL.Path)
	p.logger.WithField("fallback_path", fallback).Debug("Using fallback path processing")

	return fallback // Fallback in case processors list is empty
}

// ProcessURLString parses a URL string and processes it.
func (p *PathBuilder) ProcessURLString(urlStr string, raw bool) (string, error) {
	sourceURL, err := url.Parse(urlStr)
	if err != nil {
		p.logger.WithError(err).WithField("url_string", urlStr).Error("Failed to parse URL string")

		return "", err
	}

	return p.ProcessURL(sourceURL, raw), nil
}
