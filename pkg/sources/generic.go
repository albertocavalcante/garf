// Package sources provides implementations of the core.Source interface.
package sources

import (
	"context"
	"fmt"
	"io"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/sirupsen/logrus"
)

// GenericSource implements core.Source for generic HTTP URLs.
// It doesn't perform any special validation or processing - it just downloads from any HTTP/HTTPS URL.
type GenericSource struct {
	*HTTPSource
}

// NewGenericSource creates a new GenericSource instance.
func NewGenericSource(logger *logrus.Logger) *GenericSource {
	return &GenericSource{
		HTTPSource: NewHTTPSource(logger, "generic"),
	}
}

// List is not implemented for generic source as it requires specific URLs.
func (s *GenericSource) List(ctx context.Context) ([]*core.Artifact, error) {
	return nil, fmt.Errorf("listing artifacts from generic source is not supported, use specific URLs")
}

// Get retrieves an artifact from any HTTP/HTTPS URL.
func (s *GenericSource) Get(ctx context.Context, artifact *core.Artifact) (io.ReadCloser, error) {
	if artifact == nil {
		return nil, fmt.Errorf("artifact cannot be nil")
	}

	logger := s.GetLogger().WithFields(logrus.Fields{
		"source":        "generic",
		"url":           artifact.Location,
		"artifact_name": artifact.Name,
	})

	logger.Info("Starting generic artifact download")

	if err := s.EnsureTempDir(); err != nil {
		logger.WithError(err).Error("Failed to create temporary directory")
		return nil, err
	}

	logger.Debug("Creating HTTP request for generic download")

	req, err := CreateHTTPRequest(ctx, artifact.Location)
	if err != nil {
		logger.WithError(err).Error("Failed to create HTTP request")
		return nil, err
	}

	resp, err := s.DoRequest(req)
	if err != nil {
		logger.WithError(err).Error("Failed to download artifact")
		return nil, err
	}
	defer resp.Body.Close()

	reader, err := s.DownloadToTempFile(resp)
	if err != nil {
		logger.WithError(err).Error("Failed to save artifact to temporary file")
		return nil, err
	}

	logger.Info("Successfully completed generic artifact download")

	return reader, nil
}

// Validate checks if the source is properly configured.
func (s *GenericSource) Validate() error {
	return nil
}
