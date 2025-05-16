// Package netrc provides functionality for reading and parsing .netrc files.
package netrc

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Credentials represents a username and password pair from a .netrc file.
type Credentials struct {
	Login    string
	Password string
}

// File represents a .netrc file and provides methods to retrieve credentials.
type File struct {
	path string
}

// NewFile creates a new File instance for the specified .netrc file path.
// If path is empty, it uses the default .netrc file location.
func NewFile(path string) (*File, error) {
	if path == "" {
		// Use default path in user's home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".netrc")
	}
	return &File{path: path}, nil
}

// DefaultPath returns the path to the default .netrc file.
// It respects the NETRC environment variable, falling back to ~/.netrc.
func DefaultPath() (string, error) {
	netrcPath := os.Getenv("NETRC")
	if netrcPath != "" {
		return netrcPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".netrc"), nil
}

// GetCredentials returns the credentials for the specified host from the .netrc file.
// It returns the credentials and a boolean indicating whether credentials were found.
func (f *File) GetCredentials(host string) (Credentials, bool, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Credentials{}, false, nil // File doesn't exist, not an error
		}
		return Credentials{}, false, err
	}

	login, password, found, err := parseNetrcFile(string(data), host)
	if err != nil {
		return Credentials{}, false, err
	}

	if !found {
		return Credentials{}, false, nil
	}

	return Credentials{
		Login:    login,
		Password: password,
	}, true, nil
}

// GetHostCredentials is a convenience function that creates a File instance
// using the default .netrc path and retrieves credentials for the specified host.
func GetHostCredentials(host string) (Credentials, bool, error) {
	path, err := DefaultPath()
	if err != nil {
		return Credentials{}, false, err
	}

	f, err := NewFile(path)
	if err != nil {
		return Credentials{}, false, err
	}

	return f.GetCredentials(host)
}

// parseNetrcFile parses .netrc file content and returns credentials for the given host.
// It handles line-based parsing with proper handling of comments and whitespace.
func parseNetrcFile(content string, targetHost string) (string, string, bool, error) {
	var currentMachine string
	var login, password string

	// Split content by lines
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		// Skip empty lines and comments
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Split line into tokens and process
		tokens := strings.Fields(trimmedLine)
		if len(tokens) == 0 {
			continue
		}

		// Process tokens in this line
		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			switch token {
			case "machine":
				if i+1 < len(tokens) {
					currentMachine = tokens[i+1]
					if currentMachine != targetHost {
						// Reset credentials when switching to a different machine
						login = ""
						password = ""
					}
					i++
				}
			case "login":
				if currentMachine == targetHost && i+1 < len(tokens) {
					login = tokens[i+1]
					i++
				}
			case "password":
				if currentMachine == targetHost && i+1 < len(tokens) {
					password = tokens[i+1]
					i++
				}
			}
		}

		// Check if we have complete credentials after each line
		if currentMachine == targetHost && login != "" && password != "" {
			return login, password, true, nil
		}
	}

	// If machine isn't found or credentials are incomplete, just return not found
	return "", "", false, nil
}
