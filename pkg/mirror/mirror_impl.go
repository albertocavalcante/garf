package mirror

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/core/config"
	"github.com/sirupsen/logrus"
)

// DefaultMirror implements the Mirror interface.
type DefaultMirror struct {
	logger       *logrus.Logger
	sources      map[string]core.Source
	destinations map[string]core.Destination
	mu           sync.RWMutex
}

// NewDefaultMirror creates a new DefaultMirror instance.
func NewDefaultMirror(logger *logrus.Logger) *DefaultMirror {
	if logger == nil {
		logger = logrus.New()
	}

	return &DefaultMirror{
		logger:       logger,
		sources:      make(map[string]core.Source),
		destinations: make(map[string]core.Destination),
	}
}

// AddSource adds a new source to the mirror.
func (m *DefaultMirror) AddSource(name string, source core.Source) error {
	if name == "" {
		return fmt.Errorf("source name is required")
	}

	if source == nil {
		return fmt.Errorf("source is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sources[name]; exists {
		return fmt.Errorf("source %s already exists", name)
	}

	m.sources[name] = source

	return nil
}

// AddDestination adds a new destination to the mirror.
func (m *DefaultMirror) AddDestination(name string, destination core.Destination) error {
	if name == "" {
		return fmt.Errorf("destination name is required")
	}

	if destination == nil {
		return fmt.Errorf("destination is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.destinations[name]; exists {
		return fmt.Errorf("destination %s already exists", name)
	}

	m.destinations[name] = destination

	return nil
}

// processArtifact processes a single artifact using the first available source and destination.
func (m *DefaultMirror) processArtifact(
	ctx context.Context,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
) MirrorResult {
	logger := m.logger.WithFields(logrus.Fields{
		"artifact_name":     artifact.Name,
		"artifact_location": artifact.Location,
	})

	logger.Info("Starting artifact processing")

	result := MirrorResult{
		Artifact: artifact,
	}

	if err := m.validateArtifact(artifact); err != nil {
		logger.WithError(err).Error("Artifact validation failed")
		result.Error = err

		return result
	}

	logger.Debug("Artifact validation passed")

	if err := m.downloadAndUploadArtifact(ctx, artifact, opts, &result); err != nil {
		logger.WithError(err).Error("Download and upload failed")
		result.Error = err

		return result
	}

	logger.Info("Successfully processed artifact")

	return result
}

func (m *DefaultMirror) validateArtifact(artifact *core.Artifact) error {
	if artifact == nil {
		return fmt.Errorf("artifact is nil")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.sources) == 0 {
		return fmt.Errorf("no source available")
	}

	// Allow no destinations only in dry-run mode with "all" mode
	// This will be checked later in the dry-run handling
	if len(m.destinations) == 0 {
		m.logger.Debug("No destinations available - this is only valid in dry-run mode")
	}

	return nil
}

func (m *DefaultMirror) getSourceAndDestinations() (core.Source, []core.Destination) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var source core.Source

	destinations := make([]core.Destination, 0, len(m.destinations))

	for _, s := range m.sources {
		source = s

		break
	}

	for _, d := range m.destinations {
		destinations = append(destinations, d)
	}

	return source, destinations
}

func (m *DefaultMirror) downloadAndUploadArtifact(
	ctx context.Context,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
	result *MirrorResult,
) error {
	logger := m.logger.WithFields(logrus.Fields{
		"artifact_name":     artifact.Name,
		"artifact_location": artifact.Location,
	})

	logger.Debug("Getting source and destinations")

	source, destinations := m.getSourceAndDestinations()

	logger.WithFields(logrus.Fields{
		"num_destinations": len(destinations),
		"has_source":       source != nil,
	}).Debug("Retrieved source and destinations")

	// Check if we're in dry-run mode that skips everything
	if opts != nil && opts.DryRun && opts.DryRunMode == "all" {
		logger.Info("Dry run mode 'all' - skipping download and upload")

		return nil
	}

	// For other modes, we need destinations
	if len(destinations) == 0 {
		return fmt.Errorf("no destination available")
	}

	logger.Info("Starting artifact download from source")

	content, err := source.Get(ctx, artifact)
	if err != nil {
		logger.WithError(err).Error("Failed to download artifact from source")

		return fmt.Errorf("failed to get artifact: %w", err)
	}

	defer content.Close()

	logger.Info("Successfully downloaded artifact from source")

	// Check if we're in upload-only dry-run mode
	if opts != nil && opts.DryRun && opts.DryRunMode == "upload" {
		logger.Info("Dry run mode 'upload' - skipping upload only")

		return nil
	}

	logger.Info("Buffering artifact content for upload")
	// Buffer the content before concurrent uploads
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, content); err != nil {
		logger.WithError(err).Error("Failed to buffer artifact content")

		return fmt.Errorf("failed to buffer content: %w", err)
	}

	logger.WithField("buffer_size", buf.Len()).Debug("Successfully buffered artifact content")

	// Create a new reader for each destination
	readers := make([]io.Reader, len(destinations))
	for i := range destinations {
		readers[i] = bytes.NewReader(buf.Bytes())
	}

	logger.WithField("num_destinations", len(destinations)).Info("Starting concurrent uploads to destinations")

	var wg sync.WaitGroup

	errChan := make(chan error, len(destinations))

	for i, dest := range destinations {
		wg.Add(1)

		go func(d core.Destination, r io.Reader, index int) {
			defer wg.Done()

			destLogger := logger.WithField("destination_index", index)
			destLogger.Debug("Starting upload to destination")

			// Get raw value from options, defaulting to false if options is nil
			raw := false
			if opts != nil {
				raw = opts.Raw
			}

			if err := d.Put(ctx, artifact, r, raw); err != nil {
				destLogger.WithError(err).Error("Failed to upload to destination")
				errChan <- fmt.Errorf("failed to upload to destination: %w", err)
			} else {
				destLogger.Info("Successfully uploaded to destination")
			}
		}(dest, readers[i], i)
	}

	wg.Wait()
	close(errChan)

	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	if lastErr != nil {
		logger.WithError(lastErr).Error("One or more uploads failed")
	} else {
		logger.Info("All uploads completed successfully")
	}

	return lastErr
}

