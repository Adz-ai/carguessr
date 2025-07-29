#!/bin/bash

# Motors Price Guesser Deployment Script
set -e

echo "🚀 Deploying Motors Price Guesser to Homelab..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose > /dev/null 2>&1; then
    print_error "docker-compose not found. Please install docker-compose."
    exit 1
fi

# Create necessary directories
print_status "Creating directories..."
mkdir -p logs ssl

# Check for SSL certificates
if [ ! -f "ssl/cert.pem" ] || [ ! -f "ssl/key.pem" ]; then
    print_warning "SSL certificates not found. Generating self-signed certificates..."
    
    # Generate self-signed certificate
    openssl req -x509 -newkey rsa:4096 -keyout ssl/key.pem -out ssl/cert.pem -days 365 -nodes \
        -subj "/C=UK/ST=London/L=London/O=MotorsGuesser/CN=localhost"
    
    chmod 600 ssl/key.pem
    chmod 644 ssl/cert.pem
    
    print_success "Self-signed SSL certificates generated"
fi

# Build the application
print_status "Building Docker images..."

# Try Chrome version first
if docker-compose build --no-cache; then
    print_success "Chrome version built successfully"
else
    print_warning "Chrome version failed, trying Chromium version..."
    
    # Backup original Dockerfile and use Chromium version
    cp Dockerfile Dockerfile.chrome.backup
    cp Dockerfile.chromium Dockerfile
    
    if docker-compose build --no-cache; then
        print_success "Chromium version built successfully"
    else
        print_error "Both Chrome and Chromium builds failed"
        # Restore original Dockerfile
        cp Dockerfile.chrome.backup Dockerfile
        rm -f Dockerfile.chrome.backup
        exit 1
    fi
fi

# Start the services
print_status "Starting services..."
docker-compose up -d

# Wait for services to be healthy
print_status "Waiting for services to be ready..."
sleep 10

# Check health
if curl -f http://localhost:8080/api/health > /dev/null 2>&1; then
    print_success "Motors Price Guesser is running!"
    print_success "🌐 Access your app at:"
    print_success "   HTTP:  http://localhost"
    print_success "   HTTPS: https://localhost"
    print_success "   Direct: http://localhost:8080"
else
    print_error "Service is not responding. Checking logs..."
    docker-compose logs motors-guesser
    exit 1
fi

# Show resource usage
print_status "Container resource usage:"
docker stats --no-stream motors-price-guesser

echo ""
print_success "🎉 Deployment complete!"
echo ""
echo "Useful commands:"
echo "  📊 View logs: docker-compose logs -f"
echo "  🔄 Restart: docker-compose restart"
echo "  ⏹️  Stop: docker-compose down"
echo "  📈 Monitor: docker stats motors-price-guesser"