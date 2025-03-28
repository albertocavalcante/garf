package mirror_test

import (
	"context"
	"io"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/stretchr/testify/require"
)

type mockSource struct {
	core.Source
}

func (s *mockSource) List(ctx context.Context) ([]*core.Artifact, error) {
	return nil, nil
}

func (s *mockSource) Get(ctx context.Context, artifact *core.Artifact) (io.ReadCloser, error) {
	return nil, nil
}

func (s *mockSource) Validate() error {
	return nil
}

type mockDestination struct {
	core.Destination
}

func (d *mockDestination) Put(ctx context.Context, artifact *core.Artifact, content io.Reader) error {
	return nil
}

func (d *mockDestination) Exists(ctx context.Context, artifact *core.Artifact) (bool, error) {
	return false, nil
}

func (d *mockDestination) Validate() error {
	return nil
}

func TestDefaultMirror(t *testing.T) {
	tests := []struct {
		name          string
		validate      func(*testing.T, *mirror.DefaultMirror)
		validateError bool
	}{
		{
			name: "valid mirror",
			validate: func(t *testing.T, m *mirror.DefaultMirror) {
				require.NotNil(t, m)
			},
			validateError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mirror := mirror.NewDefaultMirror(nil)
			tt.validate(t, mirror)
		})
	}
}

type testCase[T any] struct {
	name          string
	input         T
	validate      func(*testing.T, *mirror.DefaultMirror)
	validateError bool
}

func runTest[T any](t *testing.T, tt testCase[T], testFn func(*mirror.DefaultMirror, T) error) {
	t.Run(tt.name, func(t *testing.T) {
		mirror := mirror.NewDefaultMirror(nil)

		err := testFn(mirror, tt.input)
		if tt.validateError {
			require.Error(t, err)

			return
		}

		require.NoError(t, err)
		tt.validate(t, mirror)
	})
}

type sourceInput struct {
	sourceType string
	source     core.Source
}

func TestDefaultMirrorAddSource(t *testing.T) {
	tests := []testCase[sourceInput]{
		{
			name: "valid source",
			input: sourceInput{
				sourceType: "github",
				source:     &mockSource{},
			},
			validate: func(t *testing.T, m *mirror.DefaultMirror) {
				// Verify the source was added
				source, err := m.GetSource("github")
				require.NoError(t, err)
				require.NotNil(t, source)
			},
			validateError: false,
		},
		{
			name: "empty source type",
			input: sourceInput{
				sourceType: "",
				source:     &mockSource{},
			},
			validateError: true,
		},
	}

	for _, tt := range tests {
		runTest(t, tt, func(m *mirror.DefaultMirror, input sourceInput) error {
			return m.AddSource(input.sourceType, input.source)
		})
	}
}

type destinationInput struct {
	destType    string
	destination core.Destination
}

func TestDefaultMirrorAddDestination(t *testing.T) {
	tests := []testCase[destinationInput]{
		{
			name: "valid destination",
			input: destinationInput{
				destType:    "jfrog",
				destination: &mockDestination{},
			},
			validate: func(t *testing.T, m *mirror.DefaultMirror) {
				// Verify the destination was added
				dest, err := m.GetDestination("jfrog")
				require.NoError(t, err)
				require.NotNil(t, dest)
			},
			validateError: false,
		},
		{
			name: "empty destination type",
			input: destinationInput{
				destType:    "",
				destination: &mockDestination{},
			},
			validateError: true,
		},
	}

	for _, tt := range tests {
		runTest(t, tt, func(m *mirror.DefaultMirror, input destinationInput) error {
			return m.AddDestination(input.destType, input.destination)
		})
	}
}

type mirrorInput struct {
	artifacts []*core.Artifact
	opts      *core.MirrorOptions
}

func TestDefaultMirrorMirror(t *testing.T) {
	tests := []testCase[mirrorInput]{
		{
			name: "valid mirror",
			input: mirrorInput{
				artifacts: []*core.Artifact{
					{
						Name:     "test-artifact",
						Version:  "1.0.0",
						Location: "https://example.com/test-artifact-1.0.0.zip",
					},
				},
				opts: &core.MirrorOptions{
					PreserveStructure: true,
					VerifyChecksum:    true,
					Concurrent:        4,
					Context:           context.Background(),
				},
			},
			validate: func(t *testing.T, m *mirror.DefaultMirror) {
				require.NotNil(t, m)
			},
			validateError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mirror := mirror.NewDefaultMirror(nil)

			results := mirror.Mirror(tt.input.opts.Context, tt.input.artifacts, tt.input.opts)
			if tt.validateError {
				require.Nil(t, results)

				return
			}

			require.NotNil(t, results)
			tt.validate(t, mirror)
		})
	}
}
