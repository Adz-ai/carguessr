# Motors Price Guesser - Makefile

.PHONY: build run test clean dev swagger deps help

# Default target
all: build

# Build the application
build:
	@echo "Building Motors Price Guesser..."
	go build -o bin/motors-guesser cmd/server/main.go

# Run the application
run:
	@echo "Starting Motors Price Guesser on port 8080..."
	go run cmd/server/main.go

# Run in development mode with hot reload
dev:
	@echo "Starting development server..."
	go run cmd/server/main.go

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	swag init -g cmd/server/main.go -o docs/

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Test the scraper directly
test-scraper:
	@echo "Testing Motors.co.uk scraper..."
	curl -s http://localhost:8080/api/test-scraper | jq

# Check available car listings
check-listings:
	@echo "Checking loaded car listings..."
	curl -s http://localhost:8080/api/listings | jq '.count'

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Format code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting Go code..."
	golangci-lint run

# Kill any running servers
kill:
	@echo "Killing any running servers..."
	pkill -f "cmd/server/main.go" || pkill -f "go run" || true

# Full development setup
setup: deps swagger
	@echo "Development setup complete!"

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  dev           - Run in development mode"
	@echo "  test          - Run tests"
	@echo "  test-scraper  - Test the Motors scraper"
	@echo "  check-listings- Check loaded car listings"
	@echo "  swagger       - Generate Swagger docs"
	@echo "  deps          - Install dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  kill          - Kill running servers"
	@echo "  setup         - Full development setup"
	@echo "  help          - Show this help"