#!/bin/bash
set -e

# Script to delete a webhook from a Telegram bot
# This resolves the "can't use getUpdates method while webhook is active" error

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Log function
log() {
  echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
  echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
  echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

# Check for token
TOKEN=""

# Try to get token from config file
if [ -f "config.yaml" ]; then
  TOKEN=$(grep -oP '(?<=token: ")[^"]*' config.yaml)
fi

# If no token in config, ask for it
if [ -z "$TOKEN" ]; then
  read -p "Enter your Telegram bot token: " TOKEN
  if [ -z "$TOKEN" ]; then
    error "No token provided. Exiting."
    exit 1
  fi
fi

log "Using token: ${TOKEN:0:5}...${TOKEN: -5}"

# Call the Telegram API to get information about the webhook
log "Checking current webhook status..."
WEBHOOK_INFO=$(curl -s "https://api.telegram.org/bot$TOKEN/getWebhookInfo")
echo -e "${YELLOW}Current webhook info:${NC}"
echo "$WEBHOOK_INFO" | grep -o '"url":"[^"]*' | cut -d '"' -f 4

# Delete the webhook
log "Deleting webhook..."
DELETE_RESULT=$(curl -s "https://api.telegram.org/bot$TOKEN/deleteWebhook")
if [[ "$DELETE_RESULT" == *"\"ok\":true"* ]]; then
  log "Webhook deleted successfully!"
else
  error "Failed to delete webhook. Response: $DELETE_RESULT"
  exit 1
fi

# Verify webhook is gone
log "Verifying webhook deletion..."
WEBHOOK_INFO=$(curl -s "https://api.telegram.org/bot$TOKEN/getWebhookInfo")
if [[ "$WEBHOOK_INFO" == *"\"url\":\"\""* ]]; then
  log "Webhook is now disabled. Your bot should work with polling mode."
else
  warn "Webhook may still be active. Response: $WEBHOOK_INFO"
fi

log "Done! You can now start your bot in polling mode."