// Mirror copies artifacts from sources to destinations.
func (m *DefaultMirror) Mirror(
	ctx context.Context,
	artifacts []*core.Artifact,
	opts *core.MirrorOptions,
) <-chan MirrorResult {
	if len(artifacts) == 0 {
		return nil
	}

	// Create work channel
	work := make(chan *core.Artifact, len(artifacts))
	for _, artifact := range artifacts {
		work <- artifact
	}

	close(work)

	// Create results channel
	results := make(chan MirrorResult, len(artifacts))

	// Create worker pool
	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU()
	if opts != nil && opts.Concurrent > 0 {
		numWorkers = opts.Concurrent
	}

	// Start workers
	for range make([]struct{}, numWorkers) {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for artifact := range work {
				result := m.processArtifact(ctx, artifact, opts)
				results <- result
			}
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// SetupSource creates and configures a source based on the provided configuration.
func (m *DefaultMirror) SetupSource(logger *logrus.Logger, config *config.Config) (core.Source, error) {
	return SetupSource(logger, config)
}

// SetupDestination creates and configures a destination based on the provided configuration.
func (m *DefaultMirror) SetupDestination(logger *logrus.Logger, config *config.Config) (core.Destination, error) {
	return SetupDestination(logger, config)
}

// ProcessMirrorResults processes the results from the mirror operation.
func (m *DefaultMirror) ProcessMirrorResults(logger *logrus.Logger, results <-chan MirrorResult) error {
	return ProcessMirrorResults(logger, results)
}

// GetSource retrieves a source by name.
func (m *DefaultMirror) GetSource(name string) (core.Source, error) {
	if name == "" {
		return nil, fmt.Errorf("source name is required")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	source, exists := m.sources[name]
	if !exists {
		return nil, fmt.Errorf("source %s not found", name)
	}

	return source, nil
}

// GetDestination retrieves a destination by name.
func (m *DefaultMirror) GetDestination(name string) (core.Destination, error) {
	if name == "" {
		return nil, fmt.Errorf("destination name is required")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	destination, exists := m.destinations[name]
	if !exists {
		return nil, fmt.Errorf("destination %s not found", name)
	}

	return destination, nil
}
