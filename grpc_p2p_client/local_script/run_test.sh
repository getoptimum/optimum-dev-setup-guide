#!/bin/bash

# ==============================================================================
# P2P Load Test Automation Script
#
# This script automates:
# 1. Cleaning up old data
# 2. Starting the subscriber in the background
# 3. Running the publisher in the foreground
# 4. Terminating the subscriber
# 5. Running the analysis script
# 6. Archiving all results
# ==============================================================================

# Default Configuration
BATCH=1
SLEEP="500ms"
DATASIZE=80000
COUNT=1200
TOPIC="mybnbtest"
IPFILE="ip.p2pnode.bnb.tsv"
START_IDX=0
END_IDX=1
RUNS=1
WAIT_TIME="5m"

# Usage helper
usage() {
    echo "Usage: ./run_test.sh [options]"
    echo "Options:"
    echo "  -batch <num>      (default: $BATCH)"
    echo "  -sleep <duration> (default: $SLEEP)"
    echo "  -datasize <bytes> (default: $DATASIZE)"
    echo "  -count <num>      (default: $COUNT)"
    echo "  -topic <name>     (default: $TOPIC)"
    echo "  -ipfile <path>    (default: $IPFILE)"
    echo "  -start-index <n>  (default: $START_IDX)"
    echo "  -end-index <n>    (default: $END_IDX)"
    echo "  -runs <num>       (default: $RUNS)"
    echo "  -wait <duration>  (default: $WAIT_TIME)"
    exit 1
}

# Parse Arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -batch) BATCH="$2"; shift 2 ;;
        -sleep) SLEEP="$2"; shift 2 ;;
        -datasize) DATASIZE="$2"; shift 2 ;;
        -count) COUNT="$2"; shift 2 ;;
        -topic) TOPIC="$2"; shift 2 ;;
        -ipfile) IPFILE="$2"; shift 2 ;;
        -start-index) START_IDX="$2"; shift 2 ;;
        -end-index) END_IDX="$2"; shift 2 ;;
        -runs) RUNS="$2"; shift 2 ;;
        -wait) WAIT_TIME="$2"; shift 2 ;;
        -h|--help) usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

TRACE_FILE="incoming-trace.tsv"
DATA_FILE="incoming-data.tsv"
SUB_LOG="subscriber.log"
SUMMARY_FILE="test_summary.log"

# Clear summary file at the start
echo "P2P Load Test Summary - $(date)" > "$SUMMARY_FILE"
echo "Configuration: Batch=$BATCH, Sleep=$SLEEP, DataSize=$DATASIZE, Count=$COUNT, Runs=$RUNS, Wait=$WAIT_TIME" >> "$SUMMARY_FILE"
echo "---------------------------------------------------------" >> "$SUMMARY_FILE"

for (( r=1; r<=RUNS; r++ )); do
    echo "========================================================="
    echo ">>> Starting Run $r of $RUNS"
    echo "========================================================="

    # --- Cleanup and Preparation ---
    echo "[$(date '+%H:%M:%S')] Cleaning up old test data..."
    rm -f "$TRACE_FILE" "$DATA_FILE" "$SUB_LOG"

    # --- Start Subscriber ---
    echo "[$(date '+%H:%M:%S')] Starting subscriber (Background)..."
    ./p2p-multi-subscribe \
        -topic "$TOPIC" \
        -ipfile "$IPFILE" \
        -output-trace "$TRACE_FILE" \
        -output-data "$DATA_FILE" > "$SUB_LOG" 2>&1 &

    SUB_PID=$!

    # Register cleanup function for interruption
    trap 'echo "Terminating..."; kill $SUB_PID 2>/dev/null; wait $SUB_PID 2>/dev/null; exit 1' INT TERM

    # Wait for subscriber to establish connections
    echo "[$(date '+%H:%M:%S')] Waiting 5s for subscriber to settle..."
    sleep 5

    # --- Start Publisher ---
    echo "[$(date '+%H:%M:%S')] Starting publisher (Foreground)..."
    ./p2p-multi-publish \
        -batch "$BATCH" \
        -sleep "$SLEEP" \
        -datasize "$DATASIZE" \
        -count "$COUNT" \
        -ipfile "$IPFILE" \
        -start-index "$START_IDX" \
        -end-index "$END_IDX" \
        -topic "$TOPIC"

    # --- Post-Test Propagation ---
    echo "[$(date '+%H:%M:%S')] Publisher finished. Waiting 5s for late messages..."
    sleep 5

    # --- Stop Subscriber for this run ---
    echo "[$(date '+%H:%M:%S')] Shutting down subscriber (PID: $SUB_PID)..."
    kill $SUB_PID 2>/dev/null
    wait $SUB_PID 2>/dev/null
    trap - INT TERM

    # --- Analysis ---
    echo "[$(date '+%H:%M:%S')] Running latency analysis..."
    if [ -f "$TRACE_FILE" ]; then
        echo -e "\n--- RUN $r RESULTS ---" >> "$SUMMARY_FILE"
        python3 analyze_latency.py --file "$TRACE_FILE" --data-file "$DATA_FILE" --msg-size "$DATASIZE" | tee -a "$SUMMARY_FILE"
    else
        echo "Error: $TRACE_FILE not found. Testing might have failed."
        echo "=== SUBSCRIBER LOGS ==="
        cat "$SUB_LOG" 2>/dev/null || echo "No subscriber logs found."
        echo "======================="
        echo "Run $r failed - no trace file" >> "$SUMMARY_FILE"
    fi

    if [ "$r" -lt "$RUNS" ]; then
        echo "[$(date '+%H:%M:%S')] Run $r complete. Waiting $WAIT_TIME before next run..."
        sleep "$WAIT_TIME"
    fi
done

# Archive the summary file as well
ARCHIVE_DIR="analyze/msg_size_${DATASIZE}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
cp "$SUMMARY_FILE" "$ARCHIVE_DIR/summary-${TIMESTAMP}.log"

echo "========================================================="
echo "All $RUNS runs completed successfully."
echo "Full summary available in: $SUMMARY_FILE"
echo "Summary archived to: $ARCHIVE_DIR/summary-${TIMESTAMP}.log"
echo "========================================================="
