#!/bin/bash

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
PROXY_URL="http://localhost:8081"
NODE_URL="http://localhost:9091"
CLIENT_ID="test-client-$(date +%s)"
TOPIC="example-topic"

PASS=0
FAIL=0

echo -e "${YELLOW}==============================="
echo -e " OptimumP2P API Test Suite "
echo -e "===============================${NC}"
echo -e "${BLUE}Client ID: ${CLIENT_ID}${NC}\n"

# Check required dependencies
check_dependencies() {
  local missing_deps=()
  
  if ! command -v curl >/dev/null 2>&1; then
    missing_deps+=("curl")
  fi
  
  if ! command -v bash >/dev/null 2>&1; then
    missing_deps+=("bash")
  fi
  
  if [[ ${#missing_deps[@]} -gt 0 ]]; then
    echo -e "${RED}Error: Missing required dependencies:${NC} ${missing_deps[*]}"
    echo -e "${YELLOW}Please install the missing dependencies and try again.${NC}"
    exit 1
  fi
}

# Helper to check result
test_result() {
  local actual="$1"
  local expected="$2"
  local name="$3"
  if [[ "$actual" == *"$expected"* ]]; then
    echo -e "${GREEN}[PASS]${NC} $name"
    PASS=$((PASS+1))
  else
    echo -e "${RED}[FAIL]${NC} $name"
    echo -e "  ${YELLOW}Expected:${NC} $expected"
    echo -e "  ${YELLOW}Actual:  ${NC} $actual"
    FAIL=$((FAIL+1))
  fi
}

# API helper functions
api_subscribe() {
  local client_id="$1"
  local topic="$2"
  local threshold="$3"
  curl -s -X POST "$PROXY_URL/api/subscribe" \
    -H "Content-Type: application/json" \
    -d "{\"client_id\": \"$client_id\", \"topic\": \"$topic\", \"threshold\": $threshold}"
}

api_publish() {
  local client_id="$1"
  local topic="$2"
  local message="$3"
  curl -s -X POST "$PROXY_URL/api/publish" \
    -H "Content-Type: application/json" \
    -d "{\"client_id\": \"$client_id\", \"topic\": \"$topic\", \"message\": \"$message\"}"
}

api_health() {
  curl -s "$NODE_URL/api/v1/health"
}

api_node_state() {
  curl -s "$NODE_URL/api/v1/node-state"
}

api_version() {
  curl -s "$NODE_URL/api/v1/version"
}

# Run dependency check
check_dependencies

# Test 1: Subscribe (valid)
echo -e "${YELLOW}Test: Subscribe (valid)${NC}"
resp=$(api_subscribe "$CLIENT_ID" "$TOPIC" 0.7)
test_result "$resp" '"status":"subscribed"' "Subscribe (valid)"
echo

# Test 2: Subscribe (empty topic)
echo -e "${YELLOW}Test: Subscribe (empty topic)${NC}"
resp=$(api_subscribe "$CLIENT_ID" "" 0.7)
test_result "$resp" 'topic is missing' "Subscribe (empty topic)"
echo

# Test 3: Publish (valid)
echo -e "${YELLOW}Test: Publish (valid)${NC}"
resp=$(api_publish "$CLIENT_ID" "$TOPIC" "Hello, world!")
test_result "$resp" '"status":"published"' "Publish (valid)"
echo

# Test 4: Publish (non-existent topic)
echo -e "${YELLOW}Test: Publish (non-existent topic)${NC}"
resp=$(api_publish "$CLIENT_ID" "non-existent-topic" "Test")
test_result "$resp" 'topic not assigned' "Publish (non-existent topic)"
echo

# Test 5: Health check
echo -e "${YELLOW}Test: Node health${NC}"
resp=$(api_health)
test_result "$resp" '"status":"ok"' "Node health"
echo

# Test 6: Node state
echo -e "${YELLOW}Test: Node state${NC}"
resp=$(api_node_state)
test_result "$resp" '"pub_key"' "Node state"
echo

# Test 7: Node version
echo -e "${YELLOW}Test: Node version${NC}"
resp=$(api_version)
test_result "$resp" '"commit_hash"' "Node version"
echo

# Test 8: Invalid JSON
echo -e "${YELLOW}Test: Subscribe (invalid JSON)${NC}"
resp=$(curl -s -X POST "$PROXY_URL/api/subscribe" -H "Content-Type: application/json" -d 'invalid json')
test_result "$resp" 'invalid JSON' "Subscribe (invalid JSON)"
echo

# Test 9: Rapid publish
echo -e "${YELLOW}Test: Rapid publish (5x)${NC}"
ALL_PASS=1
for i in {1..5}; do
  resp=$(api_publish "$CLIENT_ID" "$TOPIC" "Rapid test $i")
  if [[ "$resp" != *'"status":"published"'* ]]; then
    ALL_PASS=0
    echo -e "  ${RED}[FAIL]${NC} Rapid publish $i: $resp"
  fi
done
if [[ $ALL_PASS -eq 1 ]]; then
  echo -e "${GREEN}[PASS]${NC} Rapid publish (5x)"
  PASS=$((PASS+1))
else
  echo -e "${RED}[FAIL]${NC} Rapid publish (5x)"
  FAIL=$((FAIL+1))
fi
echo

# Test 10: WebSocket connection
echo -e "${YELLOW}Test: WebSocket connection${NC}"
if command -v wscat >/dev/null 2>&1; then
  # Detect timeout command (gtimeout on macOS, timeout on Linux)
  if command -v gtimeout >/dev/null 2>&1; then
    TIMEOUT_CMD="gtimeout"
  elif command -v timeout >/dev/null 2>&1; then
    TIMEOUT_CMD="timeout"
  else
    echo -e "${YELLOW}[SKIP]${NC} No timeout command available, skipping WebSocket test"
    echo
  fi
  
  if [[ -n "$TIMEOUT_CMD" ]]; then
    output=$($TIMEOUT_CMD 5 wscat -c "ws://$PROXY_URL/api/ws?client_id=$CLIENT_ID" 2>&1)
    if echo "$output" | grep -q "Connected" || echo "$output" | grep -q ">" || echo "$output" | grep -q "connected"; then
      echo -e "${GREEN}[PASS]${NC} WebSocket connection"
      PASS=$((PASS+1))
    else
      echo -e "${RED}[FAIL]${NC} WebSocket connection"
      echo -e "  ${YELLOW}Output:${NC} $output"
      FAIL=$((FAIL+1))
    fi
  fi
else
  echo -e "${YELLOW}[SKIP]${NC} wscat not installed, skipping WebSocket test"
fi
echo

# Test 11: Validation tests
echo -e "${YELLOW}Test: Subscribe validation (empty client_id)${NC}"
resp=$(api_subscribe "" "$TOPIC" 0.7)
test_result "$resp" 'client_id is missing' "Subscribe validation (empty client_id)"
echo

echo -e "${YELLOW}Test: Subscribe validation (empty topic)${NC}"
resp=$(api_subscribe "$CLIENT_ID" "" 0.7)
test_result "$resp" 'topic is missing' "Subscribe validation (empty topic)"
echo

echo -e "${YELLOW}Test: Publish validation (empty message)${NC}"
resp=$(api_publish "$CLIENT_ID" "$TOPIC" "")
test_result "$resp" 'message is missing' "Publish validation (empty message)"
echo

echo -e "${YELLOW}Test: Publish validation (missing topic)${NC}"
resp=$(curl -s -X POST "$PROXY_URL/api/publish" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\": \"$CLIENT_ID\", \"message\": \"Hello\"}")
test_result "$resp" 'topic is missing' "Publish validation (missing topic)"
echo

echo -e "${YELLOW}Test: Publish validation (missing message)${NC}"
resp=$(curl -s -X POST "$PROXY_URL/api/publish" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\": \"$CLIENT_ID\", \"topic\": \"$TOPIC\"}")
test_result "$resp" 'message is missing' "Publish validation (missing message)"
echo

# Summary
echo -e "${YELLOW}==============================="
echo -e " Test suite complete: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}."
echo -e "===============================${NC}"
if [[ $FAIL -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
else
  echo -e "${RED}Some tests failed. Please review the output above.${NC}"
fi 