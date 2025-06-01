package mirror

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/albertocavalcante/garf/pkg/core"
	"github.com/albertocavalcante/garf/pkg/core/config"
	"github.com/sirupsen/logrus"
)

const (
	// defaultDirPerm is the default permission for created directories.
	defaultDirPerm = 0o755
	// maxExtractedFileSize is the maximum size allowed for extracted files (100MB).
	maxExtractedFileSize = 100 * 1024 * 1024
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

		// Build destination paths for dry-run feedback even when skipping everything
		if len(destinations) > 0 {
			var destinationPaths []string

			for _, dest := range destinations {
				raw := false
				if opts != nil {
					raw = opts.Raw
				}

				destPath, err := dest.BuildDestinationPath(artifact, raw)
				if err != nil {
					logger.WithError(err).Warn("Failed to build destination path for dry-run")
				} else {
					destinationPaths = append(destinationPaths, destPath)
				}
			}

			// Set the first destination path in the result for logging
			if len(destinationPaths) > 0 {
				result.DestinationPath = destinationPaths[0]
				logger.WithField("destination_path", result.DestinationPath).Info("Would upload to destination")
			}
		}

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

	// Handle ZIP extraction if needed
	finalContent, finalArtifact, err := m.processContentForUpload(ctx, content, artifact, opts, m.logger)
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}
	defer finalContent.Close()

	// Check if we're in upload-only dry-run mode
	if opts != nil && opts.DryRun && opts.DryRunMode == "upload" {
		logger.Info("Dry run mode 'upload' - skipping upload only")

		// Build destination paths for dry-run feedback
		var destinationPaths []string

		for _, dest := range destinations {
			raw := false
			if opts != nil {
				raw = opts.Raw
			}

			destPath, err := dest.BuildDestinationPath(finalArtifact, raw)
			if err != nil {
				logger.WithError(err).Warn("Failed to build destination path for dry-run")
			} else {
				destinationPaths = append(destinationPaths, destPath)
			}
		}

		// Set the first destination path in the result for logging
		if len(destinationPaths) > 0 {
			result.DestinationPath = destinationPaths[0]
			logger.WithField("destination_path", result.DestinationPath).Info("Would upload to destination")
		}

		return nil
	}

	logger.Info("Buffering artifact content for upload")
	// Buffer the content before concurrent uploads
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, finalContent); err != nil {
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
	destinationPathChan := make(chan string, len(destinations))

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

			destinationPath, err := d.Put(ctx, finalArtifact, r, raw)
			if err != nil {
				destLogger.WithError(err).Error("Failed to upload to destination")
				errChan <- fmt.Errorf("failed to upload to destination: %w", err)
			} else {
				destLogger.WithField("destination_path", destinationPath).Info("Successfully uploaded to destination")
				destinationPathChan <- destinationPath
			}
		}(dest, readers[i], i)
	}

	wg.Wait()
	close(errChan)
	close(destinationPathChan)

	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	// Get the first successful destination path
	var destinationPath string
	for path := range destinationPathChan {
		if destinationPath == "" {
			destinationPath = path
		}
	}

	if lastErr != nil {
		logger.WithError(lastErr).Error("One or more uploads failed")
	} else {
		logger.WithField("destination_path", destinationPath).Info("All uploads completed successfully")
	}

	// Update the result artifact to reflect any changes (like from ZIP extraction)
	result.Artifact = finalArtifact
	// Set the destination path in the result
	result.DestinationPath = destinationPath

	return lastErr
}

