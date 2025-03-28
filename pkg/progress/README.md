# Progress Package

The `progress` package provides utilities for tracking progress of operations in the `garf` CLI tool.

## Overview

This package implements a progress tracking system that can be used to monitor the progress of operations like downloading and uploading artifacts. It provides a simple interface for tracking bytes processed and reporting progress through callback functions.

## Components

### ProgressFunc

```go
type ProgressFunc func(current, total int64, message string)
```

A callback function type that is called to report progress during operations. It receives:
- `current`: The current number of bytes processed
- `total`: The total number of bytes to process
- `message`: A descriptive message about the current operation

### Reader

```go
type Reader struct {
    reader       io.Reader
    total        int64
    current      int64
    progressFunc ProgressFunc
}
```

A wrapper around `io.Reader` that tracks progress during read operations.

## Usage

### Creating a Progress Reader

```go
import "github.com/albertocavalcante/garf/pkg/progress"

// Create a progress reader
reader := progress.NewReader(
    sourceReader,    // The original io.Reader
    totalBytes,      // Total number of bytes to process
    progressFunc,    // Callback function to report progress
)
```

### Example Progress Function

```go
progressFunc := func(current, total int64, message string) {
    if total > 0 {
        percentage := float64(current) / float64(total) * 100
        fmt.Printf("%s: %.2f%% (%d/%d bytes)\n", message, percentage, current, total)
    }
}
```

## Integration

This package is used internally by the `garf` CLI to track progress during:
- Downloading artifacts from sources
- Uploading artifacts to JFrog Artifactory
- Processing and extracting files

## Example

```go
package main

import (
    "fmt"
    "io"
    "os"

    "github.com/albertocavalcante/garf/pkg/progress"
)

func main() {
    // Open a file
    file, err := os.Open("large-file.dat")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    // Get file size
    stat, err := file.Stat()
    if err != nil {
        panic(err)
    }
    total := stat.Size()

    // Create progress function
    progressFunc := func(current, total int64, message string) {
        if total > 0 {
            percentage := float64(current) / float64(total) * 100
            fmt.Printf("%s: %.2f%% (%d/%d bytes)\n", message, percentage, current, total)
        }
    }

    // Create progress reader
    reader := progress.NewReader(file, total, progressFunc)

    // Read the file
    buffer := make([]byte, 1024)
    for {
        _, err := reader.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
    }
}
```

## Best Practices

1. Always provide a meaningful message in the progress function to help users understand what operation is being performed.
2. Handle the case where total size is unknown (total = 0) in your progress function.
3. Use appropriate buffer sizes when reading from the progress reader to ensure smooth progress updates.
4. Consider rate limiting progress updates to avoid overwhelming the output. 