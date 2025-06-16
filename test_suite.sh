#!/bin/bash

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0

echo -e "${YELLOW}==============================="
echo -e " OptimumP2P API Test Suite "
echo -e "===============================${NC}\n"

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

# 1. Subscribe (valid)
echo -e "${YELLOW}Test: Subscribe (valid)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/subscribe -H "Content-Type: application/json" -d '{"client_id": "test-client", "topic": "example-topic", "threshold": 0.7}')
test_result "$resp" '"status":"subscribed"' "Subscribe (valid)"
echo

# 2. Subscribe (empty fields)
echo -e "${YELLOW}Test: Subscribe (empty fields)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/subscribe -H "Content-Type: application/json" -d '{"client_id": "", "topic": ""}')
test_result "$resp" '"client":"","status":"subscribed"' "Subscribe (empty fields)"
echo

# 3. Publish (valid)
echo -e "${YELLOW}Test: Publish (valid)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/publish -H "Content-Type: application/json" -d '{"topic": "example-topic", "message": "Hello, world!"}')
test_result "$resp" '"status":"published"' "Publish (valid)"
echo

# 4. Publish (non-existent topic)
echo -e "${YELLOW}Test: Publish (non-existent topic)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/publish -H "Content-Type: application/json" -d '{"topic": "non-existent-topic", "message": "Test"}')
test_result "$resp" 'topic not assigned' "Publish (non-existent topic)"
echo

# 5. Health
echo -e "${YELLOW}Test: Node health${NC}"
resp=$(curl -s http://localhost:9091/api/v1/health)
test_result "$resp" '"status":"ok"' "Node health"
echo

# 6. Node state
echo -e "${YELLOW}Test: Node state${NC}"
resp=$(curl -s http://localhost:9091/api/v1/node-state)
test_result "$resp" '"pub_key"' "Node state"
echo

# 7. Node version
echo -e "${YELLOW}Test: Node version${NC}"
resp=$(curl -s http://localhost:9091/api/v1/version)
test_result "$resp" '"commit_hash"' "Node version"
echo

# 8. Invalid JSON
echo -e "${YELLOW}Test: Subscribe (invalid JSON)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/subscribe -H "Content-Type: application/json" -d 'invalid json')
test_result "$resp" 'invalid JSON' "Subscribe (invalid JSON)"
echo

# 9. Rapid publish
echo -e "${YELLOW}Test: Rapid publish (5x)${NC}"
ALL_PASS=1
for i in {1..5}; do
  resp=$(curl -s -X POST http://localhost:8081/api/publish -H "Content-Type: application/json" -d '{"topic": "example-topic", "message": "Rapid test '$i'"}')
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

# 10. WebSocket basic connection (wscat)
echo -e "${YELLOW}Test: WebSocket connection (wscat)${NC}"
if command -v wscat >/dev/null 2>&1; then
  output=$(echo | gtimeout 3 wscat -c "ws://localhost:8081/api/ws?client_id=test-client" 2>&1)
  if echo "$output" | grep -qE "(Connected|>)"; then
    echo -e "${GREEN}[PASS]${NC} WebSocket (wscat) basic connection"
    PASS=$((PASS+1))
  else
    echo -e "${RED}[FAIL]${NC} WebSocket (wscat) basic connection"
    echo -e "  ${YELLOW}Output:${NC} $output"
    FAIL=$((FAIL+1))
  fi
else
  echo -e "${YELLOW}[SKIP]${NC} wscat not installed, skipping WebSocket test"
fi
echo

# 11. Subscribe with empty client_id, valid topic
echo -e "${YELLOW}Test: Subscribe (empty client_id, valid topic)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/subscribe -H "Content-Type: application/json" -d '{"client_id": "", "topic": "example-topic"}')
test_result "$resp" '"client":"","status":"subscribed"' "Subscribe (empty client_id, valid topic)"
echo

# 12. Subscribe with valid client_id, empty topic
echo -e "${YELLOW}Test: Subscribe (valid client_id, empty topic)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/subscribe -H "Content-Type: application/json" -d '{"client_id": "test-client", "topic": ""}')
test_result "$resp" '"client":"test-client","status":"subscribed"' "Subscribe (valid client_id, empty topic)"
echo

# 13. Publish with empty message
echo -e "${YELLOW}Test: Publish (empty message)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/publish -H "Content-Type: application/json" -d '{"topic": "example-topic", "message": ""}')
test_result "$resp" '"status":"published"' "Publish (empty message)"
echo

# 14. Publish with missing topic
echo -e "${YELLOW}Test: Publish (missing topic)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/publish -H "Content-Type: application/json" -d '{"message": "Hello"}')
test_result "$resp" 'topic not assigned' "Publish (missing topic)"
echo

# 15. Publish with missing message
echo -e "${YELLOW}Test: Publish (missing message)${NC}"
resp=$(curl -s -X POST http://localhost:8081/api/publish -H "Content-Type: application/json" -d '{"topic": "example-topic"}')
test_result "$resp" '"status":"published"' "Publish (missing message)"
echo

echo -e "${YELLOW}==============================="
echo -e " Test suite complete: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}."
echo -e "===============================${NC}"
if [[ $FAIL -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
else
  echo -e "${RED}Some tests failed. Please review the output above.${NC}"
fi 