#!/bin/bash

# WhatsApp API Server Startup Script

echo "WhatsApp API Server"
echo "==================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "Error: Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or higher."
    exit 1
fi

echo "Go version: $GO_VERSION ✓"

# Check if SQLite is available
if ! command -v sqlite3 &> /dev/null; then
    echo "Warning: SQLite3 is not installed. The API will use the Go SQLite driver."
fi

# Install dependencies
echo "Installing dependencies..."
go mod tidy

# Set default port if not set
if [ -z "$PORT" ]; then
    export PORT=8080
fi

echo "Starting WhatsApp API server on port $PORT..."
echo "Press Ctrl+C to stop the server"
echo ""

# Run the server
go run *.go