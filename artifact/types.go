package artifact

// ArtifactCoordinates represents the coordinates of an artifact.
type ArtifactCoordinates struct {
	Host     string
	Org      string
	Repo     string
	Version  string
	Artifact string
	RawPath  string
}

// RawUrlPath creates a URL path from the artifact minimum coordinates.
func (ac *ArtifactCoordinates) RawUrlPath() string {
	return ac.Host + "/" + ac.RawPath
}

// UrlPath creates a URL path from the parsed artifact coordinates.
func (ac *ArtifactCoordinates) UrlPath() string {
	return ac.Host + "/" + ac.Org + "/" + ac.Repo + "/" + ac.Version + "/" + ac.Artifact
}
