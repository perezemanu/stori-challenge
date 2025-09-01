#!/bin/bash

# Build script for Lambda deployment
set -e

echo "🏗️  Building Stori Challenge Lambda function..."

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf bin/
mkdir -p bin/

# Run tests first (with native environment and test build tag)
echo "🧪 Running tests..."
go test ./... -v -short -tags=test

# Build the Lambda binary with Linux environment
echo "📦 Building Lambda binary for Linux..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/main cmd/lambda/main.go cmd/lambda/handler.go

# Verify the binary
if [ -f "bin/main" ]; then
    echo "✅ Lambda binary built successfully"
    echo "📊 Binary size: $(du -h bin/main | cut -f1)"
else
    echo "❌ Failed to build Lambda binary"
    exit 1
fi

# Create deployment package
echo "📦 Creating deployment package..."
cd bin/
zip -r ../stori-lambda.zip ./*
cd ..

echo "✅ Deployment package created: stori-lambda.zip"
echo "📊 Package size: $(du -h stori-lambda.zip | cut -f1)"

echo "🎉 Build completed successfully!"
echo ""
echo "Next steps:"
echo "1. Deploy with: ./scripts/deploy.sh"
echo "2. Or use SAM: sam deploy --guided"