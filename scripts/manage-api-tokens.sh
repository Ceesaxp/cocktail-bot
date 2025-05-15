#!/bin/bash

# Script to manage API tokens for the Cocktail Bot
# Usage:
#   ./scripts/manage-api-tokens.sh list                  # List all tokens
#   ./scripts/manage-api-tokens.sh add [token]           # Add a new token (generates random if not provided)
#   ./scripts/manage-api-tokens.sh remove [token]        # Remove a token
#   ./scripts/manage-api-tokens.sh reset                 # Reset tokens file with a new random token
#   ./scripts/manage-api-tokens.sh init                  # Initialize tokens file if it doesn't exist

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Variables
TOKENS_FILE="api_tokens.yaml"
CONFIG_FILE="config.yaml"

# Functions
generate_token() {
  # Generate a random 32-character token
  LC_ALL=C tr -dc 'A-Za-z0-9_-' < /dev/urandom | head -c 32
}

load_config() {
  # Check if config.yaml exists and API is enabled
  if [ -f "$CONFIG_FILE" ]; then
    # Extract tokens file path from config.yaml
    CONFIG_TOKENS_FILE=$(grep -oP '(?<=tokens_file: ")[^"]*' "$CONFIG_FILE" || echo "")
    if [ -n "$CONFIG_TOKENS_FILE" ]; then
      TOKENS_FILE="$CONFIG_TOKENS_FILE"
    fi
    
    # Check if API is enabled
    API_ENABLED=$(grep -oP '(?<=enabled: )(true|false)' "$CONFIG_FILE" | head -1 || echo "false")
    if [ "$API_ENABLED" != "true" ]; then
      echo -e "${YELLOW}Warning: API is not enabled in $CONFIG_FILE${NC}"
    fi
  fi
}

ensure_tokens_file() {
  # Create tokens file if it doesn't exist
  if [ ! -f "$TOKENS_FILE" ]; then
    echo -e "${YELLOW}Tokens file $TOKENS_FILE not found, creating...${NC}"
    echo "auth_tokens:" > "$TOKENS_FILE"
  fi
}

list_tokens() {
  # List all tokens in the file
  echo -e "${BLUE}API Tokens in $TOKENS_FILE:${NC}"
  grep -oP '(?<=- )[A-Za-z0-9_-]+' "$TOKENS_FILE" | nl || echo "No tokens found"
}

add_token() {
  # Add a new token to the file
  local token="$1"
  if [ -z "$token" ]; then
    token=$(generate_token)
  fi
  
  # Check if token already exists
  if grep -q "$token" "$TOKENS_FILE"; then
    echo -e "${YELLOW}Token already exists in $TOKENS_FILE${NC}"
    return
  fi
  
  # Add token to file
  # First check if auth_tokens: exists
  if ! grep -q "auth_tokens:" "$TOKENS_FILE"; then
    echo "auth_tokens:" > "$TOKENS_FILE"
  fi
  
  # Check if there are any tokens already
  local indent="    "
  if ! grep -q "^$indent- " "$TOKENS_FILE"; then
    echo "$indent- $token" >> "$TOKENS_FILE"
  else
    # Append to existing tokens
    sed -i '' -e "/auth_tokens:/a\\
$indent- $token
" "$TOKENS_FILE"
  fi
  
  echo -e "${GREEN}Token added: $token${NC}"
}

remove_token() {
  # Remove a token from the file
  local token="$1"
  if [ -z "$token" ]; then
    echo -e "${RED}Error: Token not specified${NC}"
    exit 1
  fi
  
  # Check if token exists
  if ! grep -q "$token" "$TOKENS_FILE"; then
    echo -e "${YELLOW}Token not found in $TOKENS_FILE${NC}"
    return
  fi
  
  # Remove token from file
  sed -i '' -e "/$token/d" "$TOKENS_FILE"
  echo -e "${GREEN}Token removed: $token${NC}"
}

reset_tokens() {
  # Reset tokens file with a new random token
  local token=$(generate_token)
  echo "auth_tokens:" > "$TOKENS_FILE"
  echo "    - $token" >> "$TOKENS_FILE"
  echo -e "${GREEN}Tokens reset. New token: $token${NC}"
}

# Main script
load_config

# Check command
if [ $# -eq 0 ]; then
  echo -e "${YELLOW}Usage:${NC}"
  echo "  $0 list                  # List all tokens"
  echo "  $0 add [token]           # Add a new token (generates random if not provided)"
  echo "  $0 remove [token]        # Remove a token"
  echo "  $0 reset                 # Reset tokens file with a new random token"
  echo "  $0 init                  # Initialize tokens file if it doesn't exist"
  exit 1
fi

command="$1"
token="$2"

case "$command" in
  "list")
    ensure_tokens_file
    list_tokens
    ;;
  "add")
    ensure_tokens_file
    add_token "$token"
    ;;
  "remove")
    ensure_tokens_file
    remove_token "$token"
    ;;
  "reset")
    reset_tokens
    ;;
  "init")
    if [ ! -f "$TOKENS_FILE" ]; then
      token=$(generate_token)
      echo "auth_tokens:" > "$TOKENS_FILE"
      echo "    - $token" >> "$TOKENS_FILE"
      echo -e "${GREEN}Tokens file initialized with token: $token${NC}"
    else
      echo -e "${YELLOW}Tokens file $TOKENS_FILE already exists${NC}"
      list_tokens
    fi
    ;;
  *)
    echo -e "${RED}Unknown command: $command${NC}"
    exit 1
    ;;
esac

# Remind user to update config.yaml if needed
if [ ! -f "$CONFIG_FILE" ] || ! grep -q "tokens_file: \"$TOKENS_FILE\"" "$CONFIG_FILE"; then
  echo -e "${YELLOW}Don't forget to update $CONFIG_FILE with:${NC}"
  echo "api:"
  echo "  enabled: true"
  echo "  tokens_file: \"$TOKENS_FILE\""
fi