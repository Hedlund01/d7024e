#!/bin/bash

VERSION_FILE="version.txt"

# Create version file if it doesn't exist
if [ ! -f "$VERSION_FILE" ]; then
    echo "Creating version file with initial version 1"
    echo "1" > "$VERSION_FILE"
fi

# Read current version
VERSION=$(cat "$VERSION_FILE")
echo "Current version: $VERSION"

# Increment version
NEW_VERSION=$((VERSION + 1))
BUILD_TIME=$(date -Iseconds)
echo "Building with new version: $NEW_VERSION"
echo "Build time: $BUILD_TIME"
echo "$NEW_VERSION" > "$VERSION_FILE"

# Build the Docker image using docker compose with the incremented version
echo "Building Docker image with docker build..."
docker build -f ../Dockerfile -t kadlab:latest --build-arg BUILD_VERSION="$NEW_VERSION" --build-arg BUILD_TIME="$BUILD_TIME" .. 