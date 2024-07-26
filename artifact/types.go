package artifact

// ArtifactCoordinates represents the coordinates of an artifact.
type ArtifactCoordinates struct {
	Host     string
	Org      string
	Repo     string
	Version  string
	Artifact string
}

// UrlPath creates a URL path from the artifact coordinates.
func (ac *ArtifactCoordinates) UrlPath() string {
	return ac.Host + "/" + ac.Org + "/" + ac.Repo + "/" + ac.Version + "/" + ac.Artifact
}
