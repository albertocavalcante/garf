# Garf Library Usage Examples

This directory contains comprehensive, working examples of how to use garf as a Go library in your applications.

## 📋 Quick Start

### Prerequisites

- Go 1.21 or later
- Bazel (for building)
- JFrog Artifactory instance (for actual mirroring)

### Running the Examples

```bash
# Build the examples
bazel build //examples:examples

# Run all examples (uses placeholder credentials by default)
bazel run //examples:examples

# Run with real credentials (environment variables)
export JFROG_URL="https://your-company.jfrog.io/artifactory"
export JFROG_USER="your-username"
export JFROG_PASSWORD="your-password"
bazel run //examples:examples
```

## 📚 Examples Overview

### 1. **Basic Example**
- **Purpose**: Minimal configuration and simple mirroring
- **Demonstrates**: 
  - Client creation with basic config
  - Simple mirror operation
  - Basic error handling

```go
client, err := garf.NewClient(garf.Config{
    JFrogURL:      "https://mycompany.jfrog.io/artifactory",
    JFrogUser:     "username",
    JFrogPassword: "password",
})

result, err := client.Mirror(ctx, garf.MirrorRequest{
    Source:      "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
    Destination: "tools-local",
})
```

### 2. **Advanced Example**
- **Purpose**: Custom configuration with all available options
- **Demonstrates**:
  - Custom timeouts and concurrency settings
  - Artifact properties and metadata
  - Unzip functionality
  - Path organization options

```go
client, err := garf.NewClient(garf.Config{
    JFrogURL:      "https://mycompany.jfrog.io/artifactory",
    JFrogUser:     "username", 
    JFrogPassword: "password",
    Timeout:       10 * time.Minute,
    Concurrent:    8,
})

result, err := client.Mirror(ctx, garf.MirrorRequest{
    Source:      "https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip",
    Destination: "tools-local",
    Properties: map[string]string{
        "type":     "toolchain",
        "platform": "windows",
        "arch":     "x86_64",
        "version":  "7.6.0",
    },
    Unzip: true,
    Raw:   false,
})
```

### 3. **Environment Configuration Example** ✅ **Recommended**
- **Purpose**: Production-ready configuration using environment variables
- **Demonstrates**:
  - Secure credential management
  - Environment variable usage
  - Graceful handling of missing configuration
  - Dry run validation

```go
client, err := garf.NewClient(garf.Config{
    JFrogURL:      os.Getenv("JFROG_URL"),
    JFrogUser:     os.Getenv("JFROG_USER"),
    JFrogPassword: os.Getenv("JFROG_PASSWORD"),
})
```

### 4. **Batch Operations Example**
- **Purpose**: Efficiently mirror multiple artifacts
- **Demonstrates**:
  - Batch processing patterns
  - Progress tracking
  - Error resilience (continue on individual failures)
  - Success rate reporting

```go
artifacts := []struct {
    source      string
    destination string
    properties  map[string]string
}{
    {
        source:      "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-linux-x86_64",
        destination: "tools-local",
        properties:  map[string]string{"platform": "linux", "arch": "x86_64"},
    },
    // ... more artifacts
}

for _, artifact := range artifacts {
    result, err := client.Mirror(ctx, garf.MirrorRequest{
        Source:      artifact.source,
        Destination: artifact.destination,
        Properties:  artifact.properties,
    })
    // Handle individual results...
}
```

### 5. **Error Handling & Dry Run Example**
- **Purpose**: Production-ready error handling and validation
- **Demonstrates**:
  - Dry run validation before actual operations
  - Context timeouts and cancellation
  - Comprehensive error handling patterns
  - Different error types (network, timeout, operation)

```go
// Step 1: Validate with dry run
result, err := client.Mirror(ctx, garf.MirrorRequest{
    Source:      source,
    Destination: "tools-local",
    DryRun:      true,
    DryRunMode:  "all",
})

// Step 2: Perform actual mirror with timeout
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()

result, err = client.Mirror(ctx, garf.MirrorRequest{
    Source:      source,
    Destination: "tools-local",
    Properties: map[string]string{
        "validated": "true",
        "timestamp": time.Now().Format(time.RFC3339),
    },
})

// Comprehensive error handling
switch {
case err != nil:
    if ctx.Err() == context.DeadlineExceeded {
        log.Printf("⏰ Mirror timed out: %v", err)
    } else {
        log.Printf("❌ Mirror failed: %v", err)
    }
case result.Error != nil:
    log.Printf("❌ Mirror operation failed: %v", result.Error)
default:
    fmt.Printf("✅ Successfully mirrored to %s\n", result.DestinationPath)
}
```

