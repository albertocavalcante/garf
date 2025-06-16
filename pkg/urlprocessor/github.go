package urlprocessor

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

// Constants for GitHub URL parsing to avoid magic numbers.
const (
	minGitHubStandardURLParts = 6 // For owner/repo/releases/download/version/filename
	minGitHubStrippedParts    = 3 // For owner/repo/filename
	minGitHubVersionParts     = 4 // For owner/repo/version/filename
	gitHubOwnerIndex          = 0 // Index of owner in path parts
	gitHubRepoIndex           = 1 // Index of repo in path parts
	gitHubVersionIndex        = 2 // Index of version in stripped URLs
	gitHubReleasesIndex       = 2 // Index of "releases" in standard URLs
	gitHubDownloadIndex       = 3 // Index of "download" in standard URLs
	gitHubReleaseVersionIndex = 4 // Index of version in standard URLs
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
	logger := p.createLogger(sourceURL, raw)
	filename := path.Base(sourceURL.Path)
	pathParts := p.parsePathComponents(sourceURL, logger)

	logger.WithField("filename", filename).Debug("Extracted filename from URL")

	// Try different URL patterns in order of specificity
	if result, ok := p.tryStandardReleaseURL(pathParts, filename, raw, logger); ok {
		return result
	}

	if result, ok := p.tryStrippedVersionURL(pathParts, filename, logger); ok {
		return result
	}

	if result, ok := p.tryOwnerRepoURL(pathParts, filename, logger); ok {
		return result
	}

	// Fallback to filename
	logger.WithField("result_path", filename).Info("Using filename as fallback for unrecognized GitHub URL pattern")

	return filename
}

// createLogger creates a logger with URL processing context.
func (p *GitHubProcessor) createLogger(sourceURL *url.URL, raw bool) *logrus.Entry {
	return p.logger.WithFields(logrus.Fields{
		"url":      sourceURL.String(),
		"raw_mode": raw,
	})
}

// parsePathComponents splits the URL path into components.
func (p *GitHubProcessor) parsePathComponents(sourceURL *url.URL, logger *logrus.Entry) []string {
	logger.Debug("Processing GitHub URL")

	pathParts := strings.Split(strings.TrimPrefix(sourceURL.Path, "/"), "/")
	logger.WithFields(logrus.Fields{
		"path_parts":       pathParts,
		"path_parts_count": len(pathParts),
	}).Debug("Split URL path into components")

	return pathParts
}

// tryStandardReleaseURL attempts to parse as owner/repo/releases/download/version/filename.
func (p *GitHubProcessor) tryStandardReleaseURL(pathParts []string, filename string, raw bool, logger *logrus.Entry) (string, bool) {
	if len(pathParts) < minGitHubStandardURLParts ||
		pathParts[gitHubReleasesIndex] != "releases" ||
		pathParts[gitHubDownloadIndex] != "download" {
		return "", false
	}

	logger.Debug("Detected standard GitHub release URL format")

	owner := pathParts[gitHubOwnerIndex]
	repo := pathParts[gitHubRepoIndex]
	version := pathParts[gitHubReleaseVersionIndex]

	logger.WithFields(logrus.Fields{
		"owner":    owner,
		"repo":     repo,
		"version":  version,
		"filename": filename,
	}).Debug("Extracted GitHub release components from standard URL")

	if raw {
		result := fmt.Sprintf("github.com/%s/%s/releases/download/%s/%s", owner, repo, version, filename)
		logger.WithField("result_path", result).Info("Built raw GitHub path from standard release URL")

		return result, true
	}

	result := fmt.Sprintf("github.com/%s/%s/%s/%s", owner, repo, version, filename)
	logger.WithField("result_path", result).Info("Built structured GitHub path from standard release URL")

	return result, true
}

// tryStrippedVersionURL attempts to parse as owner/repo/version/filename.
func (p *GitHubProcessor) tryStrippedVersionURL(pathParts []string, filename string, logger *logrus.Entry) (string, bool) {
	if len(pathParts) < minGitHubVersionParts {
		return "", false
	}

	logger.WithFields(logrus.Fields{
		"path_parts":       pathParts,
		"path_parts_count": len(pathParts),
	}).Debug("Checking for stripped GitHub URL pattern")

	owner := pathParts[gitHubOwnerIndex]
	repo := pathParts[gitHubRepoIndex]
	potentialVersion := pathParts[gitHubVersionIndex]

	if !p.looksLikeVersion(potentialVersion) {
		return "", false
	}

	logger.WithFields(logrus.Fields{
		"owner":             owner,
		"repo":              repo,
		"potential_version": potentialVersion,
		"filename":          filename,
	}).Debug("Detected stripped GitHub URL with version pattern")

	result := fmt.Sprintf("github.com/%s/%s/%s/%s", owner, repo, potentialVersion, filename)
	logger.WithField("result_path", result).Info("Built structured GitHub path from stripped URL coordinates")

	return result, true
}

// tryOwnerRepoURL attempts to parse as owner/repo/filename.
func (p *GitHubProcessor) tryOwnerRepoURL(pathParts []string, filename string, logger *logrus.Entry) (string, bool) {
	if len(pathParts) < minGitHubStrippedParts {
		return "", false
	}

	owner := pathParts[gitHubOwnerIndex]
	repo := pathParts[gitHubRepoIndex]

	result := fmt.Sprintf("github.com/%s/%s/%s", owner, repo, filename)
	logger.WithField("result_path", result).Info("Built structured GitHub path from owner/repo pattern")

	return result, true
}

// looksLikeVersion checks if a string looks like a version identifier.
func (p *GitHubProcessor) looksLikeVersion(s string) bool {
	return strings.Contains(s, ".") ||
		strings.HasPrefix(s, "v") ||
		strings.ContainsAny(s, "0123456789")
}
