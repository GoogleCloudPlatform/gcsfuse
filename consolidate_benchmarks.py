
import os
import json
import csv
import re
from collections import defaultdict
import statistics

def consolidate_benchmark_results():
    """
    Scans a directory of FIO JSON results, extracts read bandwidth for each run,
    and consolidates them into a single CSV file with mean and standard deviation.
    """
    # --- Configuration ---
    INPUT_DIR = os.path.expanduser('~/tmp/bs_results')
    OUTPUT_CSV_PATH = 'consolidated_benchmarks.csv'
    RUN_COUNT = 5
    # -------------------

    # Use a defaultdict to easily append to lists
    results = defaultdict(lambda: [None] * RUN_COUNT)

    # Regex to parse filename, e.g., "fio_results_30M_run2.json"
    # It captures the block size (e.g., "30M") and the run number (e.g., "2")
    filename_pattern = re.compile(r"fio_results_(\w+)_run(\d+)\.json")

    if not os.path.isdir(INPUT_DIR):
        print(f"Error: Input directory not found at {INPUT_DIR}")
        return

    print(f"Scanning for files in {INPUT_DIR}...")

    for filename in os.listdir(INPUT_DIR):
        match = filename_pattern.match(filename)
        if match:
            block_size = match.group(1)
            run_number = int(match.group(2))
            
            file_path = os.path.join(INPUT_DIR, filename)
            
            try:
                with open(file_path, 'r') as f:
                    data = json.load(f)
                    # Extract the read bandwidth: data['jobs'][0]['read']['bw']
                    # The user-provided path was slightly off, FIO output is usually nested under 'jobs'
                    if 'jobs' in data and len(data['jobs']) > 0:
                        read_stats = data['jobs'][0].get('read', {})
                        bw = read_stats.get('bw')
                        
                        if bw is not None and 1 <= run_number <= RUN_COUNT:
                            results[block_size][run_number - 1] = bw / 1024
                        else:
                            print(f"Warning: Could not find 'bw' or run number out of range in {filename}")
                    else:
                        print(f"Warning: Could not find 'jobs' in the expected structure in {filename}")

            except (json.JSONDecodeError, IOError) as e:
                print(f"Warning: Could not read or parse {filename}. Error: {e}")

    if not results:
        print("No matching result files found. Exiting.")
        return

    print(f"Processing complete. Writing results to {OUTPUT_CSV_PATH}...")

    with open(OUTPUT_CSV_PATH, 'w', newline='') as csvfile:
        writer = csv.writer(csvfile)
        
        # Write header
        header = ['block_size'] + [f'run{i+1}_read_bw' for i in range(RUN_COUNT)] + ['mean', 'std_dev']
        writer.writerow(header)
        
        # Write data rows for each block size, sorted numerically by block size
        for block_size, runs in sorted(results.items(), key=lambda item: int(re.sub(r'\D', '', item[0]))):
            # Filter out None values for calculation in case of missing runs
            valid_runs = [r for r in runs if r is not None]
            
            mean = statistics.mean(valid_runs) if len(valid_runs) > 1 else (valid_runs[0] if valid_runs else 0)
            std_dev = statistics.stdev(valid_runs) if len(valid_runs) > 1 else 0
            
            # Prepare the row for CSV writing
            row = [block_size] + runs + [f"{mean:.2f}", f"{std_dev:.2f}"]
            writer.writerow(row)

    print("CSV file generated successfully.")

if __name__ == '__main__':
    consolidate_benchmark_results()
