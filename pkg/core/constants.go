// Package core provides core types and constants for the garf application.
package core

// Source type constants.
const (
	// SourceTypeGitHub represents the GitHub source type for downloading from GitHub releases.
	SourceTypeGitHub = "github"

	// SourceTypeGeneric represents the generic HTTP source type for downloading from any HTTP/HTTPS URL.
	SourceTypeGeneric = "generic"
)

// Registry type constants.
const (
	// RegistryTypeJFrog represents the JFrog Artifactory registry type.
	RegistryTypeJFrog = "jfrog"

	// RegistryTypeCloudsmith represents the Cloudsmith registry type.
	RegistryTypeCloudsmith = "cloudsmith"
)
