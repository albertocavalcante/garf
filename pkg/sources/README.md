# Sources Package

This package provides implementations of the `core.Source` interface for different artifact sources.

## Components

### GitHub Source

The `GitHubSource` implements the `core.Source` interface for downloading artifacts from GitHub releases.

#### Features

- Downloads artifacts from GitHub release URLs
- Supports GitHub token authentication via `GITHUB_TOKEN` environment variable
- Validates GitHub URLs
- Handles temporary file management for downloads
- Provides proper cleanup of resources

#### Usage

```go
source := sources.NewGitHubSource(logger)
artifact := &core.Artifact{
    Name:     "example",
    Version:  "1.0.0",
    Location: "https://github.com/owner/repo/releases/download/v1.0.0/artifact.zip",
}
content, err := source.Get(ctx, artifact)
```

#### Configuration

The GitHub source can be configured with:
- Custom HTTP client
- Custom logger
- GitHub token (via environment variable)

## Adding New Sources

To add a new source implementation:

1. Create a new type that implements the `core.Source` interface
2. Implement the required methods:
   - `Get(ctx context.Context, artifact *core.Artifact) (io.ReadCloser, error)`
   - `List(ctx context.Context) ([]*core.Artifact, error)`
   - `Validate() error`
3. Add appropriate tests in the corresponding test file
4. Update this README with documentation for the new source 