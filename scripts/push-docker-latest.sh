#!/bin/bash
set -e

# Script to build and push a 'latest' tagged Docker image to Docker Hub

echo "===> Building Docker image with SQLite support for AMD64"
docker build --platform linux/amd64 -t ceesaxp/cocktail-bot:latest .

echo "===> Logging in to Docker Hub"
echo "Please enter your Docker Hub credentials when prompted"
docker login

echo "===> Pushing image to Docker Hub"
docker push ceesaxp/cocktail-bot:latest

echo "===> Done! Image is now available as ceesaxp/cocktail-bot:latest"
echo "     You can now pull it with: docker pull ceesaxp/cocktail-bot:latest"