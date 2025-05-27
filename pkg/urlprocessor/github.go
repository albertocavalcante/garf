package urlprocessor

import (
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
	logger.WithField("path_parts", pathParts).Debug("Split URL path into components")

	// Check if this is a GitHub release URL
	if len(pathParts) >= 5 && pathParts[2] == "releases" && pathParts[3] == "download" {
		logger.Debug("Detected GitHub release URL format")

		if raw {
			// Raw mode: Keep the full GitHub path structure
			result := path.Join(core.GitHubHost, strings.TrimPrefix(sourceURL.Path, "/"))
			logger.WithField("result_path", result).Info("Built raw GitHub path")

			return result
		} else {
			// Clean mode: Create a structured path
			owner := pathParts[0]
			repo := pathParts[1]
			version := pathParts[4]

			logger.WithFields(logrus.Fields{
				"owner":    owner,
				"repo":     repo,
				"version":  version,
				"filename": filename,
			}).Debug("Extracted GitHub release components")

			// Build path: github.com/owner/repo/version/filename
			result := path.Join(core.GitHubHost, owner, repo, version, filename)
			logger.WithField("result_path", result).Info("Built structured GitHub path")

			return result
		}
	}

	// For other GitHub URLs, use the filename
	logger.Debug("Not a GitHub release URL, using filename only")
	logger.WithField("result_path", filename).Info("Using filename as path")

	return filename
}
