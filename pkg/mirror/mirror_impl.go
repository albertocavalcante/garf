package mirror

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/albertocavalcante/garf/pkg/config"
	"github.com/albertocavalcante/garf/pkg/core"
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
func (m *DefaultMirror) processArtifact(ctx context.Context, artifact *core.Artifact, opts *core.MirrorOptions) MirrorResult {
	// Get the first available source and destination
	var source core.Source

	var destination core.Destination

	m.mu.RLock()
	for _, s := range m.sources {
		source = s

		break
	}

	for _, d := range m.destinations {
		destination = d

		break
	}
	m.mu.RUnlock()

	if source == nil {
		return MirrorResult{
			Artifact: artifact,
			Error:    fmt.Errorf("no source available"),
		}
	}

	// Skip destination check if in dry run mode
	if !opts.DryRun && destination == nil {
		return MirrorResult{
			Artifact: artifact,
			Error:    fmt.Errorf("no destination available"),
		}
	}

	// Download the artifact
	content, err := source.Get(ctx, artifact)
	if err != nil {
		return MirrorResult{
			Artifact: artifact,
			Error:    fmt.Errorf("failed to get artifact: %w", err),
		}
	}
	defer content.Close()

	// If in dry run mode, skip upload
	if opts.DryRun {
		if opts.DryRunMode == "all" {
			m.logger.WithFields(logrus.Fields{
				"artifact": artifact.Name,
				"mode":     opts.DryRunMode,
			}).Info("Dry run: skipping upload")

			return MirrorResult{
				Artifact: artifact,
			}
		}
		// For "upload" mode, we still want to process the artifact but skip the actual upload
		m.logger.WithFields(logrus.Fields{
			"artifact": artifact.Name,
			"mode":     opts.DryRunMode,
		}).Info("Dry run: simulating upload")

		return MirrorResult{
			Artifact: artifact,
		}
	}

	// Upload the artifact
	if err := destination.Put(ctx, artifact, content); err != nil {
		return MirrorResult{
			Artifact: artifact,
			Error:    fmt.Errorf("failed to put artifact: %w", err),
		}
	}

	return MirrorResult{
		Artifact: artifact,
	}
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
	for i := 0; i < numWorkers; i++ {
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
