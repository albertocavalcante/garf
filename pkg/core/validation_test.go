package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSourcePathStrip(t *testing.T) {
	tests := []struct {
		name            string
		sourcePathStrip string
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "valid path strip",
			sourcePathStrip: "artifactory.corp.net/staging/",
			wantErr:         false,
		},
		{
			name:            "empty path strip",
			sourcePathStrip: "",
			wantErr:         false,
		},
		{
			name:            "path traversal attack",
			sourcePathStrip: "../../malicious",
			wantErr:         true,
			errMsg:          "source path strip cannot contain '..' for security reasons",
		},
		{
			name:            "https scheme",
			sourcePathStrip: "https://malicious.com/",
			wantErr:         true,
			errMsg:          "source path strip should not include the URL scheme",
		},
		{
			name:            "http scheme",
			sourcePathStrip: "http://malicious.com/",
			wantErr:         true,
			errMsg:          "source path strip should not include the URL scheme",
		},
		{
			name:            "path strip with trailing dots",
			sourcePathStrip: "valid/path/..",
			wantErr:         true,
			errMsg:          "source path strip cannot contain '..' for security reasons",
		},
		{
			name:            "path strip with spaces (should be trimmed)",
			sourcePathStrip: "  valid/path  ",
			wantErr:         false,
		},
		{
			name:            "path strip with backslashes",
			sourcePathStrip: "valid\\path\\with\\backslashes",
			wantErr:         true,
			errMsg:          "source path strip should not contain backslashes",
		},
		{
			name:            "valid host only",
			sourcePathStrip: "artifactory.corp.net",
			wantErr:         false,
		},
		{
			name:            "valid path with subdirectories",
			sourcePathStrip: "artifactory.corp.net/staging/temp/",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSourcePathStrip(tt.sourcePathStrip)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateDryRunMode(t *testing.T) {
	tests := []struct {
		name       string
		dryRunMode string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid mode: all",
			dryRunMode: "all",
			wantErr:    false,
		},
		{
			name:       "valid mode: upload",
			dryRunMode: "upload",
			wantErr:    false,
		},
		{
			name:       "empty mode (valid)",
			dryRunMode: "",
			wantErr:    false,
		},
		{
			name:       "invalid mode",
			dryRunMode: "invalid",
			wantErr:    true,
			errMsg:     "invalid dry run mode: invalid. Valid modes are: all, upload",
		},
		{
			name:       "case sensitive - ALL",
			dryRunMode: "ALL",
			wantErr:    true,
			errMsg:     "invalid dry run mode: ALL. Valid modes are: all, upload",
		},
		{
			name:       "case sensitive - Upload",
			dryRunMode: "Upload",
			wantErr:    true,
			errMsg:     "invalid dry run mode: Upload. Valid modes are: all, upload",
		},
		{
			name:       "random string",
			dryRunMode: "random",
			wantErr:    true,
			errMsg:     "invalid dry run mode: random. Valid modes are: all, upload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDryRunMode(tt.dryRunMode)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
