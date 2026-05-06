import sys
import argparse
import os
import shutil
import re
from datetime import datetime
from collections import defaultdict

def get_latest_file_by_mtime(directory, base_filename):
    """Find the most recently modified file in a directory matching the base pattern."""
    if not os.path.exists(directory):
        return None
    
    filename_part, ext = os.path.splitext(base_filename)
    # Match: filename-DATETIME.ext (YYYYMMDD_HHMMSS)
    pattern = re.compile(rf"^{re.escape(filename_part)}-(\d{{8}}_\d{{6}}){re.escape(ext)}$")
    
    latest_file = None
    latest_mtime = -1
    
    for f in os.listdir(directory):
        if pattern.match(f):
            full_path = os.path.join(directory, f)
            mtime = os.path.getmtime(full_path)
            if mtime > latest_mtime:
                latest_mtime = mtime
                latest_file = full_path
    
    return latest_file

def percentile(data, p):
    if not data:
        return 0.0
    data = sorted(data)
    n = len(data)

    # Find the "ideal fractional index" for the target percentile in the array
    k = (n - 1) * (p / 100.0)
    f = int(k)
    c = int(k) + 1 if int(k) + 1 < n else f

    # If the calculated index is exactly an integer, simply return the value at that index
    if f == c:
        return data[f]
    
    # 5. Linear Interpolation
    # If k is calculated as 4.3 (falling between index 4 and index 5)
    # Use weighted average: (value at index 4 with 70% weight) + (value at index 5 with 30% weight)
    d0 = data[f] * (c - k)
    d1 = data[c] * (k - f)
    return d0 + d1

def main():
    parser = argparse.ArgumentParser(description="Calculate P2P Propagation Latency")
    parser.add_argument("--file", default="incoming-trace.tsv", help="Path to trace file")
    parser.add_argument("--data-file", help="Path to data file (optional, will be archived along with trace)")
    parser.add_argument("--skip", type=int, default=5, help="Number of initial messages to skip (warm-up)")
    parser.add_argument("--msg-size", type=str, help="Message size for archiving (e.g., 900)")
    args = parser.parse_args()

    target_file = args.file

    # --- Archiving Logic ---
    if args.msg_size:
        archive_dir = os.path.join("analyze", f"msg_size_{args.msg_size}")
        
        # Scenario A: Local file exists -> Archive it with timestamp
        if os.path.exists(args.file):
            if not os.path.exists(archive_dir):
                os.makedirs(archive_dir)
                print(f"Created archive directory: {archive_dir}")

            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename, ext = os.path.splitext(os.path.basename(args.file))
            new_path = os.path.join(archive_dir, f"{filename}-{timestamp}{ext}")

            # Move trace file
            shutil.move(args.file, new_path)
            print(f"Archived '{args.file}' to '{new_path}'")
            target_file = new_path

            # Also move data file if provided
            if args.data_file and os.path.exists(args.data_file):
                data_filename, data_ext = os.path.splitext(os.path.basename(args.data_file))
                new_data_path = os.path.join(archive_dir, f"{data_filename}-{timestamp}{data_ext}")
                shutil.move(args.data_file, new_data_path)
                print(f"Archived '{args.data_file}' to '{new_data_path}'")
        
        # Scenario B: Local file missing -> Look for latest by mtime in archive
        else:
            latest = get_latest_file_by_mtime(archive_dir, os.path.basename(args.file))
            if latest:
                print(f"Local file '{args.file}' not found. Using most recent archived file: {latest}")
                target_file = latest
            else:
                print(f"Error: Local file '{args.file}' not found and no archived files in '{archive_dir}'.")
                sys.exit(1)
    # ----------------

    if not os.path.exists(target_file):
        print(f"Error: File '{target_file}' not found.")
        sys.exit(1)

    print(f"Reading data from {target_file}...\n")
    
    # Dictionary to group timestamps by MSG_ID
    # msg_id -> list of timestamps
    msg_timestamps = defaultdict(list)
    
    try:
        with open(target_file, 'r') as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                
                parts = line.split('\t')
                if len(parts) >= 6:
                    event = parts[0]
                    # We only care about DELIVER_MESSAGE
                    if event == "DELIVER_MESSAGE":
                        msg_id = parts[3]
                        try:
                            # timestamp is the 6th column (index 5)
                            timestamp = int(parts[5])
                            msg_timestamps[msg_id].append(timestamp)
                        except ValueError:
                            pass
    except Exception as e:
        print(f"Error reading file: {e}")
        return

    if not msg_timestamps:
        print("No DELIVER_MESSAGE events found (or file is empty/invalid).")
        return

    # Calculate statistics for each MsgID
    results = []
    for msg_id, t_list in msg_timestamps.items():
        if not t_list:
            continue
        
        t_start = min(t_list)
        t_end = max(t_list)
        latency_ns = t_end - t_start
        latency_ms = latency_ns / 1_000_000.0
        node_count = len(t_list) # Simplified, assumes distinct peer trace if node counts are unique
        
        results.append({
            "msg_id": msg_id,
            "t_start": t_start,
            "latency_ms": latency_ms,
            "node_count": node_count
        })

    # 3. Sort by t_start (chronological order)
    results = sorted(results, key=lambda x: x["t_start"])
    total_msgs = len(results)
    
    print(f"Discovered a total of {total_msgs} distinct test messages.")

    # 4. Skip warm-up messages
    if total_msgs > args.skip:
        valid_msgs = results[args.skip:]
        print(f"Discarding the first {args.skip} warm-up messages. Using the remaining {len(valid_msgs)} messages for statistical analysis.\n")
    else:
        print(f"Warning: Total messages ({total_msgs}) is less than or equal to the number of skipped warm-up messages ({args.skip}). Using all messages.")
        valid_msgs = results

    # 5. Calculate overall P2P latency statistics (Temporal Distribution)
    if not valid_msgs:
        return

    latencies = [x["latency_ms"] for x in valid_msgs]
    latencies = [x["latency_ms"] for x in valid_msgs]
    
    mean_latency = sum(latencies) / len(latencies)
    max_latency = max(latencies)
    p50 = percentile(latencies, 50)
    p90 = percentile(latencies, 90)
    p95 = percentile(latencies, 95)
    p99 = percentile(latencies, 99)

    print("\n=== P2P Network Propagation Latency Summary ===")
    if args.msg_size:
        print(f"Message Size (Payload)     : {args.msg_size} bytes")
    print(f"Sample Size (after filter) : {len(latencies)} messages")
    print(f"Mean Latency               : {mean_latency:.2f} ms")
    print(f"P50 Latency (Median)       : {p50:.2f} ms")
    print(f"P90 Latency                : {p90:.2f} ms")
    print(f"P95 Latency                : {p95:.2f} ms")
    print(f"P99 Latency                : {p99:.2f} ms")
    print(f"Maximum Latency            : {max_latency:.2f} ms")
    print("===============================================")

    # Display top 5 outliers
    print("\n--- Top 5 Outlier Messages (Highest Latency) ---")
    # Sort messages by latency descending
    outliers = sorted(valid_msgs, key=lambda x: x['latency_ms'], reverse=True)[:5]
    for i, m in enumerate(outliers):
        print(f"{i+1}. Latency: {m['latency_ms']:.2f} ms | MsgID: {m['msg_id']}")

if __name__ == "__main__":
    main()

# run it with `python3 analyze_latency.py --file incoming-trace.tsv --skip 5`