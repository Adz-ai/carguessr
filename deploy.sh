#!/bin/bash

echo "🚀 Deploying CarGuessr..."

# Run minification
echo "📦 Minifying assets..."
./minify.sh

# Build the application
echo "🔨 Building application..."
go build -o bin/server cmd/server/main.go

echo "✅ Deployment preparation complete!"
echo ""
echo "To run the server:"
echo "  ./bin/server"
echo ""
echo "Remember to:"
echo "1. Run minify.sh before each deployment"
echo "2. Update version numbers in HTML when making changes"
echo "3. Clear Cloudflare cache after deployment"