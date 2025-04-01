# URL Processor Package

The `urlprocessor` package provides a flexible and extensible system for processing URLs into structured paths, particularly useful for artifact storage and organization.

## Overview

This package offers tools to transform URLs into organized file paths while supporting both raw URL preservation and clean structured paths. It's particularly optimized for handling GitHub release URLs but can be extended to support other URL patterns.

## Main Components

### Registry

The `Registry` type provides a centralized way to manage URL processors:

```go
registry := urlprocessor.NewRegistry()
result, err := registry.ProcessURLString(urlStr, raw)
```

### PathBuilder

An alternative implementation that provides URL processing capabilities:

```go
builder := urlprocessor.New()
result, err := builder.ProcessURLString(urlStr, raw)
```

### Processor Interface

Custom processors can be implemented using the `Processor` interface:

```go
type Processor interface {
    CanProcess(sourceURL *url.URL) bool
    Process(sourceURL *url.URL, raw bool) string
}
```

## Features

- **GitHub URL Support**: Special handling for GitHub release URLs
- **Raw Mode**: Preserve complete URL structure in output paths
- **Clean Mode**: Generate clean, structured paths
- **Extensible**: Easy to add custom processors for different URL patterns
- **Default Fallback**: Simple filename extraction for unrecognized URLs

## Example Usage

```go
// Create a new registry
registry := urlprocessor.NewRegistry()

// Process a GitHub release URL
url := "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe"

// Raw mode (preserves full path)
rawPath, _ := registry.ProcessURLString(url, true)
// Output: github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe

// Clean mode (structured path)
cleanPath, _ := registry.ProcessURLString(url, false)
// Output: github.com/bazelbuild/bazel/7.2.1/bazel-7.2.1-windows-x86_64.exe
```

## Adding Custom Processors

You can extend functionality by implementing the `Processor` interface or using the `PathBuilder.AddProcessor` method:

```go
builder := urlprocessor.New()
builder.AddProcessor(
    func(u *url.URL) bool { return strings.Contains(u.Host, "example.com") },
    func(u *url.URL, raw bool) string { return path.Base(u.Path) },
)
```