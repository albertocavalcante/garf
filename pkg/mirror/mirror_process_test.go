package mirror_test

import (
	"testing"

	"github.com/albertocavalcante/garf/pkg/core/config"
	"github.com/albertocavalcante/garf/pkg/destinations"
	"github.com/albertocavalcante/garf/pkg/mirror"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestSetupDestinationSimplePath(t *testing.T) {
	logger := logrus.New()

	cfg := &config.Config{
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			URL:      "https://jfrog.example.com",
			User:     "user",
			Password: "password",
			DestPath: "generic-local",
		},
	}

	dest, err := mirror.SetupDestination(logger, cfg)
	require.NoError(t, err)

	jfrogDest, ok := dest.(*destinations.JFrogDestination)
	require.True(t, ok)

	destConfig := jfrogDest.GetConfig()
	require.Equal(t, "generic-local", destConfig.DestPath)
}

func TestSetupDestinationNestedPath(t *testing.T) {
	logger := logrus.New()

	cfg := &config.Config{
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			URL:      "https://jfrog.example.com",
			User:     "user",
			Password: "password",
			DestPath: "generic/sandbox-mirror",
		},
	}

	dest, err := mirror.SetupDestination(logger, cfg)
	require.NoError(t, err)

	jfrogDest, ok := dest.(*destinations.JFrogDestination)
	require.True(t, ok)

	destConfig := jfrogDest.GetConfig()
	require.Equal(t, "generic/sandbox-mirror", destConfig.DestPath)
}

func TestSetupDestinationInvalidType(t *testing.T) {
	logger := logrus.New()

	cfg := &config.Config{
		Destination: config.DestinationConfig{
			Type: "invalid",
			URL:  "generic-local",
		},
	}

	_, err := mirror.SetupDestination(logger, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported destination type")
}

func TestSetupDestinationMissingURL(t *testing.T) {
	logger := logrus.New()

	cfg := &config.Config{
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			User:     "user",
			Password: "password",
			DestPath: "generic-local",
		},
	}

	_, err := mirror.SetupDestination(logger, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "JFrog URL cannot be empty")
}

func TestSetupDestinationInvalidURL(t *testing.T) {
	logger := logrus.New()

	cfg := &config.Config{
		Destination: config.DestinationConfig{
			Type:     "jfrog",
			URL:      "://invalid-url",
			User:     "user",
			Password: "password",
			DestPath: "generic-local",
		},
	}

	_, err := mirror.SetupDestination(logger, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid JFrog URL")
}

func TestSetupDestinationJFrogPathHandling(t *testing.T) {
	logger := logrus.New()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid jfrog config with separate URL and destination path",
			config: &config.Config{
				Destination: config.DestinationConfig{
					Type:     "jfrog",
					URL:      "https://art.io/artifactory",
					User:     "user",
					Password: "pass",
					DestPath: "generic-local",
				},
			},
			expectError: false,
		},
		{
			name: "missing destination path",
			config: &config.Config{
				Destination: config.DestinationConfig{
					Type:     "jfrog",
					URL:      "https://art.io/artifactory",
					User:     "user",
					Password: "pass",
				},
			},
			expectError: true,
			errorMsg:    "JFrog destination path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest, err := mirror.SetupDestination(logger, tt.config)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, dest)
		})
	}
}
