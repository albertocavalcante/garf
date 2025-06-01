// Package main demonstrates how to use garf as a library.
//
// This example shows the recommended way for third-party Go programs
// to integrate garf for artifact mirroring functionality.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/albertocavalcante/garf"
)

const (
	// Example configuration constants.
	exampleTimeout    = 10 * time.Minute
	exampleConcurrent = 8
	defaultTimeout    = 30 * time.Minute
	defaultConcurrent = 4
	mirrorTimeout     = 5 * time.Minute
)

func main() {
	fmt.Println("📋 Garf Library Usage Examples")
	fmt.Println("==============================")

	// Example 1: Basic usage with minimal configuration
	basicExample()

	// Example 2: Advanced usage with custom configuration
	advancedExample()

	// Example 3: Environment-based configuration (recommended)
	envConfigExample()

	// Example 4: Batch mirroring multiple artifacts
	batchExample()

	// Example 5: Error handling and dry run
	errorHandlingExample()

	// Example 6: JFrog-to-JFrog mirroring with source path stripping
	sourcePathStrippingExample()

	// Example 7: API features - source type detection and validation
	apiFeatureExample()

	fmt.Println("\n✅ All examples completed!")
}

func basicExample() {
	fmt.Println("\n=== Basic Example ===")

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://mycompany.jfrog.io/artifactory",
		JFrogUser:     "username",
		JFrogPassword: "password",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	result, err := client.Mirror(ctx, garf.MirrorRequest{
		Source:      "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe",
		Destination: "tools-local",
	})
	if err != nil {
		log.Printf("Mirror failed: %v", err)

		return
	}

	if result.Error != nil {
		log.Printf("Mirror operation failed: %v", result.Error)

		return
	}

	fmt.Printf("✓ Successfully mirrored %s to %s\n", result.Source, result.DestinationPath)
}

func advancedExample() {
	fmt.Println("\n=== Advanced Example ===")

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://mycompany.jfrog.io/artifactory",
		JFrogUser:     "username",
		JFrogPassword: "password",
		Timeout:       exampleTimeout,
		Concurrent:    exampleConcurrent,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	result, err := client.Mirror(ctx, garf.MirrorRequest{
		Source:      "https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip",
		Destination: "tools-local",
		Properties: map[string]string{
			"type":     "toolchain",
			"platform": "windows",
			"arch":     "x86_64",
			"version":  "7.6.0",
		},
		Unzip: true, // Extract single files from zip archives - fully implemented!
		Raw:   false,
	})
	if err != nil {
		log.Printf("Mirror failed: %v", err)

		return
	}

	if result.Error != nil {
		log.Printf("Mirror operation failed: %v", result.Error)

		return
	}

	fmt.Printf("✓ Successfully mirrored %s to %s\n", result.Source, result.DestinationPath)
}

func envConfigExample() {
	fmt.Println("\n=== Environment Configuration Example (Recommended) ===")

	// Check if environment variables are set
	jfrogURL := os.Getenv("JFROG_URL")
	jfrogUser := os.Getenv("JFROG_USER")
	jfrogPassword := os.Getenv("JFROG_PASSWORD")

	if jfrogURL == "" || jfrogUser == "" || jfrogPassword == "" {
		fmt.Println("⚠️  Skipping environment example - set JFROG_URL, JFROG_USER, JFROG_PASSWORD")
		fmt.Println("   Example: export JFROG_URL=https://mycompany.jfrog.io/artifactory")

		return
	}

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      jfrogURL,
		JFrogUser:     jfrogUser,
		JFrogPassword: jfrogPassword,
		Timeout:       defaultTimeout,
		Concurrent:    defaultConcurrent,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	result, err := client.Mirror(ctx, garf.MirrorRequest{
		Source:      "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-linux-x86_64",
		Destination: "tools-local",
		Properties: map[string]string{
			"type":     "toolchain",
			"platform": "linux",
			"arch":     "x86_64",
		},
		DryRun:     true,
		DryRunMode: "all",
	})
	if err != nil {
		log.Printf("Mirror failed: %v", err)

		return
	}

	fmt.Printf("✓ Dry run successful for %s\n", result.Source)
}

