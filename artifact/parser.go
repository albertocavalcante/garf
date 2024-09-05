package artifact

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
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
	if matches := rawPathRegEx.FindStringSubmatch(parsedURL.Path); len(matches) == 2 {
		coordinates.RawPath = matches[1]
	}

	gitHubReleaseRegEx := regexp.MustCompile(`^/([^/]+)/([^/]+)/releases/download/([^/]+)/(.+)$`)
	if matches := gitHubReleaseRegEx.FindStringSubmatch(parsedURL.Path); len(matches) == 5 {
		coordinates.Org = matches[1]
		coordinates.Repo = matches[2]
		coordinates.Version = matches[3]
		coordinates.Artifact = matches[4]
	}

	return coordinates, nil
}
