#!/usr/bin/env bash

set -e

# Release script for kportforward - builds and packages for release

# Configuration
VERSION=${1:-""}
if [ -z "${VERSION}" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.0.0"
    exit 1
fi

# Validate version format
if [[ ! "${VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z (e.g., v1.0.0)"
    exit 1
fi

echo "Preparing release ${VERSION}"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo "Error: There are uncommitted changes. Please commit or stash them first."
    exit 1
fi

# Check if tag already exists
if git tag -l | grep -q "^${VERSION}$"; then
    echo "Error: Tag ${VERSION} already exists"
    exit 1
fi

# Set environment variables for build
export VERSION="${VERSION}"
export COMMIT=$(git rev-parse --short HEAD)
export DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Building release binaries..."
./scripts/build.sh

echo ""
echo "Updating Homebrew formula..."
./scripts/update-homebrew.sh "${VERSION}"

echo ""
echo "Creating git tag..."
git tag -a "${VERSION}" -m "Release ${VERSION}"

echo ""
echo "Release ${VERSION} is ready!"
echo ""
echo "Next steps:"
echo "1. Review and commit the Homebrew formula changes:"
echo "   git add HomebrewFormula/kportforward.rb"
echo "   git commit -m \"Update Homebrew formula for ${VERSION}\""
echo ""
echo "2. Push the changes and tag:"
echo "   git push origin main"
echo "   git push origin ${VERSION}"
echo ""
echo "3. Create a GitHub release with the binaries in dist/"
echo "   gh release create ${VERSION} --title \"Release ${VERSION}\" --notes \"Release notes\" dist/*"
echo ""
echo "Available binaries:"
ls -la dist/

echo ""
echo "To clean up if something goes wrong:"
echo "  git tag -d ${VERSION}"
echo "  git reset --hard HEAD~1 # If you've committed the formula changes"