package urlprocessor_test

import (
	"net/url"
	"testing"

	"github.com/albertocavalcante/garf/pkg/urlprocessor"
)

func TestGitHubProcessor(t *testing.T) {
	processor := &urlprocessor.GitHubProcessor{}

	tests := []struct {
		name     string
		urlStr   string
		raw      bool
		expected string
	}{
		{
			name:     "GitHub release URL with raw=true",
			urlStr:   "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
			raw:      true,
			expected: "github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
		},
		{
			name:     "GitHub release URL with raw=false",
			urlStr:   "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
			raw:      false,
			expected: "github.com/bazelbuild/bazel/7.2.1/bazel-7.2.1-windows-x86_64.exe",
		},
		{
			name:     "Non-release GitHub URL",
			urlStr:   "https://github.com/bazelbuild/bazel/blob/master/README.md",
			raw:      false,
			expected: "README.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.urlStr)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			if !processor.CanProcess(parsed) {
				t.Fatalf("Processor should be able to handle GitHub URLs")
			}

			result := processor.Process(parsed, tt.raw)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := urlprocessor.NewRegistry()

	tests := []struct {
		name     string
		urlStr   string
		raw      bool
		expected string
	}{
		{
			name:     "GitHub URL",
			urlStr:   "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
			raw:      true,
			expected: "github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
		},
		{
			name:     "Non-GitHub URL",
			urlStr:   "https://example.com/file.zip",
			raw:      true,
			expected: "file.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.ProcessURLString(tt.urlStr, tt.raw)
			if err != nil {
				t.Fatalf("Failed to process URL: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
