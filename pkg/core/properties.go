package core

import "strings"

const propertyKeyValueParts = 2

// ParseProperties converts a slice of "key=value" strings into a map of key-value pairs.
// For each string, it splits at the first '=' character and trims any surrounding whitespace from both the key and value.
// Strings that do not consist of exactly one '=' are ignored.
func ParseProperties(props []string) map[string]string {
	result := make(map[string]string)

	for _, prop := range props {
		parts := strings.SplitN(prop, "=", propertyKeyValueParts)
		if len(parts) == propertyKeyValueParts {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	return result
}
