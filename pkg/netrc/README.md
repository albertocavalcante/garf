# netrc

The `netrc` package provides functionality for reading and parsing `.netrc` files for authentication credentials.

## Overview

This package allows reading credentials from `.netrc` files, which are commonly used to store authentication information for various services. The package follows standard `.netrc` conventions and includes security validations.

## Security

⚠️ **Security Note**: `.netrc` files store credentials in plaintext and should have restrictive permissions (0600) to prevent unauthorized access. This package validates file permissions on Unix-like systems and will return an error if the file is readable by others.

## Usage

### Basic Usage

```go
import "github.com/albertocavalcante/garf/pkg/netrc"

// Get credentials for a specific host using default .netrc location
creds, found, err := netrc.GetHostCredentials("example.com")
if err != nil {
    log.Fatal(err)
}
if found {
    fmt.Printf("User: %s\n", creds.Login)
    fmt.Printf("Password: %s\n", creds.Password)
}
```

### Using a Specific File

```go
// Use a specific .netrc file
f, err := netrc.NewFile("/path/to/.netrc")
if err != nil {
    log.Fatal(err)
}

creds, found, err := f.GetCredentials("example.com")
if err != nil {
    log.Fatal(err)
}
```

## Types

### Credentials

Represents a username and password pair from a `.netrc` file.

```go
type Credentials struct {
    Login    string
    Password string
}
```

### File

Represents a `.netrc` file and provides methods to retrieve credentials.

```go
type File struct {
    // private fields
}
```

## Functions

- `GetHostCredentials(host string) (Credentials, bool, error)` - Convenience function to get credentials from default `.netrc` location
- `NewFile(path string) (*File, error)` - Create a new File instance for a specific `.netrc` file
- `DefaultPath() (string, error)` - Get the default `.netrc` file path

## Environment Variables

The package respects the following environment variables for determining the `.netrc` file location:

1. `NETRC` - Direct path to `.netrc` file (highest priority)
2. `USERPROFILE` - Windows home directory
3. `HOME` - Unix-like home directory
4. Falls back to `os.UserHomeDir()` 