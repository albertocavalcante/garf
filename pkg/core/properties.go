package core

import "strings"

const propertyKeyValueParts = 2

// ParseProperties converts the properties array into a map.
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
