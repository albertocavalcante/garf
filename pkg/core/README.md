# Core Package

This package provides the core interfaces and types used throughout the application.

## Components

### Artifact

The `Artifact` type represents an artifact that can be mirrored between sources and destinations.

```go
type Artifact struct {
    Name     string
    Version  string
    Location string
    Metadata map[string]string
}
```

### Source Interface

The `Source` interface defines the contract for artifact sources:

```go
type Source interface {
    Get(ctx context.Context, artifact *Artifact) (io.ReadCloser, error)
    List(ctx context.Context) ([]*Artifact, error)
    Validate() error
}
```

### Destination Interface

The `Destination` interface defines the contract for artifact destinations:

```go
type Destination interface {
    Put(ctx context.Context, artifact *Artifact, content io.Reader) error
    Exists(ctx context.Context, artifact *Artifact) (bool, error)
    Validate() error
}
```

### MirrorOptions

The `MirrorOptions` type defines options for mirroring operations:

```go
type MirrorOptions struct {
    PreserveStructure bool
    VerifyChecksum    bool
    Concurrent        int
    Context          context.Context
}
```

## Usage

### Creating an Artifact

```go
artifact := &core.Artifact{
    Name:     "example",
    Version:  "1.0.0",
    Location: "path/to/artifact",
    Metadata: map[string]string{
        "key": "value",
    },
}
```

### Implementing a Source

```go
type CustomSource struct {
    // ... fields
}

func (s *CustomSource) Get(ctx context.Context, artifact *core.Artifact) (io.ReadCloser, error) {
    // Implementation
}

func (s *CustomSource) List(ctx context.Context) ([]*core.Artifact, error) {
    // Implementation
}

func (s *CustomSource) Validate() error {
    // Implementation
}
```

### Implementing a Destination

```go
type CustomDestination struct {
    // ... fields
}

func (d *CustomDestination) Put(ctx context.Context, artifact *core.Artifact, content io.Reader) error {
    // Implementation
}

func (d *CustomDestination) Exists(ctx context.Context, artifact *core.Artifact) (bool, error) {
    // Implementation
}

func (d *CustomDestination) Validate() error {
    // Implementation
}
```

## Best Practices

1. Always implement proper error handling in interface methods
2. Use context for cancellation and timeouts
3. Clean up resources properly
4. Validate inputs and configurations
5. Provide meaningful error messages
6. Use appropriate logging levels
7. Handle concurrent access safely 