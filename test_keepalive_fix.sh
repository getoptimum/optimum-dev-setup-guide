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
TOPIC="keepalive-test-$$"
MESSAGE="hello-keepalive"

PASS=0
FAIL=0

pass() { echo -e "${GREEN}PASS${NC} $1"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}FAIL${NC} $1"; FAIL=$((FAIL+1)); }
warn() { echo -e "${YELLOW}WARN${NC} $1"; }

print_sample_messages() {
    local file=$1
    local total=$2
    local label=$3
    local lines
    lines=$(grep 'Received message' "$file" | sed 's/.*Received message: //')
    if [ "$total" -gt 0 ]; then
        echo -e "${BLUE}$label sample:${NC}"
        echo "$lines" | head -3 | sed 's/^/  [first] /'
        if [ "$total" -gt 6 ]; then
            echo "  ..."
        fi
        echo "$lines" | tail -3 | sed 's/^/  [last]  /'
    fi
}

print_publish_count() {
    local file=$1
    local label=$2
    local count
    count=$(grep -c 'Published' "$file" || true)
    echo -e "${BLUE}$label: Published $count messages${NC}"
}

# Function to test subscribe with timeout and check for errors
test_subscribe() {
    local addr=$1
    local topic=$2
    local keepalive_opts=$3
    local test_name=$4
    echo "[$test_name] Testing subscribe on $addr with $keepalive_opts..."
    timeout 10s $P2P_CLIENT_SCRIPT $addr subscribe $topic $keepalive_opts > /tmp/subscribe_output.log 2>&1 &
    local subscribe_pid=$!
    sleep 2
    if kill -0 $subscribe_pid 2>/dev/null; then
        kill $subscribe_pid 2>/dev/null
        wait $subscribe_pid 2>/dev/null || true
        if grep -q "too_many_pings\|ENHANCE_YOUR_CALM\|Connection closed" /tmp/subscribe_output.log; then
            warn "$test_name - Error(s) found in subscribe log:"
            grep "too_many_pings\|ENHANCE_YOUR_CALM\|Connection closed" /tmp/subscribe_output.log | sed 's/^/  /'
            fail "$test_name - Keepalive error detected"
        else
            pass "$test_name - Subscribe connection established successfully"
        fi
    else
        fail "$test_name - Subscribe process crashed"
    fi
}

test_publish() {
    local addr=$1
    local topic=$2
    local message=$3
    local keepalive_opts=$4
    local test_name=$5
    echo "[$test_name] Testing publish on $addr with $keepalive_opts..."
    $P2P_CLIENT_SCRIPT $addr publish $topic "$message" $keepalive_opts > /tmp/publish_output.log 2>&1
    print_publish_count /tmp/publish_output.log "$test_name"
    if grep -q "Published" /tmp/publish_output.log; then
        pass "$test_name - Publish succeeded"
    else
        fail "$test_name - Publish failed"
    fi
}

# Test 1: Default keepalive settings (2m interval, 20s timeout)
test_subscribe $ADDR1 $TOPIC "" "Default keepalive subscribe"
test_publish $ADDR2 $TOPIC "$MESSAGE" "" "Default keepalive publish"

# Test 2: Custom keepalive interval (5m)
test_subscribe $ADDR1 $TOPIC "-keepalive-interval=5m" "Custom keepalive-interval=5m subscribe"
test_publish $ADDR2 $TOPIC "$MESSAGE" "-keepalive-interval=5m" "Custom keepalive-interval=5m publish"

# Test 3: Custom keepalive timeout (10s)
test_subscribe $ADDR1 $TOPIC "-keepalive-timeout=10s" "Custom keepalive-timeout=10s subscribe"
test_publish $ADDR2 $TOPIC "$MESSAGE" "-keepalive-timeout=10s" "Custom keepalive-timeout=10s publish"

# Test 4: Aggressive keepalive settings (30s interval)
test_subscribe $ADDR1 $TOPIC "-keepalive-interval=30s" "Aggressive keepalive-interval=30s subscribe"

