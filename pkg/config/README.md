# Config Package

The `config` package provides configuration management for the Garf tool. It handles both file-based configuration and environment variables.

## Configuration Structure

The configuration is structured as follows:

```go
type Config struct {
    Source      SourceConfig      `yaml:"source"`
    Destination DestinationConfig `yaml:"destination"`
    LogLevel    string           `yaml:"log_level"`
    Concurrent  int              `yaml:"concurrent"`
}

type SourceConfig struct {
    Type string `yaml:"type"`
    URL  string `yaml:"url"`
}

type DestinationConfig struct {
    Type     string `yaml:"type"`
    URL      string `yaml:"url"`
    User     string `yaml:"user"`
    Password string `yaml:"password"`
}
```

## Usage

### File-based Configuration

Create a YAML configuration file (e.g., `config.yaml`):

```yaml
source:
  type: github
  url: https://github.com/example/repo/releases/download/v1.0.0/artifact.zip

destination:
  type: jfrog
  url: https://artifactory.example.com/artifactory
  user: ${JFROG_USER}
  password: ${JFROG_PASSWORD}

log_level: info
concurrent: 4
```

### Environment Variables

The following environment variables are supported:

- `JFROG_URL`: The URL of the JFrog Artifactory instance
- `JFROG_USER`: The username for JFrog Artifactory authentication
- `JFROG_PASSWORD`: The password for JFrog Artifactory authentication

## Features

- YAML configuration file support
- Environment variable interpolation
- Default values for optional fields
- Validation of required fields
- Support for multiple source and destination types

## Example

```go
import "github.com/albertocavalcante/garf/pkg/config"

// Load configuration from file
cfg, err := config.Load("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Use configuration
source := cfg.Source
destination := cfg.Destination
``` 