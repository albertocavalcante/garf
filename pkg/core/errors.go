package core

import (
	"fmt"
	"strings"
)

// ErrorGroup collects multiple errors and combines them into a single error.
type ErrorGroup struct {
	errors []error
}

// NewErrorGroup creates a new ErrorGroup.
func NewErrorGroup() *ErrorGroup {
	return &ErrorGroup{
		errors: make([]error, 0),
	}
}

// Add adds an error to the group if it is not nil.
func (g *ErrorGroup) Add(err error) {
	if err != nil {
		g.errors = append(g.errors, err)
	}
}

// Err returns nil if no errors were added, otherwise returns
// a combined error with all error messages.
func (g *ErrorGroup) Err() error {
	if len(g.errors) == 0 {
		return nil
	}

	messages := make([]string, 0, len(g.errors))
	for _, err := range g.errors {
		messages = append(messages, err.Error())
	}

	return fmt.Errorf("multiple errors occurred:\n- %s", strings.Join(messages, "\n- "))
}
