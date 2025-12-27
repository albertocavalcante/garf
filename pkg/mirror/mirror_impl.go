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
	logger := m.createArtifactLogger(artifact)
	source, destinations := m.getSourceAndDestinations()

	logger.WithFields(logrus.Fields{
		"num_destinations": len(destinations),
		"has_source":       source != nil,
	}).Debug("Retrieved source and destinations")

	// Handle dry-run modes
	if opts != nil && opts.DryRun {
		return m.handleDryRunMode(opts, destinations, artifact, result, logger)
	}

	// For non-dry-run modes, we need destinations
	if len(destinations) == 0 {
		return fmt.Errorf("no destination available")
	}

	// Download and process content
	finalContent, finalArtifact, err := m.downloadAndProcessContent(ctx, source, artifact, opts, logger)
	if err != nil {
		return err
	}
	defer finalContent.Close()

	// Upload to all destinations
	return m.uploadToDestinations(ctx, finalContent, finalArtifact, destinations, opts, result, logger)
}

// createArtifactLogger creates a logger with artifact context.
func (m *DefaultMirror) createArtifactLogger(artifact *core.Artifact) *logrus.Entry {
	return m.logger.WithFields(logrus.Fields{
		"artifact_name":     artifact.Name,
		"artifact_location": artifact.Location,
	})
}

// handleDryRunMode handles different dry-run modes.
func (m *DefaultMirror) handleDryRunMode(
	opts *core.MirrorOptions,
	destinations []core.Destination,
	artifact *core.Artifact,
	result *MirrorResult,
	logger *logrus.Entry,
) error {
	logger.Infof("Dry run mode '%s' - skipping operations", opts.DryRunMode)

	if len(destinations) > 0 {
		destinationPaths := m.buildDestinationPaths(destinations, artifact, opts, logger)
		if len(destinationPaths) > 0 {
			result.DestinationPath = destinationPaths[0]
			logger.WithField("destination_path", result.DestinationPath).Info("Would upload to destination")
		}
	}

	return nil
}

// buildDestinationPaths builds destination paths for dry-run feedback.
func (m *DefaultMirror) buildDestinationPaths(
	destinations []core.Destination,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
	logger *logrus.Entry,
) []string {
	var destinationPaths []string

	for _, dest := range destinations {
		raw := opts != nil && opts.Raw

		destPath, err := dest.BuildDestinationPath(artifact, raw)
		if err != nil {
			logger.WithError(err).Warn("Failed to build destination path for dry-run")
		} else {
			destinationPaths = append(destinationPaths, destPath)
		}
	}

	return destinationPaths
}

// downloadAndProcessContent downloads and processes content.
func (m *DefaultMirror) downloadAndProcessContent(
	ctx context.Context,
	source core.Source,
	artifact *core.Artifact,
	opts *core.MirrorOptions,
	logger *logrus.Entry,
) (io.ReadCloser, *core.Artifact, error) {
	logger.Info("Starting artifact download from source")

	content, err := source.Get(ctx, artifact)
	if err != nil {
		logger.WithError(err).Error("Failed to download artifact from source")

		return nil, nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	logger.Info("Successfully downloaded artifact from source")

	// Verify checksum if provided
	if opts != nil && opts.Checksum != "" {
		logger.WithField("expected_checksum", opts.Checksum).Info("Verifying checksum")
		content = NewChecksumValidatingReader(content, opts.Checksum)
	}

	// Handle ZIP extraction if needed
	finalContent, finalArtifact, err := m.processContentForUpload(ctx, content, artifact, opts, m.logger)
	if err != nil {
		content.Close() // Close the original content on error

		return nil, nil, fmt.Errorf("failed to process content: %w", err)
	}

	// If no processing was needed, content == finalContent, so don't close content here
	// The caller will close finalContent which may be the same as content
	return finalContent, finalArtifact, nil
}

// uploadToDestinations uploads content to all destinations concurrently.
func (m *DefaultMirror) uploadToDestinations(
	ctx context.Context,
	finalContent io.ReadCloser,
	finalArtifact *core.Artifact,
	destinations []core.Destination,
	opts *core.MirrorOptions,
	result *MirrorResult,
	logger *logrus.Entry,
) error {
	// Buffer content for concurrent uploads
	buf, err := m.bufferContent(finalContent, logger)
	if err != nil {
		return err
	}

	// Create readers for each destination
	readers := make([]io.Reader, len(destinations))
	for i := range destinations {
		readers[i] = bytes.NewReader(buf.Bytes())
	}

	logger.WithField("num_destinations", len(destinations)).Info("Starting concurrent uploads to destinations")

	// Perform concurrent uploads
	lastErr, destinationPath := m.performConcurrentUploads(ctx, destinations, readers, finalArtifact, opts, logger)

	// Update result
	result.Artifact = finalArtifact
	result.DestinationPath = destinationPath

	if lastErr != nil {
		logger.WithError(lastErr).Error("One or more uploads failed")
	} else {
		logger.WithField("destination_path", destinationPath).Info("All uploads completed successfully")
	}

	return lastErr
}

// bufferContent buffers content for concurrent uploads.
func (m *DefaultMirror) bufferContent(content io.ReadCloser, logger *logrus.Entry) (*bytes.Buffer, error) {
	logger.Info("Buffering artifact content for upload")

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, content); err != nil {
		logger.WithError(err).Error("Failed to buffer artifact content")

		return nil, fmt.Errorf("failed to buffer content: %w", err)
	}

	logger.WithField("buffer_size", buf.Len()).Debug("Successfully buffered artifact content")

	return &buf, nil
}

