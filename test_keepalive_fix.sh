#!/usr/bin/env bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

P2P_CLIENT_SCRIPT="./script/p2p_client.sh"
ADDR1="localhost:33221"
ADDR2="localhost:33222"
TOPIC="keepalive-test"
MESSAGE="hello-keepalive"

pass() { echo -e "${GREEN}PASS${NC} $1"; }
fail() { echo -e "${RED}FAIL${NC} $1"; exit 1; }
warn() { echo -e "${YELLOW}WARN${NC} $1"; }

# Function to test subscribe with timeout
test_subscribe() {
    local addr=$1
    local topic=$2
    local keepalive_opts=$3
    local test_name=$4
    
    echo "[$test_name] Testing subscribe on $addr with $keepalive_opts..."
    
    # Start subscribe in background with timeout
    timeout 10s $P2P_CLIENT_SCRIPT $addr subscribe $topic $keepalive_opts > /tmp/subscribe_output.log 2>&1 &
    local subscribe_pid=$!
    
    # Wait a bit for connection to establish
    sleep 2
    
    # Check if process is still running (not crashed)
    if kill -0 $subscribe_pid 2>/dev/null; then
        # Kill the background process
        kill $subscribe_pid 2>/dev/null
        wait $subscribe_pid 2>/dev/null || true
        
        # Check for error messages
        if grep -q "too_many_pings\|ENHANCE_YOUR_CALM\|Connection closed" /tmp/subscribe_output.log; then
            fail "$test_name - Keepalive error detected"
        else
            pass "$test_name - Subscribe connection established successfully"
        fi
    else
        fail "$test_name - Subscribe process crashed"
    fi
}

# Function to test publish
test_publish() {
    local addr=$1
    local topic=$2
    local message=$3
    local keepalive_opts=$4
    local test_name=$5
    
    echo "[$test_name] Testing publish on $addr with $keepalive_opts..."
    
    if $P2P_CLIENT_SCRIPT $addr publish $topic "$message" $keepalive_opts 2>&1 | grep -q "Published"; then
        pass "$test_name - Publish succeeded"
    else
        fail "$test_name - Publish failed"
    fi
}

# Test 1: Default keepalive settings (2m interval, 20s timeout)
test_subscribe $ADDR1 $TOPIC "" "Test 1: Default keepalive subscribe"
test_publish $ADDR2 $TOPIC "$MESSAGE" "" "Test 2: Default keepalive publish"

# Test 3: Custom keepalive interval (5m)
test_subscribe $ADDR1 $TOPIC "-keepalive-internal=5m" "Test 3: Custom keepalive-internal=5m subscribe"
test_publish $ADDR2 $TOPIC "$MESSAGE" "-keepalive-internal=5m" "Test 4: Custom keepalive-internal=5m publish"

# Test 5: Custom keepalive timeout (10s)
test_subscribe $ADDR1 $TOPIC "-keepalive-timeout=10s" "Test 5: Custom keepalive-timeout=10s subscribe"
test_publish $ADDR2 $TOPIC "$MESSAGE" "-keepalive-timeout=10s" "Test 6: Custom keepalive-timeout=10s publish"

# Test 7: Aggressive keepalive settings (reproduce original issue)
echo "[Test 7] Testing aggressive keepalive settings (30s interval) - should NOT crash..."
test_subscribe $ADDR1 $TOPIC "-keepalive-internal=30s" "Test 7: Aggressive keepalive-internal=30s subscribe"

# Test 8: Very aggressive keepalive settings (10s interval) - this should show the fix prevents crashes
echo "[Test 8] Testing very aggressive keepalive settings (10s interval) - should NOT crash with fix..."
test_subscribe $ADDR1 $TOPIC "-keepalive-internal=10s" "Test 8: Very aggressive keepalive-internal=10s subscribe"

# --- Aggressive keepalive real-world test ---
# Usage: TEST_DURATION=600 ./test_keepalive_fix.sh

TEST_DURATION="${TEST_DURATION:-600}" # default 600 seconds (10 minutes)

# Aggressive keepalive test (reproduce original issue)
echo "[Aggressive Keepalive Test] Running subscriber with -keepalive-internal=10s for $TEST_DURATION seconds..."

$P2P_CLIENT_SCRIPT $ADDR1 subscribe $TOPIC -keepalive-internal=10s > /tmp/subscribe_long_output.log 2>&1 &
sub_pid=$!

# Wait for the test duration
sleep $TEST_DURATION

# After waiting, kill the subscriber
kill $sub_pid 2>/dev/null
wait $sub_pid 2>/dev/null || true

# Check for the error in the output
if grep -q "too_many_pings\|ENHANCE_YOUR_CALM\|Connection closed" /tmp/subscribe_long_output.log; then
    fail "Aggressive keepalive test: Subscriber crashed with keepalive error (issue reproduced)"
else
    pass "Aggressive keepalive test: Subscriber survived $TEST_DURATION seconds (fix works)"
fi

rm -f /tmp/subscribe_long_output.log

echo -e "${GREEN}All keepalive tests passed!${NC}"
echo -e "${GREEN}The fix successfully prevents 'too_many_pings' errors with aggressive keepalive settings.${NC}"

# Cleanup
rm -f /tmp/subscribe_output.log 