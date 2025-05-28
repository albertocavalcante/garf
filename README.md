# garf

`garf` is a tool for mirroring artifacts from public repositories to JFrog Artifactory with structured path organization and metadata handling.

## Overview

This CLI tool simplifies the artifact mirroring workflow by:

- Downloading artifacts from sources like GitHub Releases
- Extracting coordinates (org, repo, version, etc.) from URLs
- Uploading to JFrog Artifactory with organized path structures
- Supporting metadata through custom properties
- Extracting files from single-file zip archives when needed

## Quick Start

### Prerequisites

- [Bazel](https://bazel.build/) for building and running
- JFrog Artifactory instance credentials

### Environment Setup

```sh
# Linux/macOS
export JFROG_URL="https://your-instance.jfrog.io/artifactory"
export JFROG_USER="username"
export JFROG_PASSWORD="password"

# Windows
set JFROG_URL=https://your-instance.jfrog.io/artifactory
set JFROG_USER=username
set JFROG_PASSWORD=password
```

### Simple Example

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel-7.2.1-windows-x86_64.exe \
  --destination tools-local
```

## Usage Examples

### Mirror GitHub Release Asset

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe \
  --destination tools-local
```

### Add Metadata Properties

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe \
  --destination tools-local \
  --properties type=toolchain \
  --properties platform=windows \
  --properties version=7.2.1
```

### Use Raw URL Path Structure

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe \
  --destination tools-local \
  --raw
```

### Upload From Local File

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe \
  --from-file /path/to/bazel_nojdk-7.2.1-windows-x86_64.exe \
  --destination tools-local
```

### Extract and Upload From Zip

For zip files containing a single file (like executables packaged as zip):

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip \
  --destination tools-local \
  --unzip
```

This will extract the file (e.g., `bazel_nojdk-7.6.0-windows-x86_64.exe`) from the zip and upload it directly.

### JFrog-to-JFrog Mirroring with Source Path Stripping

For mirroring artifacts from one JFrog repository to another (e.g., staging to production), you can strip source path prefixes to avoid nested repository structures:

```sh
# Mirror from staging to production, stripping the staging prefix
bazel run //:garf -- mirror \
  --source https://artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe \
  --destination prod-repo \
  --source-path-strip "artifactory.corp.net/staging/"
```

This will:
- Strip `artifactory.corp.net/staging/` from the source URL
- Process the remaining URL (`github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe`) normally
- Upload to `prod-repo/github.com/bazelbuild/bazel/v8.2.1/bazel-win.exe`

Without source path stripping, the result would be:
`prod-repo/artifactory.corp.net/staging/github.com/bazelbuild/bazel/releases/download/v8.2.1/bazel-win.exe`

#### Common Use Cases

**Strip staging repository prefix:**
```sh
--source-path-strip "artifactory.corp.net/staging/"
```

**Strip entire host:**
```sh
--source-path-strip "artifactory.corp.net"
```

**Strip custom path prefix:**
```sh
--source-path-strip "my-company.jfrog.io/temp-repo/"
```

### Dry Run Mode

The mirror command supports dry run modes to help with testing and validation without making actual changes to Artifactory. There are two dry run modes available:

#### Skip All Operations

This mode validates the command syntax and configuration without performing any actual operations:

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip \
  --destination tools-local \
  --dry-run
```

This is useful for:
- Testing command syntax
- Validating configuration
- Checking environment variables
- Verifying artifact URLs

#### Skip Only Upload

This mode downloads and processes artifacts but skips uploading to Artifactory:

```sh
bazel run //:garf -- mirror \
  --source https://github.com/bazelbuild/bazel/releases/download/7.6.0/bazel_nojdk-7.6.0-windows-x86_64.zip \
  --destination tools-local \
  --dry-run \
  --dry-run-mode upload
```

This is useful for:
- Local development without Artifactory access
- Testing artifact processing without uploading
- Validating download and processing logic
- CI/CD environments where Artifactory is not available

## Command Reference

### Mirror Command

```
garf mirror [OPTIONS]
```

#### Options

| Option | Short | Description |
|--------|-------|-------------|
| `--destination` | `-d` | JFrog repository destination (required) |
| `--from-file` | `-f` | Local file path to upload (URL still used for coordinates) |
| `--properties` | | Add properties to the artifact (can be used multiple times) |
| `--raw` | | Use the full URL path structure rather than parsed coordinates |
| `--source` | `-s` | URL of the artifact to mirror (required) |
| `--source-path-strip` | | Strip source path prefixes for JFrog-to-JFrog mirroring |
| `--unzip` | | Extract and upload the content from zip files with a single file |
| `--dry-run` | | Perform a dry run without making actual changes |
| `--dry-run-mode` | | Dry run mode: 'all' (skip all operations), 'upload' (skip only upload to Artifactory) |

## Path Organization

### Standard Path Structure (Default)

```
{destination}/{host}/{org}/{repo}/{version}/{artifact}
```

Example:
```
tools-local/github.com/bazelbuild/bazel/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe
```

### Raw Path Structure (with --raw flag)

```
{destination}/{host}/{full_url_path}
```

Example:
```
tools-local/github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `JFROG_URL` | URL of the JFrog Artifactory instance (required) |
| `JFROG_USER` | Username for authentication (required) |
| `JFROG_PASSWORD` | Password for authentication (required) |

## Building From Source

```sh
# Clone repository
git clone https://github.com/albertocavalcante/garf.git
cd garf

# Build the binary
bazel build //:garf
```
