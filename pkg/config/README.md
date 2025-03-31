# Config Package

This package provides configuration management functionality for the application.

## Components

### Config

The `Config` type represents the application's configuration:

```go
type Config struct {
    Sources      map[string]SourceConfig
    Destinations map[string]DestinationConfig
}
```

### SourceConfig

The `SourceConfig` type defines configuration for artifact sources:

```go
type SourceConfig struct {
    Type string
    URL  string
    // Additional source-specific fields
}
```

### DestinationConfig

The `DestinationConfig` type defines configuration for artifact destinations:

```go
type DestinationConfig struct {
    Type     string
    URL      string
    User     string
    Password string
    // Additional destination-specific fields
}
```

## Usage

### Loading Configuration

```go
config, err := config.LoadConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}
```

### Configuration File Format

Example YAML configuration:

```yaml
sources:
  github:
    type: github
    url: https://github.com/owner/repo

destinations:
  jfrog:
    type: jfrog
    url: https://jfrog.example.com
    user: username
    password: password
```

### Validating Configuration

```go
if err := config.Validate(); err != nil {
    log.Printf("Invalid configuration: %v", err)
}
```

## Supported Source Types

- `github`: GitHub releases source
- (Add other source types as they are implemented)

## Supported Destination Types

- `jfrog`: JFrog Artifactory destination
- (Add other destination types as they are implemented)

## Environment Variables

The following environment variables can be used to override configuration:

- `GITHUB_TOKEN`: GitHub API token
- `JFROG_URL`: JFrog Artifactory URL
- `JFROG_USER`: JFrog username
- `JFROG_PASSWORD`: JFrog password

## Best Practices

1. Use environment variables for sensitive information
2. Validate configuration before use
3. Provide meaningful error messages
4. Use appropriate default values
5. Document all configuration options
6. Handle missing or invalid values gracefully
7. Support both file-based and environment-based configuration 