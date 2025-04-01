package urlprocessor

import (
	"net/url"
	"path"
	"strings"
)

// GitHubProcessor handles GitHub release URLs and creates structured paths.
type GitHubProcessor struct{}

// CanProcess checks if the URL is from GitHub.
func (p *GitHubProcessor) CanProcess(sourceURL *url.URL) bool {
	return strings.Contains(sourceURL.Host, "github.com")
}

// If raw=false, it creates a cleaner structure like github.com/owner/repo/version/filename.
func (p *GitHubProcessor) Process(sourceURL *url.URL, raw bool) string {
	// Extract filename
	filename := path.Base(sourceURL.Path)

	// Parse the GitHub path components
	pathParts := strings.Split(strings.TrimPrefix(sourceURL.Path, "/"), "/")

	// Check if this is a GitHub release URL
	if len(pathParts) >= 5 && pathParts[2] == "releases" && pathParts[3] == "download" {
		if raw {
			// Raw mode: Keep the full GitHub path structure
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

	// For other GitHub URLs, use the filename
	return filename
}
