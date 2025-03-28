# Destinations Package

This package provides implementations of the `core.Destination` interface for various artifact storage systems.

## Implementations

### JFrog Artifactory

The `JFrogDestination` type implements the `core.Destination` interface for JFrog Artifactory. It provides functionality to:

- Upload artifacts to Artifactory
- Check if artifacts exist
- Validate configuration
- Handle authentication
- Manage artifact properties

## Usage

```go
config := destinations.JFrogConfig{
    URL:      "https://your-instance.jfrog.io/artifactory",
    User:     "username",
    Password: "password",
}

dest := destinations.NewJFrogDestination(config, logger)
```

## Configuration

The JFrog destination requires the following configuration:

- `URL`: The base URL of your JFrog Artifactory instance
- `User`: Username for authentication
- `Password`: Password for authentication

## Environment Variables

The following environment variables are used:

- `JFROG_URL`: The base URL of your JFrog Artifactory instance
- `JFROG_USER`: Username for authentication
- `JFROG_PASSWORD`: Password for authentication 