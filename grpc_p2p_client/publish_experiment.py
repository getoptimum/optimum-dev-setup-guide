import subprocess
import itertools
import argparse

# Set up argument parsing
parser = argparse.ArgumentParser(description="Run p2p_client_multi_streams_publish with variable datasize and frequency.")
parser.add_argument("-tracefile", required=True, help="Path to the trace output file")
parser.add_argument("-ipfile", required=True, help="Path to file with the p2p node IPs")
args = parser.parse_args()

# Define the parameter sets
datasizes = [3000, 30000, 300000]
#datasizes = [3000]
frequencies = ["0.5s", "1s", "5s"]
count = 5

# Command template
command_template = [
    "./p2p_client_multi_streams_publish",
    "-start-index", "1",
    "-end-index", "2",
    "-ipfile", args.ipfile,
    "-topic", "topicA",
    "-count", str(count),
    "-sleep", "",       # placeholder
    "-datasize", "",    # placeholder
    "-output", "outgoing.txt"
]

# Iterate over all combinations of datasize and frequency
for datasize, freq in itertools.product(datasizes, frequencies):
    # Set the dynamic values
    command_template[12] = freq       # -sleep value
    command_template[14] = str(datasize)  # -datasize value
    command_template[16] = str(datasize)+"-"+freq+"-output.txt"  # -output value

    # Print the command for debugging
    print("Running command:", " ".join(command_template))

    # Execute the command
    subprocess.run(command_template)


# Parse the tracefile and record the outcomes
command_process = [
    "python3",
    "../../optimum-infra/utilities/compute-messages-latency-from-trace.py",
    "-i", args.tracefile
]

# Print the command for debugging
print("Running command:", " ".join(command_process))

with open("process.out", "w") as f:
    subprocess.run(
        command_process,
        stdout=f,
        stderr=subprocess.STDOUT
    )
    
# Read all lines from the trace file
with open("process.out", "r") as f:
    all_lines = f.readlines()

# Filter only data lines (ignore header or previous average lines)
data_lines = [line for line in all_lines if line.strip() and not line.startswith("#") and not line.startswith("msgid")]

# Start the output file
with open("computed.out", "w") as f:
    f.write(f"{"msgid":10}\t{"n":10}\t{"mean":10}\t{"p90":10}\t{"p95":10}\n")

iteration = 0
# Iterate over all combinations of datasize and frequency
for datasize, freq in itertools.product(datasizes, frequencies):
    # Determine the slice for this iteration
    start_idx = iteration * count 
    end_idx = start_idx + count
    iteration_lines = data_lines[start_idx:end_idx]

    # Compute averages
    total_mean = 0
    total_p90 = 0
    total_p95 = 0

    for line in iteration_lines:
        parts = line.split()
        mean = float(parts[2])
        p90 = float(parts[3])
        p95 = float(parts[4])
        total_mean += mean
        total_p90 += p90
        total_p95 += p95

    count_lines = len(iteration_lines)
    avg_mean = total_mean / count_lines if count_lines else 0
    avg_p90 = total_p90 / count_lines if count_lines else 0
    avg_p95 = total_p95 / count_lines if count_lines else 0

    # Append averages to the trace file
    with open("computed.out", "a") as f:
        for line in iteration_lines:
            f.write(line.rstrip() + "\n")
        f.write(f"# datasize={datasize} freq={freq} avg_mean={avg_mean:.2f} avg_p90={avg_p90:.2f} avg_p95={avg_p95:.2f}\n\n")

    print(f"Appended averages to trace file: datasize={datasize}, freq={freq}")
    iteration += 1
