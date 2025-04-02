package core_test

import (
	"context"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestMirrorOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		opts          *core.MirrorOptions
		validate      func(*testing.T, *core.MirrorOptions)
		validateError bool
	}{
		{
			name: "valid options",
			opts: &core.MirrorOptions{
				Raw:        false,
				Concurrent: 4,
				Context:    context.Background(),
			},
			validate: func(t *testing.T, opts *core.MirrorOptions) {
				require.False(t, opts.Raw)
				require.Equal(t, 4, opts.Concurrent)
				require.NotNil(t, opts.Context)
			},
			validateError: false,
		},
		{
			name: "zero concurrent",
			opts: &core.MirrorOptions{
				Raw:        false,
				Concurrent: 0,
				Context:    context.Background(),
			},
			validateError: true,
		},
		{
			name: "nil context",
			opts: &core.MirrorOptions{
				Raw:        false,
				Concurrent: 4,
				Context:    nil,
			},
			validateError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.opts.Validate()
			if tt.validateError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			tt.validate(t, tt.opts)
		})
	}
}

type artifactTestCase struct {
	name          string
	artifact      *core.Artifact
	validate      func(*testing.T, *core.Artifact)
	validateError bool
}

func getArtifactTestCases() []artifactTestCase {
	return []artifactTestCase{
		{
			name: "valid artifact",
			artifact: &core.Artifact{
				Name:     "test-artifact",
				Version:  "1.0.0",
				Location: "https://example.com/test-artifact-1.0.0.zip",
				Metadata: map[string]string{
					"type": "toolchain",
					"os":   "linux",
				},
			},
			validate: func(t *testing.T, a *core.Artifact) {
				require.Equal(t, "test-artifact", a.Name)
				require.Equal(t, "1.0.0", a.Version)
				require.Equal(t, "https://example.com/test-artifact-1.0.0.zip", a.Location)
				require.Equal(t, "toolchain", a.Metadata["type"])
				require.Equal(t, "linux", a.Metadata["os"])
			},
			validateError: false,
		},
		{
			name: "missing name",
			artifact: &core.Artifact{
				Version:  "1.0.0",
				Location: "https://example.com/test-artifact-1.0.0.zip",
			},
			validateError: true,
		},
		{
			name: "missing location",
			artifact: &core.Artifact{
				Name:    "test-artifact",
				Version: "1.0.0",
			},
			validateError: true,
		},
	}
}

func TestArtifact(t *testing.T) {
	t.Parallel()

	tests := getArtifactTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.artifact.Validate()
			if tt.validateError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			tt.validate(t, tt.artifact)
		})
	}
}
