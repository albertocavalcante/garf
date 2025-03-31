package io

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadPasswordFromStdin reads a password from stdin.
// ReadPasswordFromStdin reads a password from standard input by reading until a newline character is encountered.
// It trims any leading or trailing whitespace from the input. If an error occurs during reading, it returns an empty string
// and an error wrapping the original issue.
func ReadPasswordFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}

	return strings.TrimSpace(password), nil
}
