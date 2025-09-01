.PHONY: build test clean docker-build docker-down deps lint fmt vet

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build parameters
BINARY_NAME=processor
BINARY_PATH=./cmd/processor
BUILD_DIR=./build

# Default target
all: clean deps fmt vet test build

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt:
	@echo "Formatting code..."
	$(GOFMT) -w .

vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linter (if golangci-lint is installed)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build the application
build:
	@echo "Building application..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo -ldflags '-extldflags "-static"' -o $(BUILD_DIR)/$(BINARY_NAME) $(BINARY_PATH)


# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t stori-processor:latest .

# Hybrid Email Commands
docker-run-mailhog: docker-build
	@echo "Running with MailHog for email testing..."
	@echo "Web UI will be available at: http://localhost:8025"
	docker-compose up --build

docker-run-gmail: docker-build
	@echo "Running with Gmail for real email delivery..."
	docker-compose -f docker-compose.yml -f docker-compose.gmail.yml up --build

docker-run-interactive: docker-build
	@echo "Select email mode:"
	@echo "1) MailHog (development/testing)"
	@echo "2) Gmail (real email delivery)"
	@read -p "Enter choice (1 or 2): " choice; \
	case $$choice in \
		1) $(MAKE) docker-run-mailhog ;; \
		2) $(MAKE) docker-run-gmail ;; \
		*) echo "Invalid choice. Using MailHog mode."; $(MAKE) docker-run-mailhog ;; \
	esac

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down
	docker-compose -f docker-compose.yml -f docker-compose.gmail.yml down

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker system prune -f

# Development commands
dev-setup:
	@echo "Setting up development environment..."
	$(MAKE) deps
	@if ! command -v golangci-lint > /dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi


# Help
help:
	@echo "🏗️  Stori Challenge - Available Commands"
	@echo ""
	@echo "📦 Development:"
	@echo "  deps                 - Download and tidy Go dependencies"
	@echo "  fmt                  - Format Go code"
	@echo "  vet                  - Run go vet static analysis"
	@echo "  lint                 - Run golangci-lint (if installed)"
	@echo "  test                 - Run tests with race detection"
	@echo "  test-coverage        - Run tests and generate HTML coverage report"
	@echo "  build                - Build application for Linux (Docker compatible)"
	@echo "  clean                - Clean build artifacts and coverage files"
	@echo ""
	@echo "🐳 Docker - Hybrid Email System:"
	@echo "  docker-build         - Build Docker image"
	@echo "  docker-run-mailhog   - 🧪 Development mode (MailHog email capture)"
	@echo "  docker-run-gmail     - 📧 Production mode (real Gmail delivery)"
	@echo "  docker-run-interactive - 🔄 Interactive mode selection"
	@echo "  docker-down          - Stop all containers"
	@echo "  docker-clean         - Clean containers and Docker resources"
	@echo ""
	@echo "🛠️  Setup:"
	@echo "  dev-setup            - Setup development environment + golangci-lint"
	@echo "  help                 - Show this help"
	@echo ""
	@echo "💡 Quick Start:"
	@echo "  make docker-run-mailhog  # Test with email capture (http://localhost:8025)"
	@echo "  make docker-run-gmail    # Send real emails via Gmail SMTP"