// processContentForUpload handles ZIP extraction if needed and returns the final content and artifact.
func (m *DefaultMirror) processContentForUpload(
	ctx context.Context,
	content io.ReadCloser,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
	logger *logrus.Logger,
) (io.ReadCloser, *core.Artifact, error) {
	// Check if unzip is enabled and this is a ZIP file
	shouldUnzip := opts != nil && opts.Unzip && strings.HasSuffix(strings.ToLower(artifact.Location), ".zip")

	if !shouldUnzip {
		logger.Debug("No ZIP processing needed")

		return content, artifact, nil
	}

	logger.Info("ZIP file detected - starting extraction process")

	// Save ZIP content to temporary file
	tempZipPath, err := m.saveZipToTempFile(content, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save ZIP: %w", err)
	}
	defer os.Remove(tempZipPath) // Clean up the temp ZIP file

	// Extract the ZIP file
	extractedContent, extractedArtifact, err := m.extractZipContent(tempZipPath, artifact, opts, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract ZIP: %w", err)
	}

	return extractedContent, extractedArtifact, nil
}

// saveZipToTempFile saves the ZIP content to a temporary file and returns the path.
func (m *DefaultMirror) saveZipToTempFile(content io.ReadCloser, logger *logrus.Logger) (string, error) {
	// Create temporary file to save the downloaded ZIP
	tempFile, err := os.CreateTemp("", "garf-zip-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	logger.WithField("temp_zip_file", tempFile.Name()).Debug("Created temporary file for ZIP")

	// Copy the downloaded content to the temporary file
	if _, err := io.Copy(tempFile, content); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())

		return "", fmt.Errorf("failed to save ZIP to temporary file: %w", err)
	}

	// Close the temp file so we can read from it
	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())

		return "", fmt.Errorf("failed to close temporary ZIP file: %w", err)
	}

	logger.WithField("temp_zip_file", tempFile.Name()).Info("Successfully saved ZIP to temporary file")

	return tempFile.Name(), nil
}

// extractZipContent extracts content from a ZIP file and returns the extracted content and updated artifact.
func (m *DefaultMirror) extractZipContent(
	tempZipPath string,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
	logger *logrus.Logger,
) (io.ReadCloser, *core.Artifact, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "garf-unzip-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logger.WithField("temp_dir", tempDir).Debug("Created temporary directory for ZIP extraction")

	// Extract the single file from ZIP
	extractedPath, extractedName, err := m.extractSingleFileFromZip(tempZipPath, tempDir, logger)
	if err != nil {
		os.RemoveAll(tempDir)

		return nil, nil, fmt.Errorf("failed to extract ZIP: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"original_zip":   tempZipPath,
		"extracted_path": extractedPath,
		"extracted_name": extractedName,
		"temp_dir":       tempDir,
	}).Info("Successfully extracted ZIP file")

	// Open the extracted file for reading
	extractedFile, err := os.Open(extractedPath)
	if err != nil {
		os.RemoveAll(tempDir)

		return nil, nil, fmt.Errorf("failed to open extracted file: %w", err)
	}

	// Determine the final artifact name
	finalName := extractedName
	if opts != nil && opts.PreserveZipName {
		finalName = m.buildPreservedZipName(artifact.Name, extractedName)
		logger.WithFields(logrus.Fields{
			"original_zip_name": artifact.Name,
			"extracted_name":    extractedName,
			"preserved_name":    finalName,
		}).Info("Preserving ZIP filename with extracted file extension")
	}

	// Create a new artifact with the extracted file information
	extractedArtifact := &core.Artifact{
		Name:     finalName,
		Location: artifact.Location, // Keep original location for coordinate extraction
		Metadata: artifact.Metadata,
		Version:  artifact.Version,
	}

	logger.WithFields(logrus.Fields{
		"original_name":     artifact.Name,
		"extracted_name":    extractedName,
		"final_name":        finalName,
		"original_location": artifact.Location,
	}).Info("Updated artifact information with extracted file")

	// Return a custom ReadCloser that cleans up the temp directory when closed
	finalContent := &tempDirCleanupReader{
		ReadCloser: extractedFile,
		tempDir:    tempDir,
		logger:     logger,
	}

	return finalContent, extractedArtifact, nil
}