## ✅ Best Practices

### 1. **Use Environment Variables**
```go
// ✅ Good - Secure and flexible
client, err := garf.NewClient(garf.Config{
    JFrogURL:      os.Getenv("JFROG_URL"),
    JFrogUser:     os.Getenv("JFROG_USER"),
    JFrogPassword: os.Getenv("JFROG_PASSWORD"),
})

// ❌ Avoid - Hardcoded credentials
client, err := garf.NewClient(garf.Config{
    JFrogURL:      "https://company.jfrog.io/artifactory",
    JFrogUser:     "hardcoded-user",
    JFrogPassword: "hardcoded-password",
})
```

### 2. **Always Check Both Error Types**
```go
result, err := client.Mirror(ctx, request)
if err != nil {
    // Handle client/request errors
    return fmt.Errorf("mirror failed: %w", err)
}
if result.Error != nil {
    // Handle operation errors
    return fmt.Errorf("mirror operation failed: %w", result.Error)
}
```

### 3. **Use Context for Timeouts**
```go
// ✅ Good - Explicit timeout control
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, err := client.Mirror(ctx, request)
```

### 4. **Validate with Dry Runs**
```go
// ✅ Good - Validate before actual operation
request.DryRun = true
request.DryRunMode = "all"
_, err := client.Mirror(ctx, request)
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

// Now perform actual operation
request.DryRun = false
result, err := client.Mirror(ctx, request)
```

### 5. **Handle Batch Operations Gracefully**
```go
// ✅ Good - Continue on individual failures
successCount := 0
for _, req := range requests {
    result, err := client.Mirror(ctx, req)
    if err != nil {
        log.Printf("Skipping %s: %v", req.Source, err)
        continue // Don't fail entire batch
    }
    successCount++
}
log.Printf("Completed: %d/%d successful", successCount, len(requests))
```

## ⚙️ Configuration Options

### Environment Variables (Recommended)

For production use, configure garf using environment variables:

```bash
# Required
export JFROG_URL="https://your-company.jfrog.io/artifactory"
export JFROG_USER="your-username"
export JFROG_PASSWORD="your-password"

# Optional (with defaults)
export GARF_TIMEOUT="30m"
export GARF_CONCURRENT="4"
```

### Configuration Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `JFrogURL` | string | ✅ | - | JFrog Artifactory base URL |
| `JFrogUser` | string | ✅ | - | JFrog username |
| `JFrogPassword` | string | ✅ | - | JFrog password or API token |
| `Logger` | *logrus.Logger | ❌ | Default logger | Custom logger instance |
| `Timeout` | time.Duration | ❌ | 30 minutes | Request timeout |
| `Concurrent` | int | ❌ | 4 | Number of concurrent operations |

### Mirror Request Options

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Source` | string | ✅ | - | Source URL of the artifact |
| `Destination` | string | ✅ | - | Destination repository name |
| `Properties` | map[string]string | ❌ | nil | Metadata properties to attach |
| `Raw` | bool | ❌ | false | Preserve original URL structure |
| `Unzip` | bool | ❌ | false | Extract single files from ZIP archives |
| `FromFile` | string | ❌ | "" | Upload from local file instead |
| `DryRun` | bool | ❌ | false | Skip actual operations |
| `DryRunMode` | string | ❌ | "all" | Dry run mode: "all" or "upload" |

## 🔍 Troubleshooting

### Common Issues

1. **Authentication Errors**
   ```
   Error: invalid configuration: JFrogUser is required
   ```
   **Solution**: Ensure all required environment variables are set.

2. **Network Timeouts**
   ```
   Error: context deadline exceeded
   ```
   **Solution**: Increase timeout or check network connectivity.

3. **Permission Errors**
   ```
   Error: 403 Forbidden
   ```
   **Solution**: Verify JFrog user has write permissions to the destination repository.

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
logger := logrus.New()
logger.SetLevel(logrus.DebugLevel)

client, err := garf.NewClient(garf.Config{
    // ... other config
    Logger: logger,
})
```

## 📖 Additional Resources

- **[Library API Documentation](../docs/library_api.md)** - Complete API reference
- **[Library Maturity Assessment](../docs/library_maturity_assessment.md)** - Development roadmap
- **[Main README](../README.md)** - Project overview and CLI usage

## 🤝 Contributing

Found an issue with the examples or have suggestions for improvements? Please:

1. Check existing issues in the [GitHub repository](https://github.com/albertocavalcante/garf)
2. Create a new issue with the `examples` label
3. Submit a pull request with improvements

## 📄 License

These examples are part of the garf project and are licensed under the same terms. See [LICENSE](../LICENSE) for details.