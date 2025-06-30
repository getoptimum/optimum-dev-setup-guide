#!/bin/bash

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "gRPC Flow Control Fix Test"
echo "=========================="

# Configuration
P2P_CLIENT_DIR="./grpc_p2p_client"
TEST_TOPIC="flow-control-test"
TEST_MESSAGE="AAAAAAAAAAAAAAAAAAAAAA"
TEST_COUNT=1000
LOG_FILE="/tmp/flow_control_test.log"

# Build the client
echo "Building P2P client..."
cd "$P2P_CLIENT_DIR"
if ! go build -o p2p-client ./p2p_client.go; then
    echo -e "${RED}Failed to build P2P client${NC}"
    exit 1
fi
cd ..

# Check if containers are running
echo "Checking if OptimumP2P containers are running..."
if ! docker ps | grep -q "optimum"; then
    echo -e "${RED}Error: OptimumP2P containers are not running${NC}"
    echo "Please start the containers with: docker-compose up -d"
    exit 1
fi

# Test flow control
echo "Testing with $TEST_COUNT messages..."

cd "$P2P_CLIENT_DIR"
./p2p-client -addr=localhost:33221 -mode=subscribe -topic="$TEST_TOPIC" -buffer-size=2000 -workers=8 > "$LOG_FILE" 2>&1 &
SUB_PID=$!
sleep 3

./p2p-client -addr=localhost:33221 -mode=publish -topic="$TEST_TOPIC" -msg="$TEST_MESSAGE" -count="$TEST_COUNT" &
PUB_PID=$!

# Monitor for 2 minutes
start_time=$(date +%s)
while kill -0 $SUB_PID 2>/dev/null; do
    elapsed=$(( $(date +%s) - start_time ))
    if [ $elapsed -ge 120 ]; then break; fi
    
    count=$(grep -c "Received message:" "$LOG_FILE" 2>/dev/null || echo "0")
    echo "Progress: $count/$TEST_COUNT messages (${elapsed}s)"
    
    if [ "$count" -ge "$TEST_COUNT" ]; then
        echo -e "${GREEN}SUCCESS: All messages received${NC}"
        kill $SUB_PID $PUB_PID 2>/dev/null
        exit 0
    fi
    
    if grep -q "fatal\|panic" "$LOG_FILE"; then
        echo -e "${RED}FAILED: Fatal error${NC}"
        kill $SUB_PID $PUB_PID 2>/dev/null
        exit 1
    fi
    
    sleep 5
done

# Final check
final_count=$(grep -c "Received message:" "$LOG_FILE" 2>/dev/null || echo "0")
kill $SUB_PID $PUB_PID 2>/dev/null

echo "Final: $final_count/$TEST_COUNT messages"

if [ "$final_count" -ge 800 ]; then
    echo -e "${GREEN}PASSED: Flow control fix working${NC}"
    exit 0
else
    echo -e "${RED}FAILED: Flow control fix not working${NC}"
    exit 1
fi 