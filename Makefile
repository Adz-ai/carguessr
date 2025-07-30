# Motors Price Guesser - Makefile

.PHONY: build run test clean dev prod swagger deps help build-prod run-prod install-service service-start service-stop service-status service-logs

# Default target
all: build

# Build the application
build:
	@echo "Building Motors Price Guesser..."
	go build -o bin/motors-guesser cmd/server/main.go

# Build optimized production binary
build-prod:
	@echo "🔨 Building optimized production binary..."
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/motors-guesser-linux-amd64 cmd/server/main.go
	@echo "✅ Production binary created: bin/motors-guesser-linux-amd64"

# Build and run production binary
run-prod: build-prod
	@echo "🚀 Running production binary..."
	@export GIN_MODE=release && ./bin/motors-guesser-linux-amd64

# Run the application
run:
	@echo "Starting Motors Price Guesser on port 8080..."
	go run cmd/server/main.go

# Run in development mode with hot reload
dev:
	@echo "🚀 Starting development server (with Swagger)..."
	@echo "📚 Swagger docs available at: http://localhost:8080/swagger/index.html"
	@export GIN_MODE=debug && go run cmd/server/main.go

# Run in production mode (no Swagger, optimized)
prod:
	@echo "🚀 Starting in PRODUCTION mode..."
	@echo "🔒 Swagger documentation is disabled for security"
	@export GIN_MODE=release && go run cmd/server/main.go

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

# Server deployment targets (Ubuntu/Linux)
install-service: build-prod
	@echo "⚙️  Installing as systemd service..."
	@sudo cp bin/motors-guesser-linux-amd64 /usr/local/bin/motors-price-guesser
	@sudo chmod +x /usr/local/bin/motors-price-guesser
	@echo "📝 Creating systemd service file..."
	@echo "[Unit]" | sudo tee /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "Description=Motors Price Guesser Game" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "After=network.target" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "[Service]" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "Type=simple" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "User=www-data" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "WorkingDirectory=/opt/motors-price-guesser" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "ExecStart=/usr/local/bin/motors-price-guesser" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "Environment=GIN_MODE=release" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "Environment=PORT=8080" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "Restart=always" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "RestartSec=10" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "[Install]" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@echo "WantedBy=multi-user.target" | sudo tee -a /etc/systemd/system/motors-price-guesser.service > /dev/null
	@sudo mkdir -p /opt/motors-price-guesser
	@sudo cp -r static /opt/motors-price-guesser/
	@sudo chown -R www-data:www-data /opt/motors-price-guesser
	@sudo systemctl daemon-reload
	@echo "✅ Service installed! Use 'make service-start' to start it"

service-start:
	@echo "🚀 Starting Motors Price Guesser service..."
	@sudo systemctl start motors-price-guesser
	@sudo systemctl enable motors-price-guesser
	@echo "✅ Service started and enabled for auto-start on boot"

service-stop:
	@echo "🛑 Stopping Motors Price Guesser service..."
	@sudo systemctl stop motors-price-guesser
	@echo "✅ Service stopped"

service-restart:
	@echo "🔄 Restarting Motors Price Guesser service..."
	@sudo systemctl restart motors-price-guesser
	@echo "✅ Service restarted"

service-status:
	@echo "📊 Motors Price Guesser service status:"
	@sudo systemctl status motors-price-guesser

service-logs:
	@echo "📋 Motors Price Guesser service logs (press Ctrl+C to exit):"
	@sudo journalctl -u motors-price-guesser -f

# Help
help:
	@echo "🎮 Motors Price Guesser - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  dev           - Run in development mode (with Swagger)"
	@echo "  prod          - Run in production mode (no Swagger)"
	@echo "  build         - Build development binary"
	@echo "  run           - Run the application"
	@echo ""
	@echo "Production:"
	@echo "  build-prod    - Build optimized Linux production binary"
	@echo "  run-prod      - Build and run production binary"
	@echo ""
	@echo "Server Deployment (Ubuntu/Linux):"
	@echo "  install-service - Install as systemd service"
	@echo "  service-start   - Start the service"
	@echo "  service-stop    - Stop the service"
	@echo "  service-restart - Restart the service"
	@echo "  service-status  - Check service status"
	@echo "  service-logs    - View service logs"
	@echo ""
	@echo "Development Tools:"
	@echo "  test          - Run tests"
	@echo "  test-scraper  - Test the Bonhams scraper"
	@echo "  check-listings- Check loaded car listings"
	@echo "  swagger       - Generate Swagger docs"
	@echo "  deps          - Install dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  kill          - Kill running servers"
	@echo "  setup         - Full development setup"
	@echo ""
	@echo "Usage Examples:"
	@echo "  make dev      # Start development server"
	@echo "  make prod     # Start production server"
	@echo "  make build-prod && scp bin/motors-guesser-linux-amd64 server:/tmp/"