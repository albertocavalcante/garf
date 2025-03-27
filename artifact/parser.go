package artifact

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
)

const (
	// rawPathMatchCount is the expected number of capturing groups plus the whole match in the raw path regex.
	rawPathMatchCount = 2

	// githubReleaseMatchCount is the expected number of capturing groups plus the whole match in GitHub release regex.
	githubReleaseMatchCount = 5
)

// ExtractCoordinatesFromURL extracts the artifact coordinates from the given URL.
func ExtractCoordinatesFromURL(artifactURL string) (*ArtifactCoordinates, error) {
	parsedURL, err := url.ParseRequestURI(artifactURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	fmt.Printf("Parsed URL: %+v\n", parsedURL)

	coordinates := &ArtifactCoordinates{
		Host:     parsedURL.Host,
		Artifact: path.Base(parsedURL.Path),
	}

	rawPathRegEx := regexp.MustCompile(`^/(.+)$`)
	if matches := rawPathRegEx.FindStringSubmatch(parsedURL.Path); len(matches) == rawPathMatchCount {
		coordinates.RawPath = matches[1]
	}

	gitHubReleaseRegEx := regexp.MustCompile(`^/([^/]+)/([^/]+)/releases/download/([^/]+)/(.+)$`)
	if matches := gitHubReleaseRegEx.FindStringSubmatch(parsedURL.Path); len(matches) == githubReleaseMatchCount {
		coordinates.Org = matches[1]
		coordinates.Repo = matches[2]
		coordinates.Version = matches[3]
		coordinates.Artifact = matches[4]
	}

	return coordinates, nil
}
