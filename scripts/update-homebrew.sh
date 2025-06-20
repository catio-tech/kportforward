#!/usr/bin/env bash

set -e

# Script to update the Homebrew formula with the new version and checksums

# Configuration
VERSION=${1:-""}
if [ -z "${VERSION}" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.0.0"
    exit 1
fi

# Strip the 'v' prefix for the formula
FORMULA_VERSION=$(echo "${VERSION}" | sed 's/^v//')
FORMULA_PATH="HomebrewFormula/kportforward.rb"

echo "Updating Homebrew formula for ${VERSION}..."

# Make sure the dist directory exists and contains the binaries
if [ ! -d "dist" ]; then
    echo "Error: dist directory not found. Run ./scripts/build.sh first."
    exit 1
fi

# Get checksums for the binaries
if [ ! -f "dist/checksums.txt" ]; then
    echo "Error: checksums.txt not found. Run ./scripts/build.sh first."
    exit 1
fi

# Extract checksums for each platform
DARWIN_AMD64_CHECKSUM=$(grep "kportforward-darwin-amd64" dist/checksums.txt | awk '{print $1}')
DARWIN_ARM64_CHECKSUM=$(grep "kportforward-darwin-arm64" dist/checksums.txt | awk '{print $1}')
LINUX_AMD64_CHECKSUM=$(grep "kportforward-linux-amd64" dist/checksums.txt | awk '{print $1}')

if [ -z "$DARWIN_AMD64_CHECKSUM" ] || [ -z "$DARWIN_ARM64_CHECKSUM" ] || [ -z "$LINUX_AMD64_CHECKSUM" ]; then
    echo "Error: Could not extract all required checksums from dist/checksums.txt"
    exit 1
fi

echo "Checksums:"
echo "  darwin-amd64: ${DARWIN_AMD64_CHECKSUM}"
echo "  darwin-arm64: ${DARWIN_ARM64_CHECKSUM}"
echo "  linux-amd64: ${LINUX_AMD64_CHECKSUM}"

# Update the formula with the new version and checksums
# Using sed with different delimiters to avoid issues with slashes in URLs
sed -i.bak \
    -e "s/^  version \".*\"/  version \"${FORMULA_VERSION}\"/" \
    -e "s/sha256 \".*\"/sha256 \"${DARWIN_ARM64_CHECKSUM}\"/" \
    -e "0,/sha256 \".*\"/{//!b};0,/sha256 \".*\"/{s/sha256 \".*\"/sha256 \"${DARWIN_ARM64_CHECKSUM}\"/}" \
    -e "0,/sha256 \".*\"/{//!b};0,/sha256 \".*\"/{s/sha256 \".*\"/sha256 \"${DARWIN_AMD64_CHECKSUM}\"/;n;b}" \
    -e "s|url \"https://github.com/catio-tech/kportforward/releases/latest/download/|url \"https://github.com/catio-tech/kportforward/releases/download/${VERSION}/|g" \
    ${FORMULA_PATH}

# Handle Linux checksum separately since it's harder to match with sed patterns
awk -v checksum="${LINUX_AMD64_CHECKSUM}" '
    /linux-amd64/ { 
        print $0; 
        getline; 
        sub(/sha256 ".*"/, "sha256 \"" checksum "\""); 
        print; 
        next; 
    } 
    { print }
' ${FORMULA_PATH}.bak > ${FORMULA_PATH}

# Remove backup file
rm ${FORMULA_PATH}.bak

# Change URLs from /latest/download/ to /download/VERSION/
sed -i.bak \
    -e "s|releases/latest/download/|releases/download/${VERSION}/|g" \
    ${FORMULA_PATH}

rm ${FORMULA_PATH}.bak

echo "Homebrew formula updated successfully!"
echo "Review the changes:"
git diff ${FORMULA_PATH}

echo ""
echo "Next steps:"
echo "1. Commit the updated formula: git add ${FORMULA_PATH} && git commit -m \"Update Homebrew formula to ${VERSION}\""
echo "2. Continue with the release process"