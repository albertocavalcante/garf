package core_test

import (
	"testing"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestIsGitHubURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		want    bool
		wantErr bool
	}{
		{
			name: "valid GitHub URL",
			url:  "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
			want: true,
		},
		{
			name: "non-GitHub URL",
			url:  "https://example.com/file.zip",
			want: false,
		},
		{
			name:    "invalid URL",
			url:     "://invalid-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := core.IsGitHubURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidateGitHubURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name: "valid GitHub URL",
			url:  "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
		},
		{
			name:    "non-GitHub URL",
			url:     "https://example.com/file.zip",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "://invalid-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := core.ValidateGitHubURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
