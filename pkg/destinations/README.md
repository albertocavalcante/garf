# Destinations Package

This package provides implementations of the `core.Destination` interface for different artifact destinations.

## Components

### JFrog Destination

The `JFrogDestination` implements the `core.Destination` interface for uploading artifacts to JFrog Artifactory.

#### Features

- Uploads artifacts to JFrog Artifactory
- Supports basic authentication
- Validates destination configuration
- Handles matrix parameters for artifact properties
- Checks for existing artifacts

#### Usage

```go
config := destinations.JFrogConfig{
    URL:      "https://jfrog.example.com",
    User:     "username",
    Password: "password",
}
dest := destinations.NewJFrogDestination(config, logger)

artifact := &core.Artifact{
    Name:     "example",
    Version:  "1.0.0",
    Location: "test-repo/example",
    Metadata: map[string]string{
        "prop1": "value1",
        "prop2": "value2",
    },
}
err := dest.Put(ctx, artifact, content)
```

#### Configuration

The JFrog destination requires:
- Artifactory URL
- Username
- Password
- Valid URL scheme (http/https)

## Adding New Destinations

To add a new destination implementation:

1. Create a new type that implements the `core.Destination` interface
2. Implement the required methods:
   - `Put(ctx context.Context, artifact *core.Artifact, content io.Reader) error`
   - `Exists(ctx context.Context, artifact *core.Artifact) (bool, error)`
   - `Validate() error`
3. Add appropriate tests in the corresponding test file
4. Update this README with documentation for the new destination 