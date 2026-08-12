#!/usr/bin/env python3
import csv
import os
import matplotlib.pyplot as plt
import numpy as np

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
CSV_FILE = os.path.join(SCRIPT_DIR, "../results/benchmark_results.csv")
OUTPUT_DIR = os.path.join(SCRIPT_DIR, "../results/plots")

def load_data(csv_path):
    data = []
    with open(csv_path, mode='r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row in reader:
            # Convert values to float where applicable
            for k, v in row.items():
                if k not in ['Version', 'Protocol']:
                    try:
                        row[k] = float(v)
                    except ValueError:
                        row[k] = 0.0
            data.append(row)
    return data

def plot_grouped_bar(data, metrics, title, ylabel, filename, log_scale=False):
    labels = [f"{r['Version']} ({r['Protocol']})" for r in data]
    x = np.arange(len(metrics))  # the label locations
    width = 0.8 / len(data)  # the width of the bars

    fig, ax = plt.subplots(figsize=(12, 7))
    
    for i, row in enumerate(data):
        values = [row[m] for m in metrics]
        # Shift bars for grouping
        pos = x - 0.4 + (i * width) + (width / 2)
        label = f"{row['Version']} ({row['Protocol']})"
        bars = ax.bar(pos, values, width, label=label)
        ax.bar_label(bars, fmt='%.1f', padding=3, fontsize=8)

    # Add some text for labels, title and custom x-axis tick labels, etc.
    ax.set_ylabel(ylabel)
    ax.set_title(title)
    ax.set_xticks(x)
    ax.set_xticklabels(metrics, rotation=15, ha='right')
    ax.legend()

    if log_scale:
        ax.set_yscale('log')
        ax.set_ylabel(ylabel + " (Log Scale)")

    fig.tight_layout()
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    plt.savefig(os.path.join(OUTPUT_DIR, filename), dpi=300)
    plt.close()
    print(f"Saved plot: {filename}")

def main():
    if not os.path.exists(CSV_FILE):
        print(f"Error: CSV file not found at {CSV_FILE}")
        return

    data = load_data(CSV_FILE)
    
    if not data:
        print("No data found in CSV.")
        return

    # 1. Small Ops Latency
    small_ops_metrics = ['Small read (4KB)', 'Small write (4KB)']
    plot_grouped_bar(data, small_ops_metrics, 'Small Ops Latency', 'Latency (ms)', 'small_ops_latency.png')

    # 2. Append Latency (Log scale recommended due to huge variation)
    append_metrics = ['Append — close per record', 'Append — streaming (64KB)', 'Append — to 1 GB file']
    plot_grouped_bar(data, append_metrics, 'Append Latency (Log Scale)', 'Latency (ms)', 'append_latency.png', log_scale=True)

    # 3. Concurrency Scaling
    concurrency_metrics = ['Small read (1 thread)', 'Small read (8 threads)', 'Small write (1 thread)', 'Small write (8 threads)']
    plot_grouped_bar(data, concurrency_metrics, 'Concurrency Scaling', 'Throughput (Files/sec)', 'concurrency_scaling.png')

    # 4. Large Sequential Throughput
    seq_metrics = ['Large write (256 MB)', 'Large read cold (1 GB)']
    plot_grouped_bar(data, seq_metrics, 'Large Sequential Throughput', 'Throughput (MB/s)', 'sequential_throughput.png')

if __name__ == "__main__":
    main()
