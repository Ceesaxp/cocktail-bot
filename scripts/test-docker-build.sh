#!/bin/bash
set -e

# Test script for local Docker build with SQLite support
echo "===> Building Docker image with SQLite support"
# Detect platform
platform=$(uname -m)
if [ "$platform" = "arm64" ]; then
  echo "Building for ARM64 platform (Apple Silicon)"
  docker build --platform linux/arm64 -t cocktail-bot:test .
else
  echo "Building for AMD64 platform"
  docker build -t cocktail-bot:test .
fi

echo "===> Creating test directory"
mkdir -p ./test-data
chmod 777 ./test-data

echo "===> Making sure config is set to use SQLite"
if ! grep -q "type: \"sqlite\"" config.yaml; then
  echo "⚠️  Warning: config.yaml might not be set to use SQLite."
  echo "⚠️  Please verify the database configuration in config.yaml."
  echo "⚠️  It should contain:"
  echo "⚠️  database:"
  echo "⚠️    type: \"sqlite\""
  echo "⚠️    connection_string: \"/app/data/users.sqlite\""
fi

echo "===> Running container with SQLite database"
docker run --rm -v "$(pwd)/test-data:/app/data" \
  -v "$(pwd)/config.yaml:/app/config.yaml" \
  -p 8080:8080 \
  --name cocktail-bot-test \
  -d cocktail-bot:test

echo "===> Container started, waiting 5 seconds..."
sleep 5

echo "===> Checking container logs"
docker logs cocktail-bot-test

# Check if API is enabled and responding
if grep -q "api:" config.yaml && grep -q "enabled: true" config.yaml; then
  echo "===> Testing API endpoint (if enabled)"
  curl -s -o /dev/null -w "API Status: %{http_code}\n" http://localhost:8080/health || echo "API not responding"
fi

echo "===> Stopping container"
docker stop cocktail-bot-test

echo "===> Test completed, check logs above for any errors"
echo "===> SQLite database should be at ./test-data/users.sqlite"
ls -la ./test-data/