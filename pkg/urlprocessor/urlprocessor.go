// Package urlprocessor provides URL processing capabilities for transforming
// repository URLs into structured paths for artifact storage.
package urlprocessor

import (
	"net/url"
	"path"
	"strings"

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

// StripPrefixFromURL removes the configured prefix from the given URL's Host + Path.
// It returns a new URL instance representing the content after stripping, or the original URL if no stripping occurs.
func (p *PathBuilder) StripPrefixFromURL(u *url.URL, prefix string) (*url.URL, error) {
	if prefix == "" {
		return u, nil
	}

	baseString := p.determineBaseForStripping(u, prefix)
	strippedSegment := strings.TrimPrefix(baseString, prefix)

	if strippedSegment == baseString {
		// Try alternative approach if first attempt failed
		if altResult := p.tryAlternativeStripping(u, prefix, baseString); altResult != nil {
			return altResult, nil
		}
		p.logStrippingFailure(u, prefix, baseString)
		return u, nil // Return original URL
	}

	p.logSuccessfulStripping(baseString, prefix, strippedSegment)
	return p.parseStrippedSegment(strippedSegment)
}

// BuildStructuredPath creates the structured path component for an artifact with optional prefix stripping.
func (p *PathBuilder) BuildStructuredPath(sourceURL *url.URL, artifactName, sourcePathStrip string, raw bool) (string, error) {
	processedURL := sourceURL

	if sourcePathStrip != "" {
		strippedURL, err := p.StripPrefixFromURL(sourceURL, sourcePathStrip)
		if err != nil {
			return "", err
		}
		processedURL = strippedURL
	}

	// Get structured path from URL processor
	structuredPath := p.ProcessURL(processedURL, raw)

	// Ensure artifact name is the final component
	baseDirOfStructuredPath := path.Dir(structuredPath)
	if strings.HasSuffix(structuredPath, "/") {
		baseDirOfStructuredPath = structuredPath
	}

	return path.Join(baseDirOfStructuredPath, artifactName), nil
}

// determineBaseForStripping decides whether to strip from Host+Path or Path only.
func (p *PathBuilder) determineBaseForStripping(u *url.URL, prefix string) string {
	if strings.HasPrefix(prefix, u.Host) && u.Host != "" {
		baseString := u.Host + u.Path
		p.logger.Debugf("SourcePathStrip '%s' seems to include host '%s'. Stripping from Host+Path: '%s'", prefix, u.Host, baseString)
		return baseString
	}

	baseString := strings.TrimPrefix(u.Path, "/")
	p.logger.Debugf("SourcePathStrip '%s' does not seem to include host '%s'. Stripping from Path: '%s'", prefix, u.Host, baseString)
	return baseString
}

// tryAlternativeStripping attempts alternative stripping strategies.
func (p *PathBuilder) tryAlternativeStripping(u *url.URL, prefix, originalBase string) *url.URL {
	// Only try Host+Path if we originally tried Path-only
	if strings.HasPrefix(prefix, u.Host) && u.Host != "" {
		return nil // Already tried Host+Path approach
	}

	altBase := u.Host + u.Path
	altStripped := strings.TrimPrefix(altBase, prefix)
	if altStripped != altBase {
		p.logger.Debugf("Initial path-only strip failed. Successful strip from Host+Path: '%s' -> '%s'", altBase, altStripped)
		if result, err := p.parseStrippedSegment(altStripped); err == nil {
			return result
		}
	}
	return nil
}

// parseStrippedSegment converts a stripped segment back into a URL.
func (p *PathBuilder) parseStrippedSegment(segment string) (*url.URL, error) {
	normalized := strings.TrimPrefix(segment, "/")

	// Try GitHub URL detection first
	if githubURL := p.detectGitHubURL(normalized, segment); githubURL != nil {
		return githubURL, nil
	}

	// Try parsing as absolute URL
	if absoluteURL := p.tryParseAbsolute(normalized, segment); absoluteURL != nil {
		return absoluteURL, nil
	}

	// Try parsing with dummy scheme to detect host
	if hostURL := p.tryParseWithHost(normalized, segment); hostURL != nil {
		return hostURL, nil
	}

	// Default: treat as path-only
	return p.createPathOnlyURL(normalized, segment), nil
}

// detectGitHubURL checks if the segment contains GitHub content.
func (p *PathBuilder) detectGitHubURL(normalized, original string) *url.URL {
	if idx := strings.Index(normalized, "github.com/"); idx != -1 {
		githubSegment := normalized[idx:]
		pathAfterHost := strings.TrimPrefix(githubSegment, "github.com")

		githubURL := &url.URL{
			Host: "github.com",
			Path: pathAfterHost,
		}
		if !strings.HasPrefix(githubURL.Path, "/") && githubURL.Path != "" {
			githubURL.Path = "/" + githubURL.Path
		}

		p.logger.Debugf("Stripped segment '%s' (normalized: '%s') identified as GitHub content. New URL: %s", original, normalized, githubURL.String())
		return githubURL
	}
	return nil
}

// tryParseAbsolute attempts to parse the segment as an absolute URL.
func (p *PathBuilder) tryParseAbsolute(normalized, original string) *url.URL {
	if newU, err := url.Parse(normalized); err == nil && newU.IsAbs() {
		p.logger.Debugf("Stripped segment '%s' (normalized: '%s') parsed as absolute URL: %s", original, normalized, newU.String())
		return newU
	}
	return nil
}

// tryParseWithHost attempts to parse with a dummy scheme to identify host part.
func (p *PathBuilder) tryParseWithHost(normalized, original string) *url.URL {
	if newU, err := url.Parse("dummy://" + normalized); err == nil && newU.Host != "" {
		schemalessURL := &url.URL{
			Host: newU.Host,
			Path: newU.Path,
		}
		p.logger.Debugf("Stripped segment '%s' (normalized: '%s') parsed as host '%s' with path '%s'. Schemaless URL: %s",
			original, normalized, schemalessURL.Host, schemalessURL.Path, schemalessURL.String())
		return schemalessURL
	}
	return nil
}

// createPathOnlyURL creates a URL with only the path component.
func (p *PathBuilder) createPathOnlyURL(normalized, original string) *url.URL {
	newPath := "/" + normalized
	finalURL := &url.URL{Path: newPath}
	p.logger.Debugf("Stripped segment '%s' (normalized: '%s') treated as path-only. New URL: %s (Path: %s)",
		original, normalized, finalURL.String(), finalURL.Path)
	return finalURL
}

// logStrippingFailure logs when stripping fails.
func (p *PathBuilder) logStrippingFailure(u *url.URL, prefix, baseString string) {
	p.logger.WithFields(logrus.Fields{
		"source_url":         u.String(),
		"strip_prefix":       prefix,
		"base_for_stripping": baseString,
	}).Debug("Source path strip prefix not found in URL components")
}

// logSuccessfulStripping logs successful stripping.
func (p *PathBuilder) logSuccessfulStripping(originalBase, prefix, stripped string) {
	p.logger.WithFields(logrus.Fields{
		"original_base":    originalBase,
		"strip_pattern":    prefix,
		"stripped_segment": stripped,
	}).Debug("Source path stripped")
}
