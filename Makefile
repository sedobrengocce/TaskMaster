.PHONY: all build dev clean test migrate-up migrate-down watch-assets

# Variables
BINARY_NAME=taskmaster
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

# Variables read from .env file
include .env

# Development tools
AIR=air
MIGRATE=migrate

all: build

# Development
dev: install-tools 
	make migrate-up
	make -j4 watch-go

# Watchers
watch-go:
	$(AIR)

migrate-up:
	@echo "Running migrations up..."
	$(MIGRATE) -path db/migrations -database "mysql://$(DB_USER):$(DB_PASSWORD)@tcp(db:3306)/$(DB_NAME)" up

# Build commands
build: clean build-go

build-go:
	make migrate-up
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
	go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Help
help:
	@echo "Available commands:"
	@echo "  make dev          - Start local development with file watching"
	@echo "  make build        - Build the application"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install-tools - Install development tools"
