# Makefile for Klayengo

.PHONY: build test clean version release help

# Build variables
BINARY_NAME=klayengo
VERSION?=$(shell git describe --tags --always --dirty)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags with version information
LDFLAGS=-ldflags "-X github.com/ambiyansyah-risyal/klayengo.Version=$(VERSION) -X github.com/ambiyansyah-risyal/klayengo.GitCommit=$(GIT_COMMIT) -X github.com/ambiyansyah-risyal/klayengo.BuildDate=$(BUILD_DATE)"

# Default target
all: test build

# Build the library
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

# Build with race detection
build-race:
	@echo "Building $(BINARY_NAME) with race detection..."
	go build $(LDFLAGS) -race -o bin/$(BINARY_NAME)-race .

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Show current version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Create a new git tag (usage: make tag VERSION=v1.1.0)
tag:
	@if [ -z "$(VERSION)" ]; then echo "Error: VERSION is required. Usage: make tag VERSION=v1.1.0"; exit 1; fi
	@echo "Creating git tag $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

# Release process (usage: make release VERSION=v1.1.0)
release: test
	@if [ -z "$(VERSION)" ]; then echo "Error: VERSION is required. Usage: make release VERSION=v1.1.0"; exit 1; fi
	@echo "Releasing $(VERSION)..."
	$(MAKE) tag VERSION=$(VERSION)
	@echo "Release $(VERSION) created successfully!"
	@echo "Don't forget to:"
	@echo "1. Create a GitHub release with the changelog"
	@echo "2. Update the version in README.md if needed"
	@echo "3. Update examples/go.mod files if the API changed"

# Install development dependencies
install-dev:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Run linting
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Update dependencies
deps:
	@echo "Updating dependencies..."
	go mod tidy
	go mod download

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the library"
	@echo "  build-race    - Build with race detection"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  bench         - Run benchmarks"
	@echo "  clean         - Clean build artifacts"
	@echo "  version       - Show current version information"
	@echo "  tag           - Create a git tag (requires VERSION=v1.x.x)"
	@echo "  release       - Create a release (requires VERSION=v1.x.x)"
	@echo "  install-dev   - Install development dependencies"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  deps          - Update dependencies"
	@echo "  help          - Show this help message"
