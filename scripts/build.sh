#!/bin/bash

set -e

BASE_DIR=$(realpath $(dirname $0)/..)

# Build directory
BUILD_DIR="$BASE_DIR/build"

# Create build directory if not exists
mkdir -p $BUILD_DIR

cd $BASE_DIR

TARGET="tg-antispam"
if [ "$1" == "db" ]; then
    TARGET="db-migrate"
fi

# Build the binary
echo "Building $TARGET..."
if [ "$TARGET" == "db-migrate" ]; then
    go build -o $BUILD_DIR/$TARGET ./cmd/dbmigrate
else
    go build -o $BUILD_DIR/tg-antispam ./cmd/main
fi

# Copy configuration files
echo "Copying configuration files..."
mkdir -p $BUILD_DIR/configs
cp -r $BASE_DIR/configs/* $BUILD_DIR/configs/

# Copy scripts
echo "Copying scripts..."
mkdir -p $BUILD_DIR/scripts
cp $BASE_DIR/scripts/run.sh $BUILD_DIR/scripts/

# Make scripts executable
chmod +x $BUILD_DIR/scripts/*.sh

echo "Build completed. Binary is located at $BUILD_DIR/$TARGET"