# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import pandas as pd
import fsspec
import gcsfs
import logging
from concurrent.futures import ThreadPoolExecutor
from tqdm import tqdm
import argparse
import os
import numpy as np

# Initialize logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(filename)s:%(lineno)d - %(message)s')
logger = logging.getLogger(__name__)

def process_csv(file, fs, chunk_size=1000):
    """Reads and processes each CSV file in chunks."""
    try:
        # Read CSV file in chunks
        with fs.open(file, 'r') as f:
            df_iterator = pd.read_csv(f, chunksize=chunk_size)
            result_df = pd.concat(df_iterator, ignore_index=True)
            return result_df
    except Exception as e:
        logger.error(f"Error reading file {file}: {e}")
        return None

def calculate_latency_statistics(df):
    # Ensure that 'timestamp' is in datetime format
    df['timestamp'] = pd.to_datetime(df['timestamp'], unit='s')

    # Create a new column 'Time' with the formatted timestamp (HH:MM)
    df['Time'] = df['timestamp'].dt.strftime('%H:%M')

    # Group by 'Time' (each minute) and calculate the latency statistics
    result_df = df.groupby('Time').agg(
        min_latency=('sample_lat', 'min'),
        mean_latency=('sample_lat', 'mean'),
        p90_latency=('sample_lat', lambda x: x.quantile(0.9)),
        p99_9_latency=('sample_lat', lambda x: x.quantile(0.999)),
        p99_99_latency=('sample_lat', lambda x: x.quantile(0.9999)),
        p99_999_latency=('sample_lat', lambda x: x.quantile(0.99999)),
        p99_9999_latency=('sample_lat', lambda x: x.quantile(0.999999)),
        p99_99999_latency=('sample_lat', lambda x: x.quantile(0.9999999)),
        max_latency=('sample_lat', 'max')
    ).reset_index()

    return result_df

def analyze_metrics(path, timestamp_filter=True, chunk_size=1000, output_path="output_latencies.csv"):
    try:
        # Set up file system (GCS or local)
        if path.startswith("gs://"):
            fs = gcsfs.GCSFileSystem()
        else:
            fs = fsspec.filesystem("local")

        # Get list of CSV files
        csv_files = list(fs.glob(path))
        if not csv_files:
            logger.error("No CSV files found.")
            return None

        logger.info(f"Total number of CSV files: {len(csv_files)}")

        # Process files in parallel and collect results
        all_stats = []  # List to store results
        with ThreadPoolExecutor() as pool:
            for result_df in tqdm(pool.map(lambda file: process_csv(file, fs, chunk_size), csv_files), total=len(csv_files)):
                if result_df is not None:
                    # Calculate statistics for each chunk of data
                    stats = calculate_latency_statistics(result_df)
                    all_stats.append(stats)

        # Combine the results
        if all_stats:
            final_stats = pd.concat(all_stats, ignore_index=True)

            # Save to CSV
            final_stats.to_csv(output_path, index=False)
            logger.info(f"Results saved to {output_path}")
            return output_path

    except Exception as e:
        logger.error(f"Error in analyzing metrics: {e}")
        return None

def main():
    # Command-line argument parsing
    parser = argparse.ArgumentParser(description="Analyze latency metrics from CSV files.")
    parser.add_argument('--metrics-path', type=str, required=True, help="Path to CSV files (e.g., gs://bucket/* or local path).")
    parser.add_argument('--output-path', type=str, default="output_latencies.csv", help="Output path for the results (CSV format).")
    parser.add_argument('--chunk-size', type=int, default=1000, help="Chunk size for processing CSV files.")
    
    args = parser.parse_args()

    # Call the analyze_metrics function with the provided arguments
    output = analyze_metrics(args.metrics_path, chunk_size=args.chunk_size, output_path=args.output_path)
    
    if output:
        logger.info(f"Analysis complete. Output saved to: {output}")
    else:
        logger.error("Analysis failed.")

if __name__ == "__main__":
    main()