// performConcurrentUploads performs uploads to all destinations concurrently.
func (m *DefaultMirror) performConcurrentUploads(
	ctx context.Context,
	destinations []core.Destination,
	readers []io.Reader,
	finalArtifact *core.Artifact,
	opts *core.MirrorOptions,
	logger *logrus.Entry,
) (error, string) {
	var wg sync.WaitGroup

	errChan := make(chan error, len(destinations))
	destinationPathChan := make(chan string, len(destinations))

	for i, dest := range destinations {
		wg.Add(1)

		params := &UploadWorkerParams{
			ctx:                 ctx,
			dest:                dest,
			reader:              readers[i],
			finalArtifact:       finalArtifact,
			opts:                opts,
			logger:              logger,
			index:               i,
			wg:                  &wg,
			errChan:             errChan,
			destinationPathChan: destinationPathChan,
		}

		go m.uploadWorker(params)
	}

	wg.Wait()
	close(errChan)
	close(destinationPathChan)

	// Collect results
	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	var destinationPath string
	for path := range destinationPathChan {
		if destinationPath == "" {
			destinationPath = path
		}
	}

	return lastErr, destinationPath
}

// UploadWorkerParams holds parameters for the upload worker.
type UploadWorkerParams struct {
	ctx                 context.Context
	dest                core.Destination
	reader              io.Reader
	finalArtifact       *core.Artifact
	opts                *core.MirrorOptions
	logger              *logrus.Entry
	index               int
	wg                  *sync.WaitGroup
	errChan             chan<- error
	destinationPathChan chan<- string
}

// uploadWorker handles upload to a single destination.
func (m *DefaultMirror) uploadWorker(params *UploadWorkerParams) {
	defer params.wg.Done()

	destLogger := params.logger.WithField("destination_index", params.index)
	destLogger.Debug("Starting upload to destination")

	raw := params.opts != nil && params.opts.Raw

	destinationPath, err := params.dest.Put(params.ctx, params.finalArtifact, params.reader, raw)
	if err != nil {
		destLogger.WithError(err).Error("Failed to upload to destination")
		params.errChan <- fmt.Errorf("failed to upload to destination: %w", err)
	} else {
		destLogger.WithField("destination_path", destinationPath).Info("Successfully uploaded to destination")
		params.destinationPathChan <- destinationPath
	}
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
	// Create temp directory and extract file
	tempDir, extractedPath, extractedName, err := m.setupAndExtractZip(tempZipPath, logger)
	if err != nil {
		return nil, nil, err
	}

	// Open extracted file
	extractedFile, err := os.Open(extractedPath)
	if err != nil {
		os.RemoveAll(tempDir)

		return nil, nil, fmt.Errorf("failed to open extracted file: %w", err)
	}

	// Create artifact with final name and cleanup reader
	finalName := m.determineFinalName(artifact.Name, extractedName, opts, logger)
	extractedArtifact := m.createExtractedArtifact(artifact, finalName, extractedName, logger)
	finalContent := &tempDirCleanupReader{ReadCloser: extractedFile, tempDir: tempDir, logger: logger}

	return finalContent, extractedArtifact, nil
}

