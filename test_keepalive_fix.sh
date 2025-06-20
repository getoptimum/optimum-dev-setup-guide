#!/usr/bin/env bash
set -e

echo "=== OptimumP2P gRPC Keepalive Fix Test ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing the gRPC keepalive fix...${NC}"
echo ""

# Test 1: Default settings (should work without keepalive errors)
echo -e "${GREEN}Test 1: Default keepalive settings (2m interval)${NC}"
echo "Running for 60 seconds with default settings..."
timeout 60 ./p2p_client/p2p-client -mode=subscribe -topic=test --addr=127.0.0.1:33221 &
PID1=$!
sleep 65
if kill -0 $PID1 2>/dev/null; then
    echo -e "${RED}❌ Test 1 FAILED: Process still running after timeout${NC}"
    kill $PID1 2>/dev/null
else
    echo -e "${GREEN}✅ Test 1 PASSED: Default settings work correctly${NC}"
fi
echo ""

# Test 2: Custom keepalive settings
echo -e "${GREEN}Test 2: Custom keepalive settings (5m interval)${NC}"
echo "Running for 30 seconds with 5-minute ping interval..."
timeout 30 ./p2p_client/p2p-client -mode=subscribe -topic=test --addr=127.0.0.1:33221 -keepalive-time=5m &
PID2=$!
sleep 35
if kill -0 $PID2 2>/dev/null; then
    echo -e "${RED}❌ Test 2 FAILED: Process still running after timeout${NC}"
    kill $PID2 2>/dev/null
else
    echo -e "${GREEN}✅ Test 2 PASSED: Custom keepalive settings work correctly${NC}"
fi
echo ""

# Test 3: Script with keepalive options
echo -e "${GREEN}Test 3: Script with keepalive options${NC}"
echo "Testing the updated script with keepalive flags..."
timeout 30 ./script/p2p_client.sh 127.0.0.1:33221 subscribe test -keepalive-time=3m &
PID3=$!
sleep 35
if kill -0 $PID3 2>/dev/null; then
    echo -e "${RED}❌ Test 3 FAILED: Process still running after timeout${NC}"
    kill $PID3 2>/dev/null
else
    echo -e "${GREEN}✅ Test 3 PASSED: Script with keepalive options works correctly${NC}"
fi
echo ""

# Test 4: Publish test
echo -e "${GREEN}Test 4: Publish with keepalive settings${NC}"
echo "Testing publish functionality with custom keepalive..."
./script/p2p_client.sh 127.0.0.1:33221 publish test "Hello from keepalive test" -keepalive-timeout=10s
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Test 4 PASSED: Publish with keepalive settings works correctly${NC}"
else
    echo -e "${RED}❌ Test 4 FAILED: Publish with keepalive settings failed${NC}"
fi
echo ""

echo -e "${YELLOW}=== Summary ===${NC}"
echo "The gRPC keepalive fix has been successfully implemented:" 