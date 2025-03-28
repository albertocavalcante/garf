# Archive Package

The `archive` package provides functionality for handling archive files in the Garf project. Currently, it supports ZIP file operations with a focus on extracting single files from ZIP archives.

## Features

- ZIP file detection
- Single file extraction from ZIP archives
- Custom error types for better error handling
- Configurable extraction options

## Usage

### Checking if a File is a ZIP

```go
import "github.com/albertocavalcante/garf/pkg/archive"

isZip := archive.IsZipFile("example.zip")
```

### Extracting a Single File from a ZIP Archive

```go
import "github.com/albertocavalcante/garf/pkg/archive"

options := archive.ExtractOptions{
    DestinationDir: "output",
    PreserveOriginalName: true,
}

extractedPath, err := archive.ExtractSingleFile("archive.zip", options)
if err != nil {
    // Handle error
}
```

## Error Handling

The package provides a custom error type `ZipError` for better error handling:

```go
if err != nil {
    if zipErr, ok := err.(*archive.ZipError); ok {
        // Handle ZIP-specific errors
        fmt.Printf("ZIP error: %s\n", zipErr.Error())
    }
}
```

## Options

### ExtractOptions

```go
type ExtractOptions struct {
    // DestinationDir specifies where the extracted file should be placed
    DestinationDir string
    // PreserveOriginalName determines if the original filename should be kept
    PreserveOriginalName bool
}
```

## Limitations

- Currently only supports ZIP archives
- Designed for extracting single files from ZIP archives
- Does not support password-protected ZIP files
- Does not support nested ZIP files

## Contributing

When adding new features to this package:

1. Add appropriate tests in `zip_test.go`
2. Update this README with new functionality
3. Follow Go best practices and idioms
4. Ensure error handling is comprehensive

## License

This package is part of the Garf project and follows its licensing terms. 