package core_test

import (
	"errors"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name     string
	errors   []error
	wantErr  bool
	contains []string
}

var testCases = []testCase{
	{
		name:    "no errors",
		errors:  nil,
		wantErr: false,
	},
	{
		name:    "empty error slice",
		errors:  []error{},
		wantErr: false,
	},
	{
		name: "single error",
		errors: []error{
			errors.New("error one"),
		},
		wantErr:  true,
		contains: []string{"error one"},
	},
	{
		name: "multiple errors",
		errors: []error{
			errors.New("error one"),
			errors.New("error two"),
		},
		wantErr:  true,
		contains: []string{"error one", "error two"},
	},
	{
		name: "nil errors are ignored",
		errors: []error{
			errors.New("error one"),
			nil,
			errors.New("error two"),
		},
		wantErr:  true,
		contains: []string{"error one", "error two"},
	},
	{
		name: "all nil errors",
		errors: []error{
			nil,
			nil,
		},
		wantErr: false,
	},
}

func TestErrorGroup(t *testing.T) {
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			g := core.NewErrorGroup()
			for _, err := range tt.errors {
				g.Add(err)
			}

			err := g.Err()
			if !tt.wantErr {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			errStr := err.Error()
			for _, substr := range tt.contains {
				require.Contains(t, errStr, substr)
			}
		})
	}
}
