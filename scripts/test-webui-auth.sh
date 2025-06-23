#!/bin/bash

# Test script for WebUI authentication using tokens

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
WEBUI_URL="http://localhost:8081"
TEST_TOKEN="test_token_123"

echo "=== WebUI Token Authentication Test ==="
echo ""

# Test 1: Access protected page without token
echo -e "${YELLOW}Test 1: Access protected page without authentication${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" "${WEBUI_URL}/")
if [ "$response" = "303" ] || [ "$response" = "302" ]; then
    echo -e "${GREEN}✓ Correctly redirected to login page${NC}"
else
    echo -e "${RED}✗ Expected redirect (302/303), got $response${NC}"
fi
echo ""

# Test 2: Login with invalid token
echo -e "${YELLOW}Test 2: Login with invalid token${NC}"
response=$(curl -s -X POST \
    -F "token=invalid_token" \
    -F "redirect=/" \
    "${WEBUI_URL}/login" \
    -w "%{http_code}" \
    -o /tmp/login_response.html)

if grep -q "Invalid authentication token" /tmp/login_response.html 2>/dev/null; then
    echo -e "${GREEN}✓ Correctly rejected invalid token${NC}"
else
    echo -e "${RED}✗ Invalid token was not rejected properly${NC}"
fi
echo ""

# Test 3: Login with valid token (assuming TEST_TOKEN is configured)
echo -e "${YELLOW}Test 3: Login with valid token${NC}"
echo -e "${YELLOW}Note: Make sure to add '${TEST_TOKEN}' to your api.auth_tokens in config.yaml${NC}"
response=$(curl -s -X POST \
    -F "token=${TEST_TOKEN}" \
    -F "redirect=/" \
    -c /tmp/webui_cookies.txt \
    "${WEBUI_URL}/login" \
    -w "%{http_code}" \
    -o /dev/null)

if [ "$response" = "303" ] || [ "$response" = "302" ]; then
    echo -e "${GREEN}✓ Login successful (redirected)${NC}"
else
    echo -e "${RED}✗ Login failed with status $response${NC}"
fi
echo ""

# Test 4: Access protected page with token cookie
echo -e "${YELLOW}Test 4: Access protected page with authentication cookie${NC}"
response=$(curl -s \
    -b /tmp/webui_cookies.txt \
    "${WEBUI_URL}/" \
    -w "%{http_code}" \
    -o /tmp/dashboard.html)

if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Successfully accessed protected page${NC}"
    if grep -q "Dashboard" /tmp/dashboard.html 2>/dev/null; then
        echo -e "${GREEN}✓ Dashboard content loaded${NC}"
    fi
else
    echo -e "${RED}✗ Failed to access protected page, got status $response${NC}"
fi
echo ""

# Test 5: Logout
echo -e "${YELLOW}Test 5: Logout${NC}"
response=$(curl -s \
    -b /tmp/webui_cookies.txt \
    "${WEBUI_URL}/logout" \
    -w "%{http_code}" \
    -o /dev/null)

if [ "$response" = "303" ] || [ "$response" = "302" ]; then
    echo -e "${GREEN}✓ Logout successful${NC}"
else
    echo -e "${RED}✗ Logout failed with status $response${NC}"
fi
echo ""

# Cleanup
rm -f /tmp/webui_cookies.txt /tmp/login_response.html /tmp/dashboard.html

echo "=== Test Complete ==="
echo ""
echo -e "${YELLOW}Configuration Notes:${NC}"
echo "1. Make sure WebUI is enabled in your config.yaml:"
echo "   webui:"
echo "     enabled: true"
echo "     port: 8081"
echo ""
echo "2. Add authentication tokens to your config.yaml:"
echo "   api:"
echo "     auth_tokens:"
echo "       - \"${TEST_TOKEN}\""
echo ""
echo "3. The WebUI shares authentication tokens with the API"