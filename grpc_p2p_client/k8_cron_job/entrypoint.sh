#!/bin/bash

# ==============================================================================
# P2P Load Test Kubernetes Entrypoint
# ==============================================================================

# 1. Accept Environment Variables (with defaults)
BATCH=${BATCH:-1}
SLEEP=${SLEEP:-"450ms"}
DATASIZE=${DATASIZE:-80000}
COUNT=${COUNT:-4000}
IPFILE=${IPFILE:-"ip.p2pnode.bnb.tsv"}
START_IDX=${START_IDX:-0}
END_IDX=${END_IDX:-1}

# 2. Check if TOPIC is provided (required)
if [ -z "$TOPIC" ]; then
    echo "Error: TOPIC environment variable is required."
    exit 1
fi

# 3. Check if IP file exists and is not empty
if [ ! -s "$IPFILE" ]; then
    echo "Error: IP file '$IPFILE' is missing or empty."
    exit 1
fi

echo "========================================================="
echo "Starting P2P Load Test (Kubernetes Mode)"
echo "Configuration: Topic=$TOPIC, Batch=$BATCH, Sleep=$SLEEP, DataSize=$DATASIZE, Count=$COUNT"
echo "IP File: $IPFILE (Start: $START_IDX, End: $END_IDX)"
echo "========================================================="

TRACE_FILE="incoming-trace.tsv"
DATA_FILE="incoming-data.tsv"
SUB_LOG="subscriber.log"

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

# --- Stop Subscriber ---
echo "[$(date '+%H:%M:%S')] Shutting down subscriber (PID: $SUB_PID)..."
kill $SUB_PID 2>/dev/null
wait $SUB_PID 2>/dev/null
trap - INT TERM

# --- Analysis ---
echo "[$(date '+%H:%M:%S')] Running latency analysis..."
if [ -f "$TRACE_FILE" ]; then
    echo -e "\n--- TEST RESULTS ---"
    python3 analyze_latency.py --file "$TRACE_FILE" --data-file "$DATA_FILE" --msg-size "$DATASIZE"
else
    echo "Error: $TRACE_FILE not found. Testing might have failed."
    echo "=== SUBSCRIBER LOGS ==="
    cat "$SUB_LOG" 2>/dev/null || echo "No subscriber logs found."
    echo "======================="
    exit 1
fi

echo "========================================================="
echo "Test completed successfully."
echo "========================================================="
