#!/usr/bin/env bash

# Get the current git commit hash
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

# Get the current date in ISO 8601 format
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Get the version from git tags
# If we're on a tag, use that as the version
# Otherwise, use the latest tag with a dev suffix
VERSION=$(git describe --tags --exact-match 2>/dev/null || git describe --tags --always 2>/dev/null || echo "dev")

# Print the status information
echo "STABLE_VERSION ${VERSION}"
echo "STABLE_GIT_COMMIT ${GIT_COMMIT}"
echo "BUILD_DATE ${BUILD_DATE}" 