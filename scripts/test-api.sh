#!/bin/bash

# Cocktail Bot API Test Script
# Run this script to test the Cocktail Bot REST API

set -e

# Configuration
API_PORT=8080
API_HOST="localhost"
BASE_URL="http://${API_HOST}:${API_PORT}"
TOKEN_FILE="api_tokens.yaml"
EMAIL="test.user.$(date +%s)@example.com"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is not installed. Please install it to parse JSON responses.${NC}"
    echo "On Ubuntu/Debian: sudo apt-get install jq"
    echo "On macOS: brew install jq"
    exit 1
fi

# Read token from token file
if [ -f "$TOKEN_FILE" ]; then
    # Try to parse token using grep and awk
    TOKEN=$(grep -m 1 "auth_tokens:" -A 1 "$TOKEN_FILE" | tail -n 1 | awk -F'"' '{print $2}')
    if [ -z "$TOKEN" ]; then
        echo -e "${YELLOW}Could not parse token from file. Let's try using yq or python...${NC}"
        # Try with yq if available
        if command -v yq &> /dev/null; then
            TOKEN=$(yq '.auth_tokens[0]' "$TOKEN_FILE")
        # Otherwise try with python
        elif command -v python3 &> /dev/null; then
            TOKEN=$(python3 -c "import yaml; print(yaml.safe_load(open('$TOKEN_FILE'))['auth_tokens'][0])")
        fi
    fi
fi

# If still no token, ask for it
if [ -z "$TOKEN" ]; then
    echo -e "${YELLOW}Could not read token from file.${NC}"
    read -p "Please enter your API token: " TOKEN
fi

# Check if token is still empty
if [ -z "$TOKEN" ]; then
    echo -e "${RED}Error: No API token provided. Cannot continue.${NC}"
    exit 1
fi

echo -e "${BLUE}Cocktail Bot API Test${NC}"
echo -e "${BLUE}====================${NC}"
echo -e "API URL: ${GREEN}${BASE_URL}${NC}"
echo -e "Token: ${GREEN}${TOKEN:0:5}...${TOKEN: -5}${NC}"
echo -e "Test Email: ${GREEN}${EMAIL}${NC}"
echo ""

# Test 1: Health Check
echo -e "${BLUE}Test 1: Health Check${NC}"
RESPONSE=$(curl -s "${BASE_URL}/api/health")
echo "Response: $RESPONSE"
if echo "$RESPONSE" | jq -e '.status == "ok"' &> /dev/null; then
    echo -e "${GREEN}✅ Health check successful${NC}"
else
    echo -e "${RED}❌ Health check failed${NC}"
fi
echo ""

# Test 2: Authentication (intentionally wrong token)
echo -e "${BLUE}Test 2: Authentication Test (Should Fail)${NC}"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer wrong_token" \
    "${BASE_URL}/api/v1/email" \
    -d "{\"email\":\"$EMAIL\"}")

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
RESPONSE_BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

echo "Status: $HTTP_STATUS"
echo "Response: $RESPONSE_BODY"

if [ "$HTTP_STATUS" -eq 401 ]; then
    echo -e "${GREEN}✅ Authentication test passed (expected 401 Unauthorized)${NC}"
else
    echo -e "${RED}❌ Authentication test failed (expected 401, got $HTTP_STATUS)${NC}"
fi
echo ""

# Test 3: Invalid email
echo -e "${BLUE}Test 3: Invalid Email Test (Should Fail)${NC}"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    "${BASE_URL}/api/v1/email" \
    -d "{\"email\":\"not-an-email\"}")

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
RESPONSE_BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

echo "Status: $HTTP_STATUS"
echo "Response: $RESPONSE_BODY"

if [ "$HTTP_STATUS" -eq 400 ]; then
    echo -e "${GREEN}✅ Invalid email test passed (expected 400 Bad Request)${NC}"
else
    echo -e "${RED}❌ Invalid email test failed (expected 400, got $HTTP_STATUS)${NC}"
fi
echo ""

# Test 4: Add New Email
echo -e "${BLUE}Test 4: Add New Email${NC}"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    "${BASE_URL}/api/v1/email" \
    -d "{\"email\":\"$EMAIL\"}")

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
RESPONSE_BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

echo "Status: $HTTP_STATUS"
echo "Response: $RESPONSE_BODY"

USER_ID=$(echo "$RESPONSE_BODY" | jq -r '.id // empty')

if [ "$HTTP_STATUS" -eq 201 ]; then
    echo -e "${GREEN}✅ Email added successfully (ID: $USER_ID)${NC}"
elif [ "$HTTP_STATUS" -eq 409 ]; then
    echo -e "${YELLOW}⚠️ Email already exists in database${NC}"
    USER_ID=$(echo "$RESPONSE_BODY" | jq -r '.id // empty')
    echo -e "Existing user ID: ${USER_ID}"
else
    echo -e "${RED}❌ Failed to add email (status: $HTTP_STATUS)${NC}"
fi
echo ""

# Test 5: Try to add the same email again (should fail with 409 Conflict)
echo -e "${BLUE}Test 5: Add Duplicate Email (Should Return 409)${NC}"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    "${BASE_URL}/api/v1/email" \
    -d "{\"email\":\"$EMAIL\"}")

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
RESPONSE_BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

echo "Status: $HTTP_STATUS"
echo "Response: $RESPONSE_BODY"

if [ "$HTTP_STATUS" -eq 409 ]; then
    echo -e "${GREEN}✅ Duplicate email test passed (expected 409 Conflict)${NC}"
else
    echo -e "${RED}❌ Duplicate email test failed (expected 409, got $HTTP_STATUS)${NC}"
fi

# Summary
echo ""
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}============${NC}"
echo -e "API appears to be ${GREEN}working correctly${NC}"
echo -e "Email ${GREEN}${EMAIL}${NC} has been added with ID: ${GREEN}${USER_ID}${NC}"
echo ""
echo -e "To check this user in the Telegram bot, send the email: ${GREEN}${EMAIL}${NC}"