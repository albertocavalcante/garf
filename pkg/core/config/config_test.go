package config_test

import (
	"os"
	"testing"

	"github.com/albertocavalcante/garf/pkg/core/config"
	"github.com/stretchr/testify/require"
)

const (
	invalidType = "invalid"
	validConfig = `
source:
  type: github
  url: https://github.com/example/repo/releases/download/v1.0.0/artifact.zip
destination:
  type: jfrog
  url: https://artifactory.example.com/artifactory
  user: testuser
  password: testpass
log_level: info
concurrent: 4
`
	invalidConfig = `
source:
  type: github
  url: https://github.com/example/repo/releases/download/v1.0.0/artifact.zip
destination:
  type: jfrog
  url: https://artifactory.example.com/artifactory
  user: testuser
  password: testpass
log_level: info
concurrent: 4
invalid: yaml: :1:1
`
)

func createTempConfigFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpfile.Name()) })

	_, err = tmpfile.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	return tmpfile.Name()
}

func validateConfig(t *testing.T, cfg *config.Config) {
	require.Equal(t, "github", cfg.Source.Type)
	require.Equal(t, "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip", cfg.Source.URL)
	require.Equal(t, "jfrog", cfg.Destination.Type)
	require.Equal(t, "https://artifactory.example.com/artifactory", cfg.Destination.URL)
	require.Equal(t, "testuser", cfg.Destination.User)
	require.Equal(t, "testpass", cfg.Destination.Password)
	require.Equal(t, "info", cfg.LogLevel)
	require.Equal(t, 4, cfg.Concurrent)
}

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		configPath := createTempConfigFile(t, validConfig)
		cfg, err := config.Load(configPath)
		require.NoError(t, err)
		validateConfig(t, cfg)
	})

	t.Run("invalid config", func(t *testing.T) {
		configPath := createTempConfigFile(t, invalidConfig)
		cfg, err := config.Load(configPath)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}

func createValidConfig() *config.Config {
	return &config.Config{
		Source: config.SourceConfig{
			Type: "github",
			URL:  "https://github.com/example/repo/releases/download/v1.0.0/artifact.zip",
		},
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			URL:      "https://artifactory.example.com/artifactory",
			User:     "testuser",
			Password: "testpass",
		},
		LogLevel:   "info",
		Concurrent: 4,
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := createValidConfig()
		require.NoError(t, cfg.Validate())
	})

	t.Run("invalid source type", func(t *testing.T) {
		cfg := createValidConfig()
		cfg.Source.Type = invalidType
		require.Error(t, cfg.Validate())
	})

	t.Run("invalid destination type", func(t *testing.T) {
		cfg := createValidConfig()
		cfg.Destination.Type = invalidType
		require.Error(t, cfg.Validate())
	})

	t.Run("invalid log level", func(t *testing.T) {
		cfg := createValidConfig()
		cfg.LogLevel = invalidType
		require.Error(t, cfg.Validate())
	})

	t.Run("invalid concurrent value", func(t *testing.T) {
		cfg := createValidConfig()
		cfg.Concurrent = 0
		require.Error(t, cfg.Validate())
	})
}
