#!/usr/bin/env bash

set -e

# Build script for kportforward - builds for all target platforms

# Configuration
BINARY_NAME="kportforward"
PACKAGE="./cmd/kportforward"
BUILD_DIR="dist"
VERSION=${VERSION:-"dev"}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
DATE=${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

# Target platforms
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    # Disabled for now because the windows build is throwing errors
    "windows/amd64"
)

echo "Building ${BINARY_NAME} version ${VERSION}"
echo "Commit: ${COMMIT}"
echo "Date: ${DATE}"
echo ""

# Clean previous builds
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    
    echo "Building for ${GOOS}/${GOARCH}..."
    
    # Set binary name with appropriate extension
    if [ "${GOOS}" = "windows" ]; then
        BINARY="${BINARY_NAME}-${GOOS}-${GOARCH}.exe"
    else
        BINARY="${BINARY_NAME}-${GOOS}-${GOARCH}"
    fi
    
    # Build
    env GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build \
        -ldflags="${LDFLAGS}" \
        -o ${BUILD_DIR}/${BINARY} \
        ${PACKAGE}
    
    if [ $? -eq 0 ]; then
        echo "  ✓ Built ${BUILD_DIR}/${BINARY}"
        
        # Calculate file size
        if command -v du >/dev/null 2>&1; then
            SIZE=$(du -h ${BUILD_DIR}/${BINARY} | cut -f1)
            echo "    Size: ${SIZE}"
        fi
    else
        echo "  ✗ Failed to build for ${GOOS}/${GOARCH}"
        exit 1
    fi
    
    echo ""
done

# Create checksums
echo "Generating checksums..."
cd ${BUILD_DIR}

if command -v sha256sum >/dev/null 2>&1; then
    sha256sum * > checksums.txt
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 * > checksums.txt
else
    echo "Warning: No checksum utility found (sha256sum or shasum)"
fi

cd ..

echo "Build complete! Artifacts are in ${BUILD_DIR}/"
echo ""
echo "Available binaries:"
ls -la ${BUILD_DIR}/