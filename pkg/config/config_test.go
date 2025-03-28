package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name: "valid config",
			config: `
source:
  type: github
  url: https://github.com/example/repo/releases/download/v1.0.0/artifact.zip
destination:
  type: jfrog
  url: https://artifactory.example.com/artifactory
  user: ${JFROG_USER}
  password: ${JFROG_PASSWORD}
logLevel: info
concurrent: 4
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Equal(t, "github", cfg.Source.Type)
				require.Equal(t, "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip", cfg.Source.URL)
				require.Equal(t, "jfrog", cfg.Destination.Type)
				require.Equal(t, "https://artifactory.example.com/artifactory", cfg.Destination.URL)
				require.Equal(t, "info", cfg.LogLevel)
				require.Equal(t, 4, cfg.Concurrent)
			},
		},
		{
			name: "invalid yaml",
			config: `
source:
  type: github
  url: https://github.com/example/repo/releases/download/v1.0.0/artifact.zip
destination:
  type: jfrog
  url: https://artifactory.example.com/artifactory
  user: ${JFROG_USER}
  password: ${JFROG_PASSWORD}
logLevel: info
concurrent: invalid
`,
			wantErr: true,
		},
		{
			name: "missing required fields",
			config: `
source:
  type: github
destination:
  type: jfrog
logLevel: info
concurrent: 4
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpfile.Name())

			// Write config content
			_, err = tmpfile.Write([]byte(tt.config))
			require.NoError(t, err)

			// Load config
			cfg, err := Load(tmpfile.Name())
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			tt.validate(t, cfg)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Source: SourceConfig{
					Type: "github",
					URL:  "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip",
				},
				Destination: DestinationConfig{
					Type:     "jfrog",
					URL:      "https://artifactory.example.com/artifactory",
					User:     "user",
					Password: "password",
				},
				LogLevel:   "info",
				Concurrent: 4,
			},
			wantErr: false,
		},
		{
			name: "invalid source type",
			config: &Config{
				Source: SourceConfig{
					Type: "invalid",
					URL:  "https://example.com",
				},
				Destination: DestinationConfig{
					Type:     "jfrog",
					URL:      "https://artifactory.example.com/artifactory",
					User:     "user",
					Password: "password",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid destination type",
			config: &Config{
				Source: SourceConfig{
					Type: "github",
					URL:  "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip",
				},
				Destination: DestinationConfig{
					Type:     "invalid",
					URL:      "https://artifactory.example.com/artifactory",
					User:     "user",
					Password: "password",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Source: SourceConfig{
					Type: "github",
					URL:  "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip",
				},
				Destination: DestinationConfig{
					Type:     "jfrog",
					URL:      "https://artifactory.example.com/artifactory",
					User:     "user",
					Password: "password",
				},
				LogLevel: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid concurrent value",
			config: &Config{
				Source: SourceConfig{
					Type: "github",
					URL:  "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip",
				},
				Destination: DestinationConfig{
					Type:     "jfrog",
					URL:      "https://artifactory.example.com/artifactory",
					User:     "user",
					Password: "password",
				},
				Concurrent: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
