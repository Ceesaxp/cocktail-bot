#!/bin/bash
set -e

# Script to help debug API token configuration issues
# Usage: ./scripts/debug-api-tokens.sh

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CONFIG_FILE="config.yaml"
TOKENS_FILE=""
CONTAINER_NAME="cocktail-bot"

echo -e "${BLUE}Cocktail Bot API Token Diagnostics${NC}"
echo -e "${BLUE}==================================${NC}"

# Check if running inside Docker container
IN_CONTAINER=false
if [ -f "/.dockerenv" ]; then
  IN_CONTAINER=true
  echo -e "${YELLOW}Running inside a Docker container${NC}"
fi

# Check config.yaml
if [ -f "$CONFIG_FILE" ]; then
  echo -e "${GREEN}✓ Found config.yaml${NC}"
  API_ENABLED=$(grep -oP '(?<=enabled: )(true|false)' "$CONFIG_FILE" | head -1 || echo "not found")
  echo -e "  API enabled: ${YELLOW}$API_ENABLED${NC}"
  
  # Extract tokens file path
  TOKENS_FILE=$(grep -oP '(?<=tokens_file: ")[^"]*' "$CONFIG_FILE" || echo "")
  if [ -n "$TOKENS_FILE" ]; then
    echo -e "  Tokens file: ${YELLOW}$TOKENS_FILE${NC}"
  else
    echo -e "  ${RED}⨯ No tokens_file defined in config.yaml${NC}"
  fi
else
  echo -e "${RED}⨯ config.yaml not found${NC}"
fi

# Check environment variables
if [ -n "$COCKTAILBOT_API_TOKENS" ]; then
  # Count tokens
  TOKEN_COUNT=$(echo "$COCKTAILBOT_API_TOKENS" | awk -F',' '{print NF}')
  TOKEN_PREVIEW=$(echo "$COCKTAILBOT_API_TOKENS" | cut -c1-10)
  echo -e "${GREEN}✓ Found API tokens in environment variable${NC}"
  echo -e "  Tokens count: ${YELLOW}$TOKEN_COUNT${NC}"
  echo -e "  First token preview: ${YELLOW}${TOKEN_PREVIEW}...${NC}"
else
  echo -e "${YELLOW}⚠ No API tokens found in environment variable${NC}"
fi

# Check tokens file
if [ -n "$TOKENS_FILE" ] && [ -f "$TOKENS_FILE" ]; then
  echo -e "${GREEN}✓ Tokens file exists: $TOKENS_FILE${NC}"
  
  # Try to parse tokens
  TOKEN_COUNT=$(grep -c -oP '(?<=- )[A-Za-z0-9_-]+' "$TOKENS_FILE" || echo "0")
  if [ "$TOKEN_COUNT" -gt 0 ]; then
    FIRST_TOKEN=$(grep -oP '(?<=- )[A-Za-z0-9_-]+' "$TOKENS_FILE" | head -1)
    TOKEN_PREVIEW=${FIRST_TOKEN:0:10}
    echo -e "  Tokens count: ${YELLOW}$TOKEN_COUNT${NC}"
    echo -e "  First token preview: ${YELLOW}${TOKEN_PREVIEW}...${NC}"
  else
    echo -e "  ${RED}⨯ No tokens found in $TOKENS_FILE${NC}"
    echo -e "  ${YELLOW}File content:${NC}"
    cat "$TOKENS_FILE" | head -5
  fi
else
  if [ -n "$TOKENS_FILE" ]; then
    echo -e "${RED}⨯ Tokens file does not exist: $TOKENS_FILE${NC}"
  else
    echo -e "${YELLOW}⚠ No tokens file path specified${NC}"
  fi
fi

