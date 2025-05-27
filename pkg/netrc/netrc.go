// Package netrc provides functionality for reading and parsing .netrc files.
//
// Security Note: .netrc files store credentials in plaintext and should have
// restrictive permissions (0600) to prevent unauthorized access. This package
// validates file permissions and will return an error if the file is readable
// by others.
//
// Example usage:
//
//	// Get credentials for a specific host
//	creds, found, err := netrc.GetHostCredentials("example.com")
//	if err != nil {
//		log.Fatal(err)
//	}
//	if found {
//		fmt.Printf("User: %s\n", creds.Login)
//	}
//
//	// Use a specific .netrc file
//	f, err := netrc.NewFile("/path/to/.netrc")
//	if err != nil {
//		log.Fatal(err)
//	}
//	creds, found, err := f.GetCredentials("example.com")
package netrc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// DefaultNetrcFilename is the standard filename for netrc files.
const DefaultNetrcFilename = ".netrc"

// Credentials represents a username and password pair from a .netrc file.
// It holds the Login (username) and Password.
type Credentials struct {
	Login    string
	Password string
}

// IsEmpty returns true if both Login and Password are empty.
func (c Credentials) IsEmpty() bool {
	return c.Login == "" && c.Password == ""
}

// File represents a .netrc file and provides methods to retrieve credentials.
// Its main purpose is to encapsulate the file path.
type File struct {
	path string
}

// NewFile creates a new File instance for the specified .netrc file path.
// If the provided path is empty, it uses the default .netrc file location.
func NewFile(path string) (*File, error) {
	if path == "" {
		var err error

		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	return &File{path: path}, nil
}

// DefaultPath returns the path to the default .netrc file.
// It checks NETRC, USERPROFILE, HOME environment variables, then os.UserHomeDir().
func DefaultPath() (string, error) {
	if netrcPath := os.Getenv("NETRC"); netrcPath != "" {
		return netrcPath, nil
	}

	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return filepath.Join(userProfile, DefaultNetrcFilename), nil
	}

	if homeEnv := os.Getenv("HOME"); homeEnv != "" {
		return filepath.Join(homeEnv, DefaultNetrcFilename), nil
	}

	osHome, err := os.UserHomeDir()
	if err == nil && osHome != "" {
		return filepath.Join(osHome, DefaultNetrcFilename), nil
	}

	return "", errors.New("netrc: could not determine home directory " +
		"(checked NETRC, USERPROFILE, HOME, and os.UserHomeDir)")
}

// validateFilePermissions checks if the .netrc file has secure permissions.
// On Unix-like systems, it ensures the file is not readable by group or others.
func validateFilePermissions(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	perm := info.Mode().Perm()
	if perm&0o077 != 0 {
		return fmt.Errorf("netrc: .netrc file %q has insecure permissions %o, should be 0600 or more restrictive", path, perm)
	}

	return nil
}

// GetCredentials reads the .netrc file and parses it to find credentials for the specified host.
// If the .netrc file does not exist, it returns false for found and no error.
func (f *File) GetCredentials(host string) (Credentials, bool, error) {
	if host == "" {
		return Credentials{}, false, fmt.Errorf("netrc: host cannot be empty")
	}

	if _, err := os.Stat(f.path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Credentials{}, false, nil
		}

		return Credentials{}, false, fmt.Errorf("netrc: failed to stat file %q: %w", f.path, err)
	}

	if err := validateFilePermissions(f.path); err != nil {
		return Credentials{}, false, err
	}

	data, err := os.ReadFile(f.path)
	if err != nil {
		return Credentials{}, false, fmt.Errorf("netrc: failed to read file %q: %w", f.path, err)
	}

	login, password, found, parseErr := ParseNetrcFile(string(data), host)
	if parseErr != nil {
		return Credentials{}, false, fmt.Errorf("netrc: failed to parse file %q: %w", f.path, parseErr)
	}

	if !found {
		return Credentials{}, false, nil
	}

	return Credentials{
		Login:    login,
		Password: password,
	}, true, nil
}

// GetHostCredentials is a convenience function that gets credentials for the specified host
// using the default .netrc path.
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