// setupAndExtractZip creates temp directory and extracts ZIP file.
func (m *DefaultMirror) setupAndExtractZip(tempZipPath string, logger *logrus.Logger) (string, string, string, error) {
	tempDir, err := os.MkdirTemp("", "garf-unzip-*")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logger.WithField("temp_dir", tempDir).Debug("Created temporary directory for ZIP extraction")

	extractedPath, extractedName, err := m.extractSingleFileFromZip(tempZipPath, tempDir, logger)
	if err != nil {
		os.RemoveAll(tempDir)

		return "", "", "", fmt.Errorf("failed to extract ZIP: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"original_zip":   tempZipPath,
		"extracted_path": extractedPath,
		"extracted_name": extractedName,
		"temp_dir":       tempDir,
	}).Info("Successfully extracted ZIP file")

	return tempDir, extractedPath, extractedName, nil
}

// determineFinalName determines the final artifact name based on options.
func (m *DefaultMirror) determineFinalName(originalName, extractedName string, opts *core.MirrorOptions, logger *logrus.Logger) string {
	if opts != nil && opts.PreserveZipName {
		finalName := m.buildPreservedZipName(originalName, extractedName)
		logger.WithFields(logrus.Fields{
			"original_zip_name": originalName,
			"extracted_name":    extractedName,
			"preserved_name":    finalName,
		}).Info("Preserving ZIP filename with extracted file extension")

		return finalName
	}

	return extractedName
}

// createExtractedArtifact creates a new artifact with extracted file information.
func (m *DefaultMirror) createExtractedArtifact(original *core.Artifact, finalName, extractedName string, logger *logrus.Logger) *core.Artifact {
	extractedArtifact := &core.Artifact{
		Name:     finalName,
		Location: original.Location,
		Metadata: original.Metadata,
		Version:  original.Version,
	}

	logger.WithFields(logrus.Fields{
		"original_name":     original.Name,
		"extracted_name":    extractedName,
		"final_name":        finalName,
		"original_location": original.Location,
	}).Info("Updated artifact information with extracted file")

	return extractedArtifact
}

// extractSingleFileFromZip extracts a single file from a ZIP archive.
// Returns the path to the extracted file and the filename.
func (m *DefaultMirror) extractSingleFileFromZip(
	zipPath, destDir string,
	logger *logrus.Logger,
) (string, string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	targetFile, err := m.findSingleFileInZip(reader)
	if err != nil {
		return "", "", err
	}

	if err := m.validateFileSize(targetFile); err != nil {
		return "", "", err
	}

	logger.WithField("zip_file_name", targetFile.Name).Debug("Found file in ZIP archive")

	return m.extractFileFromZip(targetFile, destDir, logger)
}

// findSingleFileInZip finds exactly one non-directory file in the ZIP.
func (m *DefaultMirror) findSingleFileInZip(reader *zip.ReadCloser) (*zip.File, error) {
	var targetFile *zip.File

	for _, f := range reader.File {
		if !f.FileInfo().IsDir() {
			if targetFile != nil {
				return nil, fmt.Errorf("ZIP file contains multiple files, expected exactly one")
			}

			targetFile = f
		}
	}

	if targetFile == nil {
		return nil, fmt.Errorf("no files found in ZIP archive")
	}

	return targetFile, nil
}

// validateFileSize checks if the file size is within limits.
func (m *DefaultMirror) validateFileSize(file *zip.File) error {
	if file.UncompressedSize64 > maxExtractedFileSize {
		return fmt.Errorf("file too large: %d bytes (max allowed: %d bytes)",
			file.UncompressedSize64, maxExtractedFileSize)
	}

	return nil
}

// extractFileFromZip extracts a file from ZIP to the destination directory.
func (m *DefaultMirror) extractFileFromZip(targetFile *zip.File, destDir string, logger *logrus.Logger) (string, string, error) {
	if err := os.MkdirAll(destDir, defaultDirPerm); err != nil {
		return "", "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	fileName := filepath.Base(targetFile.Name)
	destPath := filepath.Join(destDir, fileName)

	if err := m.copyFileFromZip(targetFile, destPath); err != nil {
		return "", "", err
	}

	logger.WithFields(logrus.Fields{
		"zip_file_name": targetFile.Name,
		"extracted_to":  destPath,
		"file_size":     targetFile.UncompressedSize64,
	}).Debug("Successfully extracted file from ZIP")

	return destPath, fileName, nil
}

// copyFileFromZip copies a file from ZIP to destination with size protection.
func (m *DefaultMirror) copyFileFromZip(targetFile *zip.File, destPath string) error {
	rc, err := targetFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open file inside ZIP: %w", err)
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	// Copy with size limit protection
	limitedReader := io.LimitReader(rc, maxExtractedFileSize+1)

	bytesWritten, err := io.Copy(outFile, limitedReader)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	if bytesWritten > maxExtractedFileSize {
		return fmt.Errorf("file exceeded size limit during extraction: %d bytes", bytesWritten)
	}

	return nil
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