# Test 5: Very aggressive keepalive settings (10s interval)
test_subscribe $ADDR1 $TOPIC "-keepalive-interval=10s" "Very aggressive keepalive-interval=10s subscribe"

# --- Real-World Stress Test ---
STRESS_TOPIC="stress-test-$$"
STRESS_COUNT=1200
STRESS_SLEEP=10ms

# Start subscriber in background
$P2P_CLIENT_SCRIPT $ADDR1 subscribe $STRESS_TOPIC -keepalive-interval=1m > /tmp/stress_sub_output.log 2>&1 &
SUB_PID=$!
sleep 2

# Publish many messages
$P2P_CLIENT_SCRIPT $ADDR2 publish $STRESS_TOPIC "random" -count=$STRESS_COUNT -sleep=$STRESS_SLEEP > /tmp/stress_pub_output.log 2>&1
print_publish_count /tmp/stress_pub_output.log "Stress test"

# Wait for subscriber to catch up
sleep 5
kill $SUB_PID 2>/dev/null || true
wait $SUB_PID 2>/dev/null || true

RECEIVED=$(grep -c "Received message" /tmp/stress_sub_output.log)
print_sample_messages /tmp/stress_sub_output.log $RECEIVED "Stress test"
if [ "$RECEIVED" -ge $((STRESS_COUNT * 95 / 100)) ]; then
    pass "Stress test: Subscriber received $RECEIVED/$STRESS_COUNT messages"
else
    fail "Stress test: Only $RECEIVED/$STRESS_COUNT messages received"
fi

# --- Aggressive keepalive with real message flow ---
AGG_TOPIC="aggressive-keepalive-$$"
AGG_COUNT=300
AGG_SLEEP=20ms
$P2P_CLIENT_SCRIPT $ADDR1 subscribe $AGG_TOPIC -keepalive-interval=10s > /tmp/aggressive_sub_output.log 2>&1 &
AGG_PID=$!
sleep 2
$P2P_CLIENT_SCRIPT $ADDR2 publish $AGG_TOPIC "random" -count=$AGG_COUNT -sleep=$AGG_SLEEP > /tmp/aggressive_pub_output.log 2>&1
print_publish_count /tmp/aggressive_pub_output.log "Aggressive keepalive"
sleep 5
kill $AGG_PID 2>/dev/null || true
wait $AGG_PID 2>/dev/null || true
AGG_RECEIVED=$(grep -c "Received message" /tmp/aggressive_sub_output.log)
print_sample_messages /tmp/aggressive_sub_output.log $AGG_RECEIVED "Aggressive keepalive"
if [ "$AGG_RECEIVED" -ge $((AGG_COUNT * 90 / 100)) ]; then
    pass "Aggressive keepalive: Subscriber received $AGG_RECEIVED/$AGG_COUNT messages"
else
    fail "Aggressive keepalive: Only $AGG_RECEIVED/$AGG_COUNT messages received"
fi

# Cleanup
delete_temp() {
    rm -f /tmp/subscribe_output.log /tmp/publish_output.log /tmp/stress_sub_output.log /tmp/stress_pub_output.log /tmp/aggressive_sub_output.log /tmp/aggressive_pub_output.log
}
delete_temp

echo -e "\n${YELLOW}Summary Table:${NC}"
echo -e "${GREEN}Keepalive subscribe/publish tests:${NC} 5 subscribe, 5 publish"
echo -e "${GREEN}Stress test:${NC} $STRESS_COUNT published, $RECEIVED received"
echo -e "${GREEN}Aggressive keepalive:${NC} $AGG_COUNT published, $AGG_RECEIVED received"
echo -e "${YELLOW}Total: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"
if [ "$FAIL" -eq 0 ]; then
    echo -e "${GREEN}All keepalive and stress tests passed!${NC}"
else
    echo -e "${RED}Some tests failed. See above for details.${NC}"
    exit 1
fi 