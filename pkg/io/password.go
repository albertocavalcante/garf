package io

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadPasswordFromStdin reads a password from standard input.
// It trims any leading or trailing whitespace from the input.
// If an error occurs during reading, it returns an empty string and the error.
func ReadPasswordFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}

	return strings.TrimSpace(password), nil
}