# Check Docker container if available
if command -v docker &> /dev/null && ! $IN_CONTAINER; then
  if docker ps -q --filter "name=$CONTAINER_NAME" | grep -q .; then
    echo -e "${GREEN}✓ Docker container '$CONTAINER_NAME' is running${NC}"
    
    # Check environment variables in container
    CONTAINER_ENV=$(docker exec $CONTAINER_NAME env | grep COCKTAILBOT_API_TOKENS || echo "")
    if [ -n "$CONTAINER_ENV" ]; then
      echo -e "  ${GREEN}✓ API tokens found in container environment${NC}"
      echo -e "  ${YELLOW}$CONTAINER_ENV${NC}"
    else
      echo -e "  ${RED}⨯ No API tokens found in container environment${NC}"
    fi
    
    # Check if config file is mounted
    docker exec $CONTAINER_NAME ls -la /app/config.yaml &> /dev/null
    if [ $? -eq 0 ]; then
      echo -e "  ${GREEN}✓ config.yaml is mounted in the container${NC}"
    else
      echo -e "  ${RED}⨯ config.yaml is not mounted in the container${NC}"
    fi
    
    # Check if tokens file is mounted
    if [ -n "$TOKENS_FILE" ]; then
      CONTAINER_TOKENS_FILE="/app/$(basename $TOKENS_FILE)"
      docker exec $CONTAINER_NAME ls -la $CONTAINER_TOKENS_FILE &> /dev/null
      if [ $? -eq 0 ]; then
        echo -e "  ${GREEN}✓ Tokens file is mounted in the container${NC}"
        CONTAINER_TOKENS=$(docker exec $CONTAINER_NAME cat $CONTAINER_TOKENS_FILE)
        echo -e "  ${YELLOW}Container tokens file content:${NC}"
        echo "$CONTAINER_TOKENS" | head -5
      else
        echo -e "  ${RED}⨯ Tokens file is not correctly mounted in the container${NC}"
        echo -e "  ${YELLOW}Looking for: $CONTAINER_TOKENS_FILE${NC}"
      fi
    fi
  else
    echo -e "${YELLOW}⚠ No Docker container named '$CONTAINER_NAME' is running${NC}"
  fi
else
  if $IN_CONTAINER; then
    # Inside container, check for tokens file
    if [ -n "$TOKENS_FILE" ]; then
      if [ -f "$TOKENS_FILE" ]; then
        echo -e "${GREEN}✓ Tokens file exists inside container: $TOKENS_FILE${NC}"
        echo -e "  ${YELLOW}Content:${NC}"
        cat "$TOKENS_FILE" | head -5
      else
        echo -e "${RED}⨯ Tokens file does not exist inside container: $TOKENS_FILE${NC}"
      fi
    fi
  else
    echo -e "${YELLOW}⚠ Docker command not available${NC}"
  fi
fi

echo -e "\n${BLUE}API Token Status Summary:${NC}"
if [ -n "$COCKTAILBOT_API_TOKENS" ]; then
  echo -e "${GREEN}✓ API tokens are available via environment variable${NC}"
elif [ -n "$TOKENS_FILE" ] && [ -f "$TOKENS_FILE" ] && [ "$TOKEN_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ API tokens are available via tokens file${NC}"
else
  echo -e "${RED}⨯ No API tokens available!${NC}"
  echo -e "${YELLOW}Try setting the COCKTAILBOT_API_TOKENS environment variable:${NC}"
  echo "export COCKTAILBOT_API_TOKENS=\"your-token-here\""
fi

echo -e "\n${BLUE}Recommended solutions:${NC}"
echo -e "1. ${GREEN}Use environment variables (preferred for Docker)${NC}"
echo "   docker run -e COCKTAILBOT_API_TOKENS=\"token1,token2\" ..."
echo -e "2. ${GREEN}Initialize tokens and mount the file${NC}"
echo "   ./scripts/manage-api-tokens.sh init"
echo "   docker run -v \$(pwd)/api_tokens.yaml:/app/api_tokens.yaml ..."
echo -e "3. ${GREEN}To test the API with current token:${NC}"
echo "   ./scripts/test-api.sh"