func batchExample() {
	fmt.Println("\n=== Batch Example ===")

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://mycompany.jfrog.io/artifactory",
		JFrogUser:     "username",
		JFrogPassword: "password",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	artifacts := []struct {
		source      string
		destination string
		properties  map[string]string
		unzip       bool
		description string
	}{
		{
			source:      "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-linux-x86_64",
			destination: "tools-local",
			properties:  map[string]string{"platform": "linux", "arch": "x86_64"},
			unzip:       false,
			description: "Linux binary",
		},
		{
			source:      "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-darwin-x86_64",
			destination: "tools-local",
			properties:  map[string]string{"platform": "darwin", "arch": "x86_64"},
			unzip:       false,
			description: "macOS binary",
		},
		{
			source:      "https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip",
			destination: "tools-local",
			properties:  map[string]string{"platform": "windows", "arch": "x86_64", "type": "zip"},
			unzip:       true,
			description: "Windows ZIP (will be extracted)",
		},
	}

	ctx := context.Background()
	successCount := 0

	for i, artifact := range artifacts {
		fmt.Printf("Mirroring artifact %d/%d: %s (%s)\n", i+1, len(artifacts), artifact.description, artifact.source)

		result, err := client.Mirror(ctx, garf.MirrorRequest{
			Source:      artifact.source,
			Destination: artifact.destination,
			Properties:  artifact.properties,
			Unzip:       artifact.unzip,
		})
		if err != nil {
			log.Printf("❌ Failed to mirror %s: %v", artifact.source, err)

			continue
		}

		if result.Error != nil {
			log.Printf("❌ Mirror operation failed for %s: %v", artifact.source, result.Error)

			continue
		}

		if artifact.unzip {
			fmt.Printf("✓ Extracted and mirrored %s\n", result.DestinationPath)
		} else {
			fmt.Printf("✓ Mirrored %s\n", result.DestinationPath)
		}

		successCount++
	}

	fmt.Printf("📊 Batch complete: %d/%d artifacts mirrored successfully\n", successCount, len(artifacts))
}

func errorHandlingExample() {
	fmt.Println("\n=== Error Handling & Dry Run Example ===")

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://mycompany.jfrog.io/artifactory",
		JFrogUser:     "username",
		JFrogPassword: "password",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	source := "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe"

	// Step 1: Validate with dry run
	fmt.Println("🔍 Validating with dry run...")

	_, err = client.Mirror(ctx, garf.MirrorRequest{
		Source:      source,
		Destination: "tools-local",
		DryRun:      true,
		DryRunMode:  "all",
	})
	if err != nil {
		log.Printf("❌ Dry run validation failed: %v", err)

		return
	}

	fmt.Println("✓ Dry run validation passed")

	// Step 2: Perform actual mirror with timeout
	fmt.Println("⚡ Performing actual mirror...")

	ctx, cancel := context.WithTimeout(ctx, mirrorTimeout)

	defer cancel()

	result, err := client.Mirror(ctx, garf.MirrorRequest{
		Source:      source,
		Destination: "tools-local",
		Properties: map[string]string{
			"validated": "true",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})

	// Comprehensive error handling
	switch {
	case err != nil:
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("⏰ Mirror timed out: %v", err)
		} else {
			log.Printf("❌ Mirror failed: %v", err)
		}

		return
	case result.Error != nil:
		log.Printf("❌ Mirror operation failed: %v", result.Error)

		return
	default:
		fmt.Printf("✅ Successfully mirrored to %s\n", result.DestinationPath)
	}
}

