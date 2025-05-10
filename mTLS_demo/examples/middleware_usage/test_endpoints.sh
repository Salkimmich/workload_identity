#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Testing Authentication Endpoints"
echo "==============================="

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
    fi
}

# 1. Test health endpoint (no auth required)
echo -e "\n1. Testing health endpoint..."
response=$(curl -s -w "\n%{http_code}" http://localhost:8080/health)
status=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n1)
print_result $([ "$status" = "200" ] && [ "$body" = "OK" ] && echo 0 || echo 1) "Health endpoint"

# 2. Test rate limited endpoint
echo -e "\n2. Testing rate limited endpoint..."
response=$(curl -s -w "\n%{http_code}" http://localhost:8080/api/protected)
status=$(echo "$response" | tail -n1)
print_result $([ "$status" = "200" ] && echo 0 || echo 1) "Rate limited endpoint"

# 3. Test service endpoint with invalid API key
echo -e "\n3. Testing service endpoint with invalid API key..."
response=$(curl -s -w "\n%{http_code}" -H "X-API-Key: invalid-key" http://localhost:8080/api/service)
status=$(echo "$response" | tail -n1)
print_result $([ "$status" = "401" ] && echo 0 || echo 1) "Service endpoint with invalid API key"

# 4. Test service endpoint with valid API key
echo -e "\n4. Testing service endpoint with valid API key..."
response=$(curl -s -w "\n%{http_code}" -H "X-API-Key: test-key" http://localhost:8080/api/service)
status=$(echo "$response" | tail -n1)
print_result $([ "$status" = "200" ] && echo 0 || echo 1) "Service endpoint with valid API key"

# 5. Get tokens for users
echo -e "\n5. Getting tokens for users..."
# Get user token
user_token=$(curl -s -X POST http://localhost:8081/realms/demo/protocol/openid-connect/token \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=user" \
    -d "password=password" \
    -d "grant_type=password" \
    -d "client_id=demo-client" \
    -d "client_secret=demo-secret" | jq -r '.access_token')

# Get admin token
admin_token=$(curl -s -X POST http://localhost:8081/realms/demo/protocol/openid-connect/token \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=admin" \
    -d "password=password" \
    -d "grant_type=password" \
    -d "client_id=demo-client" \
    -d "client_secret=demo-secret" | jq -r '.access_token')

# 6. Test user endpoint with user token
echo -e "\n6. Testing user endpoint with user token..."
response=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $user_token" http://localhost:8080/api/user)
status=$(echo "$response" | tail -n1)
print_result $([ "$status" = "200" ] && echo 0 || echo 1) "User endpoint with user token"

# 7. Test user endpoint with admin token
echo -e "\n7. Testing user endpoint with admin token..."
response=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $admin_token" http://localhost:8080/api/user)
status=$(echo "$response" | tail -n1)
print_result $([ "$status" = "200" ] && echo 0 || echo 1) "User endpoint with admin token"

# 8. Test admin endpoint with user token (should fail)
echo -e "\n8. Testing admin endpoint with user token..."
response=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $user_token" http://localhost:8080/api/admin)
status=$(echo "$response" | tail -n1)
print_result $([ "$status" = "403" ] && echo 0 || echo 1) "Admin endpoint with user token"

# 9. Test admin endpoint with admin token
echo -e "\n9. Testing admin endpoint with admin token..."
response=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $admin_token" http://localhost:8080/api/admin)
status=$(echo "$response" | tail -n1)
print_result $([ "$status" = "200" ] && echo 0 || echo 1) "Admin endpoint with admin token"

# 10. Test rate limiting
echo -e "\n10. Testing rate limiting..."
echo "Making 15 requests in quick succession..."
for i in {1..15}; do
    response=$(curl -s -w "\n%{http_code}" http://localhost:8080/api/protected)
    status=$(echo "$response" | tail -n1)
    if [ "$status" = "429" ]; then
        echo -e "${GREEN}Rate limit hit on request $i${NC}"
        break
    fi
done

echo -e "\nTest Summary"
echo "============"
echo "All tests completed!" 