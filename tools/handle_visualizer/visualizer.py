#!/usr/bin/env python3
import argparse
import json
import re
import sys
import collections
import os
import time
import threading

# Try to import matplotlib
try:
    import matplotlib.pyplot as plt
    import matplotlib.animation as animation
    HAS_MATPLOTLIB = True
except ImportError:
    HAS_MATPLOTLIB = False

class LogParser:
    def __init__(self):
        # Regex to extract ReadFile info
        self.read_pattern = re.compile(r'<- ReadFile \(inode\s+(\d+),\s+PID\s+\d+,\s+handle\s+(\d+),\s+offset\s+(\d+),\s+(\d+)\s+bytes\)')
        self.reads = collections.defaultdict(list)
        self.handle_info = {}
        self.lock = threading.Lock()
        self.count = 0

    def parse_line(self, line):
        line = line.strip()
        if not line:
            return None
        try:
            entry = json.loads(line)
        except json.JSONDecodeError:
            return None

        timestamp = entry.get('timestamp', {})
        if isinstance(timestamp, dict):
            seconds = int(timestamp.get('seconds', 0))
            nanos = int(timestamp.get('nanos', 0))
            ts = seconds + nanos / 1e9
        else:
            return None

        message = entry.get('message', '')
        if not message:
            return None

        # Check for ReadFile
        match = self.read_pattern.search(message)
        if match:
            inode = match.group(1)
            handle = match.group(2)
            offset = int(match.group(3))
            size = int(match.group(4))

            with self.lock:
                self.reads[handle].append((ts, offset, size))
                if handle not in self.handle_info:
                    self.handle_info[handle] = {'inode': inode}
                self.count += 1
            return True
        return False

    def get_data(self):
        with self.lock:
            # Return a copy to avoid modification during iteration in another thread
            return self.reads.copy(), self.handle_info.copy()

def follow_file(filepath, parser, stop_event):
    """Reads from a file indefinitely (tail -f style)."""
    try:
        if filepath == '-':
            # Read from stdin
            f = sys.stdin
            while not stop_event.is_set():
                line = f.readline()
                if not line:
                    break # EOF for stdin
                parser.parse_line(line)
        else:
            # Read from file
            # Wait for file to exist
            while not os.path.exists(filepath):
                if stop_event.is_set():
                    return
                time.sleep(0.5)

            with open(filepath, 'r') as f:
                # Go to end of file initially?
                # Actually, usually when we say "follow" we might mean from now on,
                # OR if we want to see history, we read from start.
                # Let's read from start.
                while not stop_event.is_set():
                    line = f.readline()
                    if not line:
                        time.sleep(0.1)
                        continue
                    parser.parse_line(line)
    except Exception as e:
        print(f"Error reading file: {e}")

class LiveVisualizer:
    def __init__(self, parser, output_file=None):
        self.parser = parser
        self.output_file = output_file
        self.fig, self.ax = plt.subplots(figsize=(12, 6))
        self.scatters = {} # handle -> PathCollection
        self.start_time = None

    def update(self, frame):
        reads, handle_info = self.parser.get_data()

        if not reads:
            return

        # Calculate start time if not set
        if self.start_time is None:
             all_times = [t for events in reads.values() for t, _, _ in events]
             if all_times:
                 self.start_time = min(all_times)
             else:
                 return

        self.ax.clear()
        self.ax.set_xlabel("Time (s) from first read")
        self.ax.set_ylabel("Offset (bytes)")
        self.ax.set_title("File Read Patterns (Live)")
        self.ax.grid(True, which='both', linestyle='--', linewidth=0.5)

        sorted_handles = sorted(reads.keys(), key=lambda h: int(h))

        for handle in sorted_handles:
            events = reads[handle]
            # No need to sort if appended in order, but safe to sort
            events.sort(key=lambda x: x[0])

            times = [e[0] - self.start_time for e in events]
            offsets = [e[1] for e in events]

            inode = handle_info.get(handle, {}).get('inode', '?')
            label = f"Handle {handle} (Inode {inode})"

            self.ax.scatter(times, offsets, label=label, alpha=0.7, s=30, edgecolors='none')

        self.ax.legend()

        # If running headless/saving to file periodically
        if self.output_file:
             # In some environments, saving inside animation loop might be slow
             try:
                self.fig.savefig(self.output_file, dpi=100)
             except Exception:
                pass

def run_live(logfile, output_file):
    if not HAS_MATPLOTLIB:
        print("Error: matplotlib is not installed.")
        return

    print(f"Starting live visualization for {logfile}...")
    parser = LogParser()
    stop_event = threading.Event()

    # Start reader thread
    t = threading.Thread(target=follow_file, args=(logfile, parser, stop_event), daemon=True)
    t.start()

    # Setup plot
    vis = LiveVisualizer(parser, output_file)

    # Animate
    # interval=1000ms
    ani = animation.FuncAnimation(vis.fig, vis.update, interval=1000, cache_frame_data=False)

    try:
        # Check if we are running in a headless environment/backend
        # 'agg' is the standard non-interactive backend
        if plt.get_backend().lower() == 'agg':
            print(f"Running in headless mode (backend: {plt.get_backend()}). Saving plots to {output_file}.")
            # Manual event loop since plt.show() does nothing for Agg
            while not stop_event.is_set():
                if not t.is_alive():
                    print("Log reader thread ended.")
                    break
                vis.update(0)
                time.sleep(1)
        else:
            plt.show()
    except KeyboardInterrupt:
        print("Stopping...")
    finally:
        stop_event.set()

def analyze_sequentiality(reads):
    results = {}
    for handle, events in reads.items():
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
            if curr[1] == prev[1] + prev[2]:
                sequential_count += 1
        percent = (sequential_count / total_transitions) * 100
        results[handle] = f"{percent:.1f}% sequential"
    return results

def run_static(logfile, output_file):
    if not os.path.exists(logfile):
        print(f"Error: Log file '{logfile}' not found.")
        sys.exit(1)

    parser = LogParser()
    print(f"Parsing {logfile}...")
    with open(logfile, 'r') as f:
        for line in f:
            parser.parse_line(line)

    reads, handle_info = parser.get_data()
    print(f"Parsed {parser.count} ReadFile events.")

    if not reads:
        print("No read operations found in the logs.")
        sys.exit(0)

    print(f"Found {len(reads)} handles with read operations.")
    analysis = analyze_sequentiality(reads)
    for handle in sorted(reads.keys(), key=lambda h: int(h)):
        inode = handle_info[handle]['inode']
        print(f"  Handle {handle} (Inode {inode}): {len(reads[handle])} reads, {analysis.get(handle)}")

    if HAS_MATPLOTLIB:
        vis = LiveVisualizer(parser, output_file)
        vis.update(0) # Draw once
        print(f"Plot saved to {os.path.abspath(output_file)}")
    else:
        print("Matplotlib not installed, skipping plot.")

def main():
    parser = argparse.ArgumentParser(description="Visualize file handle read patterns from GCSFuse logs.")
    parser.add_argument("logfile", help="Path to the JSON log file (or '-' for stdin).")
    parser.add_argument("--output", "-o", default="read_pattern.png", help="Output image file path.")
    parser.add_argument("--live", action="store_true", help="Run in live mode, following the log file.")

    args = parser.parse_args()

    if args.live:
        run_live(args.logfile, args.output)
    else:
        run_static(args.logfile, args.output)

if __name__ == "__main__":
    main()
