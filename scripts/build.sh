#!/bin/bash

# Build script for Stori Transaction Processor
set -e

echo "🏗️  Building Stori Transaction Processor..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | cut -d'o' -f2)
print_status "Using Go version: $GO_VERSION"

# Create build directory
BUILD_DIR="./build"
mkdir -p "$BUILD_DIR"

# Download dependencies
print_status "Downloading dependencies..."
go mod download
go mod tidy

# Format code
print_status "Formatting code..."
gofmt -w .

# Run go vet
print_status "Running go vet..."
go vet ./...

# Run tests
print_status "Running tests..."
if go test -v -race ./...; then
    print_status "All tests passed ✅"
else
    print_error "Tests failed ❌"
    exit 1
fi

# Build for Linux (Docker)
print_status "Building for Linux (Docker)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static" -s -w' \
    -o "$BUILD_DIR/processor-linux" \
    ./cmd/processor

# Build for current OS
print_status "Building for current OS..."
go build -o "$BUILD_DIR/processor" ./cmd/processor

# Create data directory
mkdir -p ./data

print_status "Build completed successfully! 🎉"
print_status "Binaries created:"
print_status "  - $BUILD_DIR/processor (current OS)"
print_status "  - $BUILD_DIR/processor-linux (for Docker)"

echo ""
echo "Next steps:"
echo "  1. Run locally: make run"
echo "  2. Run with Docker: make docker-run"
echo "  3. View logs: docker-compose logs -f"
echo "  4. Access MailHog UI: http://localhost:8025"