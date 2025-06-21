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

# Create a new formula with the updated version and checksums
cat > ${FORMULA_PATH} << EOF
class Kportforward < Formula
  desc "Modern Kubernetes port-forward manager with TUI"
  homepage "https://github.com/catio-tech/kportforward"
  license "MIT"
  version "${FORMULA_VERSION}"

  # Use explicit file naming and SHA256 checksums
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/catio-tech/kportforward/releases/download/${VERSION}/kportforward-darwin-arm64"
      sha256 "${DARWIN_ARM64_CHECKSUM}"
    else
      url "https://github.com/catio-tech/kportforward/releases/download/${VERSION}/kportforward-darwin-amd64"
      sha256 "${DARWIN_AMD64_CHECKSUM}"
    end
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/catio-tech/kportforward/releases/download/${VERSION}/kportforward-linux-amd64"
    sha256 "${LINUX_AMD64_CHECKSUM}"
  end

  depends_on "kubectl" => :recommended

  def install
    # Move the downloaded binary to the bin directory with the name "kportforward"
    # First, find what files we have in the current directory
    binary = Dir["*"].first
    bin.install binary => "kportforward"
    
    # Ensure binary is executable
    chmod 0755, bin/"kportforward"
  end

  test do
    assert_match(/kportforward/i, shell_output("#{bin}/kportforward version 2>&1", 2))
  end
end
EOF

echo "Homebrew formula updated successfully!"
echo "Review the changes:"
git diff ${FORMULA_PATH}

echo ""
echo "Next steps:"
echo "1. Commit the updated formula: git add ${FORMULA_PATH} && git commit -m \"Update Homebrew formula to ${VERSION}\""
echo "2. Continue with the release process"