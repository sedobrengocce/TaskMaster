.PHONY: all build dev clean test migrate-up migrate-down watch-assets

# Variables
BINARY_NAME=taskmaster
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

# Development tools
AIR=air

all: build

# Development
dev: install-tools
	make -j4 watch-go

# Watchers
watch-go:
	$(AIR)

# Build commands
build: clean build-go

build-go:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/server

# Testing
test:
	go test -v ./...

# Dependencies
vendor: go.mod go.sum
	go mod vendor
	go mod tidy

# Cleanup
clean:
	rm -rf \
		$(BINARY_NAME) \
		tmp/* \
		vendor/

# Install development tools
install-tools: 
	go install github.com/air-verse/air@latest

# Help
help:
	@echo "Available commands:"
	@echo "  make dev          - Start local development with file watching"
	@echo "  make build        - Build the application"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install-tools - Install development tools"
