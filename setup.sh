#!/bin/bash

# Setup script for FlowC XDS Control Plane

echo "Setting up FlowC XDS Control Plane..."

# Initialize Go module if not already done
if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    go mod init github.com/flowc-labs/flowc
fi

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Tidy up dependencies
echo "Tidying dependencies..."
go mod tidy

# Build the project
echo "Building the project..."
go build -o bin/xds-server ./cmd/server

if [ $? -eq 0 ]; then
    echo "Build successful! XDS server binary created at bin/xds-server"
    echo "Run with: ./bin/xds-server"
else
    echo "Build failed. Please check the error messages above."
    exit 1
fi
