#!/bin/bash

# Flow Control Demo Script
# Demonstrates the flow control features of the gRPC P2P client

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
NODE_ADDR="localhost:33212"
TEST_TOPIC="flow-control-demo"
MESSAGE_COUNT=500

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

print_header() {
    echo
    echo "=========================================="
    echo "$1"
    echo "=========================================="
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if node is running
    if ! timeout 5 bash -c "</dev/tcp/localhost/33212" 2>/dev/null; then
        log_error "Node is not running. Please start the node first."
        log_info "You can start it with: docker-compose up -d"
        exit 1
    fi
    
    # Check if client binary exists
    if [ ! -f "./grpc_p2p_client/p2p_client" ]; then
        log_error "P2P client binary not found. Please build it first."
        log_info "You can build it with: cd grpc_p2p_client && go build -o p2p_client ."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

demo_basic_flow_control() {
    print_header "Demo 1: Basic Flow Control"
    
    log_info "This demo shows basic flow control with conservative settings"
    log_info "Configuration: 50 initial credits, 5 credit increment, 100μs pacing"
    
    # Start subscriber
    log_info "Starting subscriber..."
    timeout 60 ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber_basic.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish with basic flow control
    log_info "Publishing messages with basic flow control..."
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
        -max-concurrent=5 > publisher_basic.log 2>&1
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber
    wait $SUBSCRIBER_PID || true
    
    # Show results
    received_count=$(grep -c "Received message" subscriber_basic.log || echo "0")
    sent_count=$(grep -c "Published message" publisher_basic.log || echo "0")
    
    log_info "Basic Flow Control Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Duration: ${duration}s"
    log_info "  Throughput: $(echo "scale=2; $sent_count / $duration" | bc -l) msg/s"
    
    if [ "$received_count" -ge "$((MESSAGE_COUNT * 95 / 100))" ]; then
        log_success "Basic flow control demo successful!"
    else
        log_warning "Some messages may have been lost"
    fi
}

demo_high_performance_flow_control() {
    print_header "Demo 2: High Performance Flow Control"
    
    log_info "This demo shows aggressive flow control settings for high throughput"
    log_info "Configuration: 100 initial credits, 10 credit increment, 10μs pacing"
    
    # Start subscriber
    log_info "Starting subscriber..."
    timeout 60 ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber_high.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish with high performance settings
    log_info "Publishing messages with high performance flow control..."
    start_time=$(date +%s.%N)
    
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=$MESSAGE_COUNT \
        -flow-control=true \
        -initial-credits=100 \
        -credit-increment=10 \
        -pacing-delay=10us \
        -max-concurrent=20 > publisher_high.log 2>&1
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber
    wait $SUBSCRIBER_PID || true
    
    # Show results
    received_count=$(grep -c "Received message" subscriber_high.log || echo "0")
    sent_count=$(grep -c "Published message" publisher_high.log || echo "0")
    
    log_info "High Performance Flow Control Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Duration: ${duration}s"
    log_info "  Throughput: $(echo "scale=2; $sent_count / $duration" | bc -l) msg/s"
    
    if [ "$received_count" -ge "$((MESSAGE_COUNT * 95 / 100))" ]; then
        log_success "High performance flow control demo successful!"
    else
        log_warning "Some messages may have been lost"
    fi
}

demo_retry_logic() {
    print_header "Demo 3: Retry Logic with Flow Control"
    
    log_info "This demo shows retry logic with conservative flow control"
    log_info "Configuration: 30 initial credits, 3 credit increment, 200μs pacing, 5 retries"
    
    # Start subscriber
    log_info "Starting subscriber..."
    timeout 60 ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=subscribe \
        -topic="$TEST_TOPIC" > subscriber_retry.log 2>&1 &
    SUBSCRIBER_PID=$!
    
    sleep 2
    
    # Publish with retry logic
    log_info "Publishing messages with retry logic..."
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
    duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Wait for subscriber
    wait $SUBSCRIBER_PID || true
    
    # Show results
    received_count=$(grep -c "Received message" subscriber_retry.log || echo "0")
    sent_count=$(grep -c "Published message" publisher_retry.log || echo "0")
    retry_count=$(grep -c "attempt" publisher_retry.log || echo "0")
    
    log_info "Retry Logic Results:"
    log_info "  Messages sent: $sent_count"
    log_info "  Messages received: $received_count"
    log_info "  Retry attempts: $retry_count"
    log_info "  Duration: ${duration}s"
    
    if [ "$received_count" -ge "$((MESSAGE_COUNT * 90 / 100))" ]; then
        log_success "Retry logic demo successful!"
    else
        log_warning "Some messages may have been lost"
    fi
}

demo_comparison() {
    print_header "Demo 4: Flow Control vs No Flow Control Comparison"
    
    log_info "This demo compares performance with and without flow control"
    
    # Test without flow control
    log_info "Testing without flow control..."
    start_time=$(date +%s.%N)
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=200 \
        -flow-control=false > publisher_no_fc.log 2>&1
    end_time=$(date +%s.%N)
    no_fc_duration=$(echo "$end_time - $start_time" | bc -l)
    
    sleep 2
    
    # Test with flow control
    log_info "Testing with flow control..."
    start_time=$(date +%s.%N)
    ./grpc_p2p_client/p2p_client \
        -addr="$NODE_ADDR" \
        -mode=publish \
        -topic="$TEST_TOPIC" \
        -count=200 \
        -flow-control=true \
        -initial-credits=50 \
        -credit-increment=5 \
        -pacing-delay=100us > publisher_with_fc.log 2>&1
    end_time=$(date +%s.%N)
    with_fc_duration=$(echo "$end_time - $start_time" | bc -l)
    
    log_info "Comparison Results:"
    log_info "  No flow control duration: ${no_fc_duration}s"
    log_info "  With flow control duration: ${with_fc_duration}s"
    log_info "  Performance ratio: $(echo "scale=2; $with_fc_duration / $no_fc_duration" | bc -l)x"
    
    performance_ratio=$(echo "scale=2; $with_fc_duration / $no_fc_duration" | bc -l)
    if (( $(echo "$performance_ratio < 2" | bc -l) )); then
        log_success "Flow control provides good performance!"
    else
        log_warning "Flow control may impact performance significantly"
    fi
}

show_usage() {
    echo "Flow Control Demo Script"
    echo
    echo "Usage: $0 [OPTION]"
    echo
    echo "Options:"
    echo "  basic      Run basic flow control demo"
    echo "  high       Run high performance flow control demo"
    echo "  retry      Run retry logic demo"
    echo "  compare    Run comparison demo"
    echo "  all        Run all demos"
    echo "  help       Show this help message"
    echo
    echo "Examples:"
    echo "  $0 basic     # Run basic flow control demo"
    echo "  $0 all       # Run all demos"
}

main() {
    case "${1:-all}" in
        "basic")
            check_prerequisites
            demo_basic_flow_control
            ;;
        "high")
            check_prerequisites
            demo_high_performance_flow_control
            ;;
        "retry")
            check_prerequisites
            demo_retry_logic
            ;;
        "compare")
            check_prerequisites
            demo_comparison
            ;;
        "all")
            check_prerequisites
            demo_basic_flow_control
            demo_high_performance_flow_control
            demo_retry_logic
            demo_comparison
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
    
    echo
    log_success "Demo completed!"
    log_info "Check the log files for detailed results:"
    log_info "  subscriber_*.log - Subscriber output"
    log_info "  publisher_*.log - Publisher output"
}

main "$@" 