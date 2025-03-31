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
	result := MirrorResult{
		Artifact: artifact,
	}

	if err := m.validateArtifact(artifact); err != nil {
		result.Error = err

		return result
	}

	if err := m.downloadAndUploadArtifact(ctx, artifact, opts, &result); err != nil {
		result.Error = err

		return result
	}

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

	if len(m.destinations) == 0 {
		return fmt.Errorf("no destination available")
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

func (m *DefaultMirror) handleDryRun(artifact *core.Artifact, opts *core.MirrorOptions) bool {
	if opts != nil && opts.DryRun {
		if opts.DryRunMode == "all" {
			m.logger.WithFields(logrus.Fields{
				"artifact": artifact.Name,
				"mode":     opts.DryRunMode,
			}).Info("Dry run: skipping upload")

			return true
		}

		m.logger.WithFields(logrus.Fields{
			"artifact": artifact.Name,
			"mode":     opts.DryRunMode,
		}).Info("Dry run: simulating upload")

		return false
	}

	return false
}

func (m *DefaultMirror) uploadToDestinations(
	ctx context.Context,
	artifact *core.Artifact,
	content io.Reader,
	destinations []core.Destination,
) error {
	var wg sync.WaitGroup

	errChan := make(chan error, len(destinations))

	for _, dest := range destinations {
		wg.Add(1)

		go func(d core.Destination) {
			defer wg.Done()

			if err := d.Put(ctx, artifact, content); err != nil {
				errChan <- fmt.Errorf("failed to upload to destination: %w", err)

				return
			}
		}(dest)
	}

	wg.Wait()
	close(errChan)

	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	return lastErr
}

func (m *DefaultMirror) downloadAndUploadArtifact(
	ctx context.Context,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
	_ *MirrorResult,
) error {
	source, destinations := m.getSourceAndDestinations()

	content, err := source.Get(ctx, artifact)
	if err != nil {
		return fmt.Errorf("failed to get artifact: %w", err)
	}
	defer content.Close()

	if m.handleDryRun(artifact, opts) {
		return nil
	}

	// Buffer the content before concurrent uploads
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, content); err != nil {
		return fmt.Errorf("failed to buffer content: %w", err)
	}

	// Create a new reader for each destination
	readers := make([]io.Reader, len(destinations))
	for i := range destinations {
		readers[i] = bytes.NewReader(buf.Bytes())
	}

	var wg sync.WaitGroup

	errChan := make(chan error, len(destinations))

	for i, dest := range destinations {
		wg.Add(1)

		go func(d core.Destination, r io.Reader) {
			defer wg.Done()

			if err := d.Put(ctx, artifact, r); err != nil {
				errChan <- fmt.Errorf("failed to upload to destination: %w", err)
			}
		}(dest, readers[i])
	}

	wg.Wait()
	close(errChan)

	var lastErr error
	for err := range errChan {
		lastErr = err
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
func (m *DefaultMirror) SetupSource(logger *logrus.Logger, config *config.Config) (config.Source, error) {
	return SetupSource(logger, config)
}

// SetupDestination creates and configures a destination based on the provided configuration.
func (m *DefaultMirror) SetupDestination(logger *logrus.Logger, config *config.Config) (config.Destination, error) {
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
