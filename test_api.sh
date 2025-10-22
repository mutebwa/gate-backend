#!/bin/bash

# GateKeeper API Testing Script
# Tests all authentication and basic endpoints

echo "🧪 GateKeeper API Testing Script"
echo "================================"
echo ""

BASE_URL="http://localhost:8080"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to test endpoint
test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    local auth=$5
    
    echo -n "Testing: $name... "
    
    if [ -z "$auth" ]; then
        if [ -z "$data" ]; then
            response=$(curl -s -X $method "$BASE_URL$endpoint")
        else
            response=$(curl -s -X $method "$BASE_URL$endpoint" -H "Content-Type: application/json" -d "$data")
        fi
    else
        if [ -z "$data" ]; then
            response=$(curl -s -X $method "$BASE_URL$endpoint" -H "Authorization: Bearer $auth")
        else
            response=$(curl -s -X $method "$BASE_URL$endpoint" -H "Content-Type: application/json" -H "Authorization: Bearer $auth" -d "$data")
        fi
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}"
        echo "  Response: $response"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo "1️⃣  Testing Health Endpoint"
echo "----------------------------"
test_endpoint "Health Check" "GET" "/health"
echo ""

echo "2️⃣  Testing Authentication"
echo "----------------------------"

# Test admin login
echo -n "Admin Login... "
admin_response=$(curl -s -X POST "$BASE_URL/api/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"password"}')
admin_token=$(echo $admin_response | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -n "$admin_token" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "  Token: ${admin_token:0:50}..."
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test supervisor login
echo -n "Supervisor Login... "
supervisor_response=$(curl -s -X POST "$BASE_URL/api/login" -H "Content-Type: application/json" -d '{"username":"supervisor_john","password":"password"}')
supervisor_token=$(echo $supervisor_response | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -n "$supervisor_token" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "  Token: ${supervisor_token:0:50}..."
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test operator login
echo -n "Operator Login... "
operator_response=$(curl -s -X POST "$BASE_URL/api/login" -H "Content-Type: application/json" -d '{"username":"op_east","password":"password"}')
operator_token=$(echo $operator_response | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -n "$operator_token" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "  Token: ${operator_token:0:50}..."
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test invalid login
echo -n "Invalid Login (should fail)... "
invalid_response=$(curl -s -X POST "$BASE_URL/api/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"wrongpassword"}')
if echo "$invalid_response" | grep -q "error"; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "  Response: $invalid_response"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""

echo "3️⃣  Testing Protected Endpoints"
echo "----------------------------"

# Test sync endpoints with authentication
test_endpoint "Sync Pull (authenticated)" "GET" "/api/sync/pull" "" "$admin_token"
test_endpoint "Sync Push (authenticated)" "POST" "/api/sync/push" '{"entries":[]}' "$admin_token"

# Test without authentication (should fail)
echo -n "Sync Pull (no auth - should fail)... "
no_auth_response=$(curl -s -X GET "$BASE_URL/api/sync/pull")
if echo "$no_auth_response" | grep -q "error"; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "  Response: $no_auth_response"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""

echo "4️⃣  Testing Role-Based Access"
echo "----------------------------"

# Test admin endpoints with admin token
test_endpoint "Admin Users (admin)" "GET" "/api/admin/users" "" "$admin_token"

# Test admin endpoints with operator token (should fail)
echo -n "Admin Users (operator - should fail)... "
forbidden_response=$(curl -s -X GET "$BASE_URL/api/admin/users" -H "Authorization: Bearer $operator_token")
if echo "$forbidden_response" | grep -q "error\|Forbidden"; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "  Response: $forbidden_response"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test supervisor endpoints with supervisor token
test_endpoint "Supervisor Entries (supervisor)" "GET" "/api/supervisor/entries" "" "$supervisor_token"

# Test supervisor endpoints with operator token (should fail)
echo -n "Supervisor Entries (operator - should fail)... "
forbidden_response=$(curl -s -X GET "$BASE_URL/api/supervisor/entries" -H "Authorization: Bearer $operator_token")
if echo "$forbidden_response" | grep -q "error\|Forbidden"; then
    echo -e "${GREEN}✓ PASS${NC}"
    echo "  Response: $forbidden_response"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAIL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""

echo "================================"
echo "📊 Test Results"
echo "================================"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo "Total: $((TESTS_PASSED + TESTS_FAILED))"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some tests failed${NC}"
    exit 1
fi
