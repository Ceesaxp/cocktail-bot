#!/bin/bash

set -e

echo "Setting up Cocktail Bot environment..."

# Create directories
mkdir -p data

# Copy config file if it doesn't exist
if [ ! -f config.yaml ]; then
    echo "Creating config.yaml from template..."
    cp config.yaml.example config.yaml
    echo "Please edit config.yaml to set your Telegram bot token."
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Build the bot
echo "Building Cocktail Bot..."
go build -o cocktail-bot ./cmd/bot
go build -o importcsv ./cmd/importcsv

# Create sample CSV file if it doesn't exist
if [ ! -f data/users.csv ]; then
    echo "Creating sample users.csv file..."
    echo "ID,Email,Date Added,Already Consumed" > data/users.csv
    echo "1,user1@example.com,$(date -u +"%Y-%m-%dT%H:%M:%SZ")," >> data/users.csv
    echo "2,user2@example.com,$(date -u +"%Y-%m-%dT%H:%M:%SZ")," >> data/users.csv
    echo "Sample users.csv created. You can replace it with your own data."
fi

echo ""
echo "Setup complete!"
echo "Next steps:"
echo "1. Edit config.yaml to set your Telegram bot token"
echo "2. Run ./cocktail-bot to start the bot"
echo "3. Or use Docker: docker-compose up -d"
echo ""
echo "For importing emails from a CSV file, use:"
echo "./importcsv -input your_emails.csv -output data/users.csv -column 1"
echo ""
