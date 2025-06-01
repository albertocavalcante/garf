package urlprocessor

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

// GitHubProcessor handles GitHub release URLs and creates structured paths.
type GitHubProcessor struct {
	logger *logrus.Logger
}

// NewGitHubProcessor creates a new GitHubProcessor with optional logger.
func NewGitHubProcessor(logger *logrus.Logger) *GitHubProcessor {
	if logger == nil {
		logger = logrus.New()
	}

	return &GitHubProcessor{logger: logger}
}

// CanProcess checks if the URL is from GitHub.
func (p *GitHubProcessor) CanProcess(sourceURL *url.URL) bool {
	isGitHub, _ := core.IsGitHubURL(sourceURL.String())

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"url":       sourceURL.String(),
			"is_github": isGitHub,
		}).Debug("Checking if URL can be processed by GitHub processor")
	}

	return isGitHub
}

// Process transforms a GitHub URL into a structured path.
// If raw=true, it preserves the complete URL structure.
// If raw=false, it creates a cleaner structure like github.com/owner/repo/version/filename.
func (p *GitHubProcessor) Process(sourceURL *url.URL, raw bool) string {
	logger := p.logger.WithFields(logrus.Fields{
		"url":      sourceURL.String(),
		"raw_mode": raw,
	})

	logger.Debug("Processing GitHub URL")

	// Extract filename
	filename := path.Base(sourceURL.Path)
	logger.WithField("filename", filename).Debug("Extracted filename from URL")

	// Parse the GitHub path components
	pathParts := strings.Split(strings.TrimPrefix(sourceURL.Path, "/"), "/")
	logger.WithFields(logrus.Fields{
		"path_parts":       pathParts,
		"path_parts_count": len(pathParts),
	}).Debug("Split URL path into components")

	// Check if this is a standard GitHub release URL (owner/repo/releases/download/version/filename)
	if len(pathParts) >= 6 && pathParts[2] == "releases" && pathParts[3] == "download" {
		logger.Debug("Detected standard GitHub release URL format")

		owner := pathParts[0]
		repo := pathParts[1]
		version := pathParts[4]

		logger.WithFields(logrus.Fields{
			"owner":    owner,
			"repo":     repo,
			"version":  version,
			"filename": filename,
		}).Debug("Extracted GitHub release components from standard URL")

		if raw {
			result := fmt.Sprintf("github.com/%s/%s/releases/download/%s/%s", owner, repo, version, filename)
			logger.WithField("result_path", result).Info("Built raw GitHub path from standard release URL")

			return result
		} else {
			result := fmt.Sprintf("github.com/%s/%s/%s/%s", owner, repo, version, filename)
			logger.WithField("result_path", result).Info("Built structured GitHub path from standard release URL")

			return result
		}
	}

	// Check if this is a stripped GitHub URL with coordinates (owner/repo/version/filename)
	// This handles URLs that have been processed by source path stripping
	if len(pathParts) >= 3 {
		logger.WithFields(logrus.Fields{
			"path_parts":       pathParts,
			"path_parts_count": len(pathParts),
		}).Debug("Checking for stripped GitHub URL pattern")

		// Try to detect if this looks like owner/repo/version/filename pattern
		// We can identify this by checking if the third component looks like a version
		if len(pathParts) >= 4 {
			owner := pathParts[0]
			repo := pathParts[1]
			potentialVersion := pathParts[2]

			// Check if the third component looks like a version (contains digits or dots or starts with 'v')
			isVersion := strings.Contains(potentialVersion, ".") ||
				strings.HasPrefix(potentialVersion, "v") ||
				strings.ContainsAny(potentialVersion, "0123456789")

			if isVersion {
				logger.WithFields(logrus.Fields{
					"owner":             owner,
					"repo":              repo,
					"potential_version": potentialVersion,
					"filename":          filename,
				}).Debug("Detected stripped GitHub URL with version pattern")

				result := fmt.Sprintf("github.com/%s/%s/%s/%s", owner, repo, potentialVersion, filename)
				logger.WithField("result_path", result).Info("Built structured GitHub path from stripped URL coordinates")

				return result
			}
		}

		// If we have at least 3 parts but it doesn't look like a version pattern,
		// treat it as owner/repo/filename (fallback for non-standard patterns)
		if len(pathParts) >= 3 {
			owner := pathParts[0]
			repo := pathParts[1]

			result := fmt.Sprintf("github.com/%s/%s/%s", owner, repo, filename)
			logger.WithField("result_path", result).Info("Built structured GitHub path from owner/repo pattern")

			return result
		}
	}

	// Fallback: if we can't parse the structure, just return the filename
	logger.WithField("result_path", filename).Info("Using filename as fallback for unrecognized GitHub URL pattern")

	return filename
}
