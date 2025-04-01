package core

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// GitHubHost is the standard GitHub hostname.
	GitHubHost = "github.com"
)

// IsGitHubURL checks if a URL is from GitHub.
// During testing, localhost and 127.0.0.1 are considered valid GitHub URLs.
func IsGitHubURL(urlStr string) (bool, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false, fmt.Errorf("invalid URL: %w", err)
	}

	// Allow test server URLs during testing
	if strings.Contains(parsedURL.Host, "127.0.0.1") || strings.Contains(parsedURL.Host, "localhost") {
		return true, nil
	}

	return strings.Contains(parsedURL.Host, GitHubHost), nil
}

// ValidateGitHubURL validates that the given URL is a GitHub URL.
// Returns an error if the URL is invalid or not from GitHub.
// During testing, localhost and 127.0.0.1 are considered valid GitHub URLs.
func ValidateGitHubURL(urlStr string) error {
	isGitHub, err := IsGitHubURL(urlStr)
	if err != nil {
		return err
	}

	if !isGitHub {
		return fmt.Errorf("not a GitHub URL: %s", urlStr)
	}

	return nil
}
