#!/bin/bash

# Flow Control Test Suite for gRPC P2P Client
# Tests various flow control mechanisms and configurations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NODE_ADDR="localhost:33221"
TEST_TOPIC="flow-control-test"
MESSAGE_COUNT=1000
HIGH_LOAD_COUNT=10000
SUBSCRIPTION_TIMEOUT=30

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_test_header() {
    echo
    echo "=========================================="
    echo "Test: $1"
    echo "=========================================="
}

print_test_result() {
    local test_name="$1"
    local result="$2"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$result" = "PASS" ]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_success "$test_name passed"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        log_error "$test_name failed"
    fi
}

wait_for_node() {
    log_info "Waiting for node to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if timeout 5 bash -c "</dev/tcp/localhost/33221" 2>/dev/null; then
            log_success "Node is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    log_error "Node not ready after $max_attempts attempts"
    return 1
}

cleanup() {
    log_info "Cleaning up test processes..."
    pkill -f "p2p_client" || true
    pkill -f "gateway_client" || true
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Test 1: Basic Flow Control Test
test_basic_flow_control() {
    print_test_header "Basic Flow Control Test"
    
    # Start subscriber in background
    log_info "Starting subscriber..."
    timeout $SUBSCRIPTION_TIMEOUT ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish messages with flow control enabled
    log_info "Publishing messages with flow control..."
    start_time=$(date +%s.%N)
    
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=$MESSAGE_COUNT \
        -flow-control=true \
        -initial-credits=50 \
        -credit-increment=5 \
        -pacing-delay=100us \
        -max-concurrent=5 > publisher.log 2>&1
    
    end_time=$(date +%s.%N)
    publish_duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber to finish
    wait $SUBSCRIBER_PID || true
    
    # Check results
    received_count=$(grep -c "Received message" subscriber.log || echo "0")
    sent_count=$(grep -c "Published message" publisher.log || echo "0")
    
    log_info "Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Publish duration: ${publish_duration}s"
    
    if [ "$received_count" -ge "$((MESSAGE_COUNT * 95 / 100))" ]; then
        print_test_result "Basic Flow Control" "PASS"
    else
        print_test_result "Basic Flow Control" "FAIL"
    fi
}

# Test 2: High Load Test
test_high_load() {
    print_test_header "High Load Test"
    
    # Start subscriber in background
    log_info "Starting subscriber for high load test..."
    timeout $SUBSCRIPTION_TIMEOUT ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber_high.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish high load with flow control
    log_info "Publishing high load messages..."
    start_time=$(date +%s.%N)
    
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=$HIGH_LOAD_COUNT \
        -flow-control=true \
        -initial-credits=100 \
        -credit-increment=10 \
        -pacing-delay=10us \
        -max-concurrent=20 \
        -max-retries=5 > publisher_high.log 2>&1
    
    end_time=$(date +%s.%N)
    publish_duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber to finish
    wait $SUBSCRIBER_PID || true
    
    # Check results
    received_count=$(grep -c "Received message" subscriber_high.log || echo "0")
    sent_count=$(grep -c "Published message" publisher_high.log || echo "0")
    dropped_count=$(grep -c "dropped after" publisher_high.log || echo "0")
    
    log_info "High Load Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Messages dropped: $dropped_count"
    log_info "  Publish duration: ${publish_duration}s"
    log_info "  Throughput: $(echo "scale=2; $sent_count / $publish_duration" | bc -l) msg/s"
    
    # Success criteria: less than 5% message loss
    loss_percentage=$(echo "scale=2; ($sent_count - $received_count) * 100 / $sent_count" | bc -l)
    if (( $(echo "$loss_percentage < 5" | bc -l) )); then
        print_test_result "High Load Test" "PASS"
    else
        print_test_result "High Load Test" "FAIL"
    fi
}

# Test 3: Flow Control Disabled Test
test_flow_control_disabled() {
    print_test_header "Flow Control Disabled Test"
    
    # Start subscriber in background
    log_info "Starting subscriber..."
    timeout $SUBSCRIPTION_TIMEOUT ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber_disabled.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish messages without flow control
    log_info "Publishing messages without flow control..."
    start_time=$(date +%s.%N)
    
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=$MESSAGE_COUNT \
        -flow-control=false > publisher_disabled.log 2>&1
    
    end_time=$(date +%s.%N)
    publish_duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber to finish
    wait $SUBSCRIBER_PID || true
    
    # Check results
    received_count=$(grep -c "Received message" subscriber_disabled.log || echo "0")
    sent_count=$(grep -c "Published" publisher_disabled.log || echo "0")
    
    log_info "Flow Control Disabled Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Publish duration: ${publish_duration}s"
    
    if [ "$received_count" -ge "$((MESSAGE_COUNT * 90 / 100))" ]; then
        print_test_result "Flow Control Disabled" "PASS"
    else
        print_test_result "Flow Control Disabled" "FAIL"
    fi
}

# Test 4: Adaptive Pacing Test
test_adaptive_pacing() {
    print_test_header "Adaptive Pacing Test"
    
    # Start subscriber in background
    log_info "Starting subscriber..."
    timeout $SUBSCRIPTION_TIMEOUT ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber_adaptive.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish with adaptive pacing
    log_info "Publishing with adaptive pacing..."
    start_time=$(date +%s.%N)
    
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=$MESSAGE_COUNT \
        -flow-control=true \
        -adaptive-pacing=true \
        -initial-credits=75 \
        -credit-increment=8 \
        -pacing-delay=50us \
        -max-concurrent=15 > publisher_adaptive.log 2>&1
    
    end_time=$(date +%s.%N)
    publish_duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber to finish
    wait $SUBSCRIBER_PID || true
    
    # Check results
    received_count=$(grep -c "Received message" subscriber_adaptive.log || echo "0")
    sent_count=$(grep -c "Published message" publisher_adaptive.log || echo "0")
    
    log_info "Adaptive Pacing Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Publish duration: ${publish_duration}s"
    
    if [ "$received_count" -ge "$((MESSAGE_COUNT * 95 / 100))" ]; then
        print_test_result "Adaptive Pacing" "PASS"
    else
        print_test_result "Adaptive Pacing" "FAIL"
    fi
}

# Test 5: Retry Logic Test
test_retry_logic() {
    print_test_header "Retry Logic Test"
    
    # Start subscriber in background
    log_info "Starting subscriber..."
    timeout $SUBSCRIPTION_TIMEOUT ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber_retry.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish with aggressive retry settings
    log_info "Publishing with retry logic..."
    start_time=$(date +%s.%N)
    
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=$MESSAGE_COUNT \
        -flow-control=true \
        -max-retries=5 \
        -retry-delay=50ms \
        -initial-credits=30 \
        -credit-increment=3 \
        -pacing-delay=200us > publisher_retry.log 2>&1
    
    end_time=$(date +%s.%N)
    publish_duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber to finish
    wait $SUBSCRIBER_PID || true
    
    # Check results
    received_count=$(grep -c "Received message" subscriber_retry.log || echo "0")
    sent_count=$(grep -c "Published message" publisher_retry.log || echo "0")
    retry_count=$(grep -c "attempt" publisher_retry.log || echo "0")
    
    log_info "Retry Logic Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Retry attempts: $retry_count"
    log_info "  Publish duration: ${publish_duration}s"
    
    if [ "$received_count" -ge "$((MESSAGE_COUNT * 90 / 100))" ]; then
        print_test_result "Retry Logic" "PASS"
    else
        print_test_result "Retry Logic" "FAIL"
    fi
}

# Test 6: Performance Comparison
test_performance_comparison() {
    print_test_header "Performance Comparison Test"
    
    log_info "Testing performance with different configurations..."
    
    # Test 1: No flow control
    log_info "Test 1: No flow control"
    start_time=$(date +%s.%N)
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=1000 \
        -flow-control=false > perf_no_fc.log 2>&1
    end_time=$(date +%s.%N)
    no_fc_duration=$(echo "$end_time - $start_time" | bc -l)
    
    sleep 2
    
    # Test 2: With flow control
    log_info "Test 2: With flow control"
    start_time=$(date +%s.%N)
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=1000 \
        -flow-control=true \
        -initial-credits=100 \
        -credit-increment=10 \
        -pacing-delay=10us > perf_with_fc.log 2>&1
    end_time=$(date +%s.%N)
    with_fc_duration=$(echo "$end_time - $start_time" | bc -l)
    
    log_info "Performance Results:"
    log_info "  No flow control duration: ${no_fc_duration}s"
    log_info "  With flow control duration: ${with_fc_duration}s"
    log_info "  Performance ratio: $(echo "scale=2; $with_fc_duration / $no_fc_duration" | bc -l)x"
    
    # Success if flow control doesn't degrade performance by more than 50%
    performance_ratio=$(echo "scale=2; $with_fc_duration / $no_fc_duration" | bc -l)
    if (( $(echo "$performance_ratio < 1.5" | bc -l) )); then
        print_test_result "Performance Comparison" "PASS"
    else
        print_test_result "Performance Comparison" "FAIL"
    fi
}

# Main test execution
main() {
    echo "=========================================="
    echo "Flow Control Test Suite"
    echo "=========================================="
    echo "Node Address: $NODE_ADDR"
    echo "Test Topic: $TEST_TOPIC"
    echo "Message Count: $MESSAGE_COUNT"
    echo "High Load Count: $HIGH_LOAD_COUNT"
    echo "=========================================="
    
    # Check if node is running
    if ! wait_for_node; then
        log_error "Node is not running. Please start the node first."
        exit 1
    fi
    
    # Check if client binary exists
    if [ ! -f "./grpc_p2p_client/p2p_client" ]; then
        log_error "P2P client binary not found. Please build it first."
        exit 1
    fi
    
    # Run tests
    test_basic_flow_control
    test_high_load
    test_flow_control_disabled
    test_adaptive_pacing
    test_retry_logic
    test_performance_comparison
    
    # Print summary
    echo
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    echo "Total Tests: $TOTAL_TESTS"
    echo "Passed: $PASSED_TESTS"
    echo "Failed: $FAILED_TESTS"
    echo "Success Rate: $((PASSED_TESTS * 100 / TOTAL_TESTS))%"
    echo "=========================================="
    
    if [ $FAILED_TESTS -eq 0 ]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "$FAILED_TESTS test(s) failed!"
        exit 1
    fi
}

# Run main function
main "$@" 