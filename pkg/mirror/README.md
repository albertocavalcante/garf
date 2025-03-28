# Mirror Package

This package provides the core mirroring functionality for copying artifacts between sources and destinations.

## Components

### DefaultMirror

The `DefaultMirror` type implements the `Mirror` interface and provides the main functionality for artifact mirroring.

#### Features

- Concurrent artifact processing
- Multiple source and destination support
- Progress tracking
- Artifact validation
- Configurable concurrency
- Thread-safe operations

#### Usage

```go
mirror := mirror.NewDefaultMirror(logger)

// Add source and destination
err := mirror.AddSource("github", githubSource)
err = mirror.AddDestination("jfrog", jfrogDest)

// Configure mirroring options
opts := &core.MirrorOptions{
    PreserveStructure: true,
    VerifyChecksum:    true,
    Concurrent:        4,
    Context:           ctx,
}

// Start mirroring
results := mirror.Mirror(ctx, artifacts, opts)

// Process results
for result := range results {
    if result.Error != nil {
        log.Printf("Failed to mirror %s: %v", result.Artifact.Name, result.Error)
    } else {
        log.Printf("Successfully mirrored %s to %s", result.Artifact.Name, result.DestinationPath)
    }
}
```

### Interfaces

#### Mirror

The `Mirror` interface defines the core mirroring functionality:

```go
type Mirror interface {
    Mirror(ctx context.Context, artifacts []*core.Artifact, opts *core.MirrorOptions) <-chan MirrorResult
    AddSource(name string, source core.Source) error
    AddDestination(name string, destination core.Destination) error
    GetSource(name string) (core.Source, error)
    GetDestination(name string) (core.Destination, error)
}
```

#### ProgressTracker

The `ProgressTracker` interface provides progress tracking capabilities:

```go
type ProgressTracker interface {
    Start(total int64)
    Update(current int64)
    Complete()
}
```

#### Validator

The `Validator` interface provides artifact validation capabilities:

```go
type Validator interface {
    ValidateArtifact(artifact *core.Artifact) error
    ValidateChecksum(artifact *core.Artifact, content io.Reader) error
}
```

## MirrorResult

The `MirrorResult` type contains the result of a mirror operation:

```go
type MirrorResult struct {
    Artifact         *core.Artifact
    DestinationPath  string
    Error           error
}
```

## Configuration

The mirror can be configured with:
- Custom logger
- Number of concurrent operations
- Progress tracking
- Artifact validation
- Source and destination mappings

## Error Handling

The mirror provides detailed error information through the `MirrorResult` channel, including:
- Artifact identification
- Destination path
- Specific error messages
- Operation status 