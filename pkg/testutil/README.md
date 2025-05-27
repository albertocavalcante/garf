# testutil

The `testutil` package provides shared testing utilities for the garf project.

## Functions

### CreateTemporaryNetrc

Creates a temporary `.netrc` file with the specified content and returns its path. The file is created with 0600 permissions for security.

```go
func CreateTemporaryNetrc(t *testing.T, content string) string
```

### WithEnv

Temporarily sets environment variables for the duration of a test function. It automatically restores the original values when the test completes.

```go
func WithEnv(t *testing.T, env map[string]string, fn func())
```

## Usage

```go
import "github.com/albertocavalcante/garf/pkg/testutil"

func TestExample(t *testing.T) {
    // Create a temporary .netrc file
    netrcPath := testutil.CreateTemporaryNetrc(t, "machine example.com login user password pass")
    
    // Set environment variables temporarily
    testutil.WithEnv(t, map[string]string{"NETRC": netrcPath}, func() {
        // Your test code here
    })
} 