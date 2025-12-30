#!/usr/bin/env python3
import argparse
import json
import re
import sys
import collections
import os

# Try to import matplotlib
try:
    import matplotlib.pyplot as plt
    HAS_MATPLOTLIB = True
except ImportError:
    HAS_MATPLOTLIB = False

def parse_logs(logfile):
    """
    Parses the log file and extracts read operations for each file handle.
    Returns:
        reads: dict mapping handle_id -> list of (timestamp, offset, size)
        handle_info: dict mapping handle_id -> {inode: ...}
    """
    reads = collections.defaultdict(list)
    handle_info = {}

    # Regex to extract ReadFile info
    # Example: "fuse_debug: Op 0x000000b6        connection.go:453] <- ReadFile (inode 3, PID 1765689, handle 7, offset 0, 4096 bytes)"
    read_pattern = re.compile(r'<- ReadFile \(inode\s+(\d+),\s+PID\s+\d+,\s+handle\s+(\d+),\s+offset\s+(\d+),\s+(\d+)\s+bytes\)')

    print(f"Parsing {logfile}...")
    count = 0
    with open(logfile, 'r') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                entry = json.loads(line)
            except json.JSONDecodeError:
                continue

            timestamp = entry.get('timestamp', {})
            if isinstance(timestamp, dict):
                seconds = int(timestamp.get('seconds', 0))
                nanos = int(timestamp.get('nanos', 0))
                ts = seconds + nanos / 1e9
            else:
                continue

            message = entry.get('message', '')
            if not message:
                continue

            # Check for ReadFile
            match = read_pattern.search(message)
            if match:
                inode = match.group(1)
                handle = match.group(2)
                offset = int(match.group(3))
                size = int(match.group(4))

                reads[handle].append((ts, offset, size))
                if handle not in handle_info:
                    handle_info[handle] = {'inode': inode}
                count += 1

    print(f"Parsed {count} ReadFile events.")
    return reads, handle_info

def analyze_sequentiality(reads):
    results = {}
    for handle, events in reads.items():
        # events: list of (ts, offset, size)
        # Sort by time
        sorted_events = sorted(events, key=lambda x: x[0])

        if not sorted_events:
            results[handle] = "No reads"
            continue

        sequential_count = 0
        total_transitions = len(sorted_events) - 1

        if total_transitions == 0:
            results[handle] = "Single read"
            continue

        for i in range(1, len(sorted_events)):
            prev = sorted_events[i-1]
            curr = sorted_events[i]

            # Check if current offset is previous offset + previous size
            if curr[1] == prev[1] + prev[2]:
                sequential_count += 1

        percent = (sequential_count / total_transitions) * 100
        results[handle] = f"{percent:.1f}% sequential"

    return results

def plot_reads(reads, handle_info, output_file):
    if not HAS_MATPLOTLIB:
        print("Error: matplotlib is not installed.")
        print("Please install it to generate the plot:")
        print("  pip install matplotlib")
        return

    plt.figure(figsize=(12, 6))

    # Sort handles numerically
    sorted_handles = sorted(reads.keys(), key=lambda h: int(h))

    # Calculate global start time to normalize
    all_times = [t for events in reads.values() for t, _, _ in events]
    if not all_times:
        print("No data to plot.")
        return
    min_time = min(all_times)

    for handle in sorted_handles:
        events = reads[handle]
        # sort events by time
        events.sort(key=lambda x: x[0])

        times = [e[0] - min_time for e in events]
        offsets = [e[1] for e in events]

        inode = handle_info.get(handle, {}).get('inode', '?')
        label = f"Handle {handle} (Inode {inode})"

        # Plot points
        plt.scatter(times, offsets, label=label, alpha=0.7, s=30, edgecolors='none')

    plt.xlabel("Time (s) from first read")
    plt.ylabel("Offset (bytes)")
    plt.title("File Read Patterns (Sequential vs Random)")
    plt.legend()
    plt.grid(True, which='both', linestyle='--', linewidth=0.5)

    # Save the plot
    try:
        plt.savefig(output_file, dpi=100)
        print(f"Plot saved to {os.path.abspath(output_file)}")
    except Exception as e:
        print(f"Failed to save plot: {e}")

def main():
    parser = argparse.ArgumentParser(description="Visualize file handle read patterns from GCSFuse logs.")
    parser.add_argument("logfile", help="Path to the JSON log file.")
    parser.add_argument("--output", "-o", default="read_pattern.png", help="Output image file path.")

    args = parser.parse_args()

    if not os.path.exists(args.logfile):
        print(f"Error: Log file '{args.logfile}' not found.")
        sys.exit(1)

    reads, handle_info = parse_logs(args.logfile)

    if not reads:
        print("No read operations found in the logs.")
        sys.exit(0)

    print(f"Found {len(reads)} handles with read operations.")

    analysis = analyze_sequentiality(reads)

    for handle in sorted(reads.keys(), key=lambda h: int(h)):
        count = len(reads[handle])
        inode = handle_info[handle]['inode']
        seq_info = analysis.get(handle, "")
        print(f"  Handle {handle} (Inode {inode}): {count} reads, {seq_info}")

    plot_reads(reads, handle_info, args.output)

if __name__ == "__main__":
    main()
