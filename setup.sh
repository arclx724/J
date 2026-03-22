#!/bin/bash
# RoboKaty — Setup Script
set -e

echo "🐱 RoboKaty Setup"
echo "================================"

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed!"
    echo "Install Go 1.21+ from: https://go.dev/dl/"
    exit 1
fi

GO_VER=$(go version | awk '{print $3}')
echo "✅ Go version: $GO_VER"

# Check config.env
if [ ! -f "config.env" ]; then
    echo ""
    echo "⚠️  config.env not found!"
    echo "Creating from sample..."
    cp config.env.sample config.env
    echo "📝 Please fill in config.env with your values, then run this script again."
    exit 1
fi

# Check required env vars
source config.env 2>/dev/null || true
if [ -z "$BOT_TOKEN" ] || [ -z "$DATABASE_URI" ]; then
    echo "❌ BOT_TOKEN or DATABASE_URI is empty in config.env"
    exit 1
fi

echo "✅ config.env looks good"

# Create required directories
mkdir -p downloads cache
echo "✅ Directories created"

# Download dependencies
echo ""
echo "📦 Downloading dependencies..."
go mod tidy
echo "✅ Dependencies ready"

# Build
echo ""
echo "🔨 Building RoboKaty..."
go build -o robokaty .
echo "✅ Build successful!"

echo ""
echo "================================"
echo "🚀 Setup complete! Run with:"
echo "   ./robokaty"
echo ""
echo "📋 Or run as systemd service:"
echo "   sudo cp robokaty.service /etc/systemd/system/"
echo "   sudo systemctl enable --now robokaty"
echo "================================"
