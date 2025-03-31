package io

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadPasswordFromStdin reads a password from stdin.
// It trims any whitespace from the input.
func ReadPasswordFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}

	return strings.TrimSpace(password), nil
}