// extractSingleFileFromZip extracts a single file from a ZIP archive.
// Returns the path to the extracted file and the filename.
func (m *DefaultMirror) extractSingleFileFromZip(
	zipPath, destDir string,
	logger *logrus.Logger,
) (string, string, error) {
	// Open the ZIP file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	// Find the first non-directory file
	var targetFile *zip.File

	for _, f := range reader.File {
		if !f.FileInfo().IsDir() {
			if targetFile != nil {
				return "", "", fmt.Errorf("ZIP file contains multiple files, expected exactly one")
			}

			targetFile = f
		}
	}

	if targetFile == nil {
		return "", "", fmt.Errorf("no files found in ZIP archive")
	}

	// Check file size to prevent ZIP bombs
	if targetFile.UncompressedSize64 > maxExtractedFileSize {
		return "", "", fmt.Errorf("file too large: %d bytes (max allowed: %d bytes)",
			targetFile.UncompressedSize64, maxExtractedFileSize)
	}

	logger.WithField("zip_file_name", targetFile.Name).Debug("Found file in ZIP archive")

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, defaultDirPerm); err != nil {
		return "", "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Get the base filename (remove any path components)
	fileName := filepath.Base(targetFile.Name)
	destPath := filepath.Join(destDir, fileName)

	// Open the file inside the ZIP
	rc, err := targetFile.Open()
	if err != nil {
		return "", "", fmt.Errorf("failed to open file inside ZIP: %w", err)
	}
	defer rc.Close()

	// Create the destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	// Copy the file contents with size limit protection
	limitedReader := io.LimitReader(rc, maxExtractedFileSize+1) // +1 to detect oversized files

	bytesWritten, err := io.Copy(outFile, limitedReader)
	if err != nil {
		return "", "", fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Check if the file exceeded the size limit during extraction
	if bytesWritten > maxExtractedFileSize {
		return "", "", fmt.Errorf("file exceeded size limit during extraction: %d bytes", bytesWritten)
	}

	logger.WithFields(logrus.Fields{
		"zip_file_name": targetFile.Name,
		"extracted_to":  destPath,
		"file_size":     targetFile.UncompressedSize64,
	}).Debug("Successfully extracted file from ZIP")

	return destPath, fileName, nil
}

// buildPreservedZipName constructs a filename that preserves the ZIP name but uses the extracted file's extension.
func (m *DefaultMirror) buildPreservedZipName(zipName, extractedName string) string {
	// Remove .zip extension from the ZIP name
	zipBaseName := strings.TrimSuffix(zipName, ".zip")
	zipBaseName = strings.TrimSuffix(zipBaseName, ".ZIP") // Handle uppercase too

	// Extract all extensions from the extracted file name
	// For files like "file.tar.gz", we want to preserve ".tar.gz", not just ".gz"
	extractedBaseName := extractedName

	var extensions []string

	// Keep extracting extensions until we can't find any more
	for {
		ext := filepath.Ext(extractedBaseName)
		if ext == "" {
			break
		}

		extensions = append([]string{ext}, extensions...) // Prepend to maintain order
		extractedBaseName = strings.TrimSuffix(extractedBaseName, ext)
	}

	// If the extracted file has no extensions, return the ZIP base name as-is
	if len(extensions) == 0 {
		return zipBaseName
	}

	// Combine the ZIP base name with all the extracted file's extensions
	return zipBaseName + strings.Join(extensions, "")
}

// tempDirCleanupReader wraps a ReadCloser and cleans up a temporary directory when closed.
type tempDirCleanupReader struct {
	io.ReadCloser
	tempDir string
	logger  *logrus.Logger
}

func (r *tempDirCleanupReader) Close() error {
	// Close the underlying reader first
	err := r.ReadCloser.Close()

	// Clean up the temporary directory
	if cleanupErr := os.RemoveAll(r.tempDir); cleanupErr != nil {
		if r.logger != nil {
			r.logger.WithError(cleanupErr).WithField("temp_dir", r.tempDir).Warn("Failed to clean up temporary directory")
		}
		// Don't override the original error if there was one
		if err == nil {
			err = cleanupErr
		}
	} else if r.logger != nil {
		r.logger.WithField("temp_dir", r.tempDir).Debug("Cleaned up temporary directory")
	}

	return err
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
