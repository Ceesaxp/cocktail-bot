#!/bin/bash

# Test WebUI redirect behavior

echo "Testing WebUI redirect behavior..."
echo ""

# Test 1: Check if accessing root redirects to login
echo "Test 1: Accessing / without authentication"
response=$(curl -s -o /dev/null -w "%{http_code}:%{redirect_url}" -D - "http://localhost:8081/" | grep -E "(HTTP/|Location:)")
echo "Response headers:"
echo "$response"
echo ""

# Test 2: Check the actual redirect with verbose output
echo "Test 2: Following redirects"
curl -v -L "http://localhost:8081/" 2>&1 | grep -E "(< HTTP/|< Location:|> GET)"
echo ""

# Test 3: Check if login page is accessible
echo "Test 3: Accessing /login directly"
response_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8081/login")
echo "Login page response code: $response_code"
echo ""

# Test 4: Check server is actually running
echo "Test 4: Checking if server is running on port 8081"
nc -zv localhost 8081 2>&1