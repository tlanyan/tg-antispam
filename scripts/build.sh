#!/bin/bash

set -e

# Build directory
BUILD_DIR="build"

# Create build directory if not exists
mkdir -p $BUILD_DIR

# Build the binary
echo "Building tg-antispam..."
go build -o $BUILD_DIR/tg-antispam ./cmd/tg-antispam

# Copy configuration files
echo "Copying configuration files..."
mkdir -p $BUILD_DIR/configs
cp -r configs/* $BUILD_DIR/configs/

# Copy scripts
echo "Copying scripts..."
mkdir -p $BUILD_DIR/scripts
cp scripts/run.sh $BUILD_DIR/scripts/

# Make scripts executable
chmod +x $BUILD_DIR/scripts/*.sh

echo "Build completed. Binary is located at $BUILD_DIR/tg-antispam"