// unquoteToken removes surrounding double quotes from a token and unescapes sequences.
// Supported escapes: \n, \r, \t, \", \\.
// Unknown escapes (e.g., \q) result in the backslash and the character being preserved.
func unquoteToken(token string) string {
	if len(token) < 2 || token[0] != '"' || token[len(token)-1] != '"' {
		return token
	}

	s := token[1 : len(token)-1]

	var sb strings.Builder

	sb.Grow(len(s))

	for i := 0; i < len(s); i++ {
		char := s[i]
		if char == '\\' {
			if i+1 < len(s) {
				i++

				escapedChar := s[i]
				switch escapedChar {
				case 'n':
					sb.WriteRune('\n')
				case 'r':
					sb.WriteRune('\r')
				case 't':
					sb.WriteRune('\t')
				case '"':
					sb.WriteRune('"')
				case '\\':
					sb.WriteRune('\\')
				default:
					sb.WriteRune('\\')
					sb.WriteRune(rune(escapedChar))
				}
			} else {
				sb.WriteRune('\\')
			}
		} else {
			sb.WriteRune(rune(char))
		}
	}

	return sb.String()
}

// tokenizeLine splits a line from a .netrc file into tokens.
// It respects double-quoted strings, treating them as single tokens (quotes included).
// Whitespace outside of quotes acts as a delimiter.
// Backslashes within quotes are preserved for unquoteToken to handle.
func tokenizeLine(line string) []string {
	var tokens []string

	var currentToken strings.Builder

	inQuotes := false
	escapeNextInQuote := false

	for _, char := range line {
		if escapeNextInQuote {
			currentToken.WriteRune(char)

			escapeNextInQuote = false

			continue
		}

		if char == '\\' {
			currentToken.WriteRune(char)

			if inQuotes {
				escapeNextInQuote = true
			}

			continue
		}

		if char == '"' {
			currentToken.WriteRune(char)

			inQuotes = !inQuotes
			if !inQuotes {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}

			continue
		}

		if unicode.IsSpace(char) && !inQuotes {
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}

			continue
		}

		currentToken.WriteRune(char)
	}

	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	return tokens
}

// ParseNetrcFile parses .netrc file content and returns credentials for the given host.
// It aims to follow common .netrc conventions, including handling of 'machine', 'default',
// 'login', 'password' keywords, quoted strings with escapes, and skipping comments/macdef.
// The first complete 'machine' entry matching targetHost wins.
// If no specific machine matches, the first complete 'default' entry wins.
func ParseNetrcFile(content string, targetHost string) (login, password string, found bool, err error) {
	var hostLogin, hostPassword string

	var hostFound bool

	var defaultLogin, defaultPassword string

	var defaultFound bool

	activeMachine := ""
	activeLogin := ""
	activeLoginSet := false
	inMachineContext := false
	inDefaultContext := false

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if hostFound {
			break
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") || strings.HasPrefix(trimmedLine, "macdef") {
			continue
		}

		tokens := tokenizeLine(trimmedLine)
		for i := 0; i < len(tokens); i++ {
			keyword := tokens[i]
			value := ""
			valueExists := (i+1 < len(tokens))

			if valueExists {
				value = tokens[i+1]
			}

			switch keyword {
			case "machine":
				inMachineContext = false
				inDefaultContext = false
				activeLogin = ""
				activeLoginSet = false

				if valueExists && value != "" {
					activeMachine = unquoteToken(value)
					inMachineContext = true
					i++
				} else {
					activeMachine = ""
				}
			case "default":
				inMachineContext = false
				inDefaultContext = true
				activeLogin = ""
				activeLoginSet = false
				activeMachine = ""
			case "login":
				if (inMachineContext || inDefaultContext) && valueExists {
					activeLogin = unquoteToken(value)
					activeLoginSet = true
					i++
				} else {
					activeLogin = ""
					activeLoginSet = false

					if valueExists {
						i++
					}
				}
			case "password":
				if (inMachineContext || inDefaultContext) && activeLoginSet && valueExists {
					currentPass := unquoteToken(value)

					if inMachineContext && activeMachine == targetHost && !hostFound {
						hostLogin = activeLogin
						hostPassword = currentPass
						hostFound = true

						break
					} else if inDefaultContext && !defaultFound {
						defaultLogin = activeLogin
						defaultPassword = currentPass
						defaultFound = true
					}

					activeLogin = ""
					activeLoginSet = false
					i++
				} else {
					activeLogin = ""
					activeLoginSet = false

					if valueExists {
						i++
					}
				}
			default:
				if valueExists {
					nextToken := tokens[i+1]
					if nextToken != "machine" && nextToken != "default" && nextToken != "login" && nextToken != "password" {
						i++
					}
				}
			}

			if hostFound {
				break
			}
		}
	}

	if hostFound {
		return hostLogin, hostPassword, true, nil
	}

	if defaultFound {
		return defaultLogin, defaultPassword, true, nil
	}

	return "", "", false, nil
}