func sourcePathStrippingExample() {
	fmt.Println("\n=== Source Path Stripping Example ===")

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      os.Getenv("JFROG_URL"),
		JFrogUser:     os.Getenv("JFROG_USER"),
		JFrogPassword: os.Getenv("JFROG_PASSWORD"),
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Example 1: Mirror from staging to production with prefix stripping
	fmt.Println("📦 JFrog-to-JFrog mirroring with staging prefix strip")

	result1, err := client.Mirror(ctx, garf.MirrorRequest{
		Source:          "https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
		Destination:     "prod-repo",
		SourcePathStrip: "artifactory.corp.net/staging/",
		Properties:      map[string]string{"type": "binary", "platform": "windows"},
		DryRun:          true, // Use dry run for demonstration
		DryRunMode:      "all",
	})
	if err != nil {
		log.Printf("Mirror failed: %v", err)

		return
	}

	if result1.Error != nil {
		log.Printf("Mirror operation failed: %v", result1.Error)

		return
	}

	fmt.Printf("✓ Mirrored: %s -> %s\n", result1.Source, result1.DestinationPath)

	// Example 2: Mirror with host-only stripping
	fmt.Println("📦 JFrog-to-JFrog mirroring with host strip")

	result2, err := client.Mirror(ctx, garf.MirrorRequest{
		Source:          "https://artifactory.corp.net/repo/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe",
		Destination:     "prod-repo",
		SourcePathStrip: "artifactory.corp.net",
		Properties:      map[string]string{"type": "binary", "platform": "windows"},
		DryRun:          true, // Use dry run for demonstration
		DryRunMode:      "all",
	})
	if err != nil {
		log.Printf("Mirror failed: %v", err)

		return
	}

	if result2.Error != nil {
		log.Printf("Mirror operation failed: %v", result2.Error)

		return
	}

	fmt.Printf("✓ Mirrored: %s -> %s\n", result2.Source, result2.DestinationPath)

	fmt.Println("💡 Key benefits of source path stripping:")
	fmt.Println("   - Clean JFrog-to-JFrog mirroring without nested repository paths")
	fmt.Println("   - Flexible prefix removal for different repository structures")
	fmt.Println("   - Maintains proper artifact organization and metadata")
}

func apiFeatureExample() {
	fmt.Println("\n=== API Features Example ===")

	client, err := garf.NewClient(garf.Config{
		JFrogURL:      "https://mycompany.jfrog.io/artifactory",
		JFrogUser:     "username",
		JFrogPassword: "password",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Example 1: Source type detection for GitHub URLs
	fmt.Println("🔍 Detecting source types...")

	githubURL := "https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe"
	sourceType := client.DetectSourceType(githubURL, "")
	fmt.Printf("✓ GitHub URL '%s' detected as: %s\n", githubURL, sourceType)

	// Example 2: Source type detection for generic HTTP URLs
	genericURL := "https://releases.example.com/artifacts/v1.0.0/tool.tar.gz"
	sourceType = client.DetectSourceType(genericURL, "")
	fmt.Printf("✓ Generic URL '%s' detected as: %s\n", genericURL, sourceType)

	// Example 3: Source type detection with path stripping
	jfrogURL := "https://staging.jfrog.io/artifactory/staging-repo/github.com/owner/repo/releases/download/v1.0.0/artifact.zip"
	pathStrip := "staging.jfrog.io/artifactory/staging-repo/"
	sourceType = client.DetectSourceType(jfrogURL, pathStrip)
	fmt.Printf("✓ JFrog URL with path strip detected as: %s\n", sourceType)
	fmt.Printf("  Original: %s\n", jfrogURL)
	fmt.Printf("  Strip: %s\n", pathStrip)

	// Example 4: Ensure source availability
	fmt.Println("🔍 Checking source availability...")

	if err := client.EnsureSourceAvailable("github"); err != nil {
		log.Printf("❌ GitHub source not available: %v", err)
	} else {
		fmt.Println("✓ GitHub source is available")
	}

	if err := client.EnsureSourceAvailable("generic"); err != nil {
		log.Printf("❌ Generic source not available: %v", err)
	} else {
		fmt.Println("✓ Generic source is available")
	}

	// Example 5: Request validation
	fmt.Println("🔍 Validating mirror requests...")

	validRequest := garf.MirrorRequest{
		Source:          "https://github.com/example/repo/releases/download/v1.0/file.zip",
		Destination:     "test-repo",
		SourcePathStrip: "valid/path/strip/",
		Unzip:           true,
		Properties:      map[string]string{"type": "binary"},
	}

	if err := client.ValidateRequest(validRequest); err != nil {
		log.Printf("❌ Request validation failed: %v", err)
	} else {
		fmt.Println("✓ Request validation passed")
	}

	// Example 6: Invalid request validation
	invalidRequest := garf.MirrorRequest{
		Source:          "https://github.com/example/repo/releases/download/v1.0/file.zip",
		Destination:     "",                // Invalid: empty destination
		SourcePathStrip: "../invalid/path", // Invalid: contains ..
	}

	if err := client.ValidateRequest(invalidRequest); err != nil {
		fmt.Printf("✓ Invalid request correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid request was incorrectly accepted")
	}

	fmt.Println("📊 Client statistics:")
	fmt.Printf("  Cached destinations: %d\n", client.GetCachedDestinationsCount())
}
