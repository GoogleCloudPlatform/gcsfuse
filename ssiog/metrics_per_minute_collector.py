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
import argparse
import logging
import os
import psutil
from concurrent.futures import ThreadPoolExecutor
from tqdm import tqdm
import numpy as np
from collections import defaultdict

# Initialize the global logger with basic INFO level log.
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(filename)s:%(lineno)d - %(message)s')
logger = logging.getLogger(__name__)

def convert_bytes_to_mib(bytes):
    return bytes / (1024 ** 2)

def get_system_memory():
    mem = psutil.virtual_memory()
    return convert_bytes_to_mib(mem.total), convert_bytes_to_mib(mem.used), convert_bytes_to_mib(mem.free)

def get_memory_usage():
    process = psutil.Process(os.getpid())
    mem_info = process.memory_info()
    return convert_bytes_to_mib(mem_info.rss)

def calculate_percentiles(latencies):
    # Calculate percentiles and the min/max latency values
    percentiles = {
        'p90': np.percentile(latencies, 90),
        'p99': np.percentile(latencies, 99),
        'p99.9': np.percentile(latencies, 99.9),
        'p99.99': np.percentile(latencies, 99.99),
        'p99.999': np.percentile(latencies, 99.999),
        'p99.9999': np.percentile(latencies, 99.9999),
        'p99.99999': np.percentile(latencies, 99.99999),
        'min': np.min(latencies),
        'max': np.max(latencies)
    }
    return percentiles

def process_csv(file, fs):
    try:
        with fs.open(file, 'r') as f:
            df = pd.read_csv(f)
            if 'sample_lat' not in df.columns:
                logger.warning(f"File {file} does not contain 'sample_lat' column. Skipping file.")
                return pd.DataFrame()  # Return an empty DataFrame if no 'sample_lat' column
            
            if not df.empty:
                # Assuming 'timestamp' and 'sample_lat' columns are present
                df['timestamp'] = pd.to_datetime(df['timestamp'], unit='s')
                return df
            else:
                return pd.DataFrame()
    except Exception as e:
        logger.error(f"Error processing file {file}: {e}")
        return pd.DataFrame()  # Return an empty DataFrame on error

def analyze_metrics(path, timestamp_filter=True):
    """
    Analyzes metrics from CSV files in a Google Cloud Storage bucket or local filesystem.
    """

    try:
        if path.startswith("gs://"):  # if GCS path
            fs = gcsfs.GCSFileSystem()
        else:  # otherwise assume it's a local path
            fs = fsspec.filesystem("local")
        
        # Find all CSV files in the path using glob-like pattern matching.
        csv_files = list(fs.glob(path))
        if not csv_files:
            return None
        
        logger.info(f"Total number of CSV files: {len(csv_files)}")
        systemMemory = get_system_memory()
        logger.info(f"Total system memory: {systemMemory[0]} MiB")
        logger.info(f"Used system memory: {systemMemory[1]} MiB")
        logger.info(f"Free system memory: {systemMemory[2]} MiB")
        logger.info(f"Memory usage by process before loading CSV files: {get_memory_usage()} MiB")    

        with ThreadPoolExecutor() as pool:
            results = list(tqdm(pool.map(lambda file: process_csv(file, fs), csv_files), total=len(csv_files)))
        
        # Initialize a dictionary to store latency data per minute
        minute_data = defaultdict(list)
        start_timestamps = []
        end_timestamps = []

        # Process data
        for df in results:
            if not df.empty:
                # Round timestamp to the start of the minute (i.e., remove seconds and microseconds)
                df['minute'] = df['timestamp'].dt.floor('min')  # Update to 'min' instead of 'T'
                for minute, group in df.groupby('minute'):
                    minute_data[minute].extend(group['sample_lat'].values)

        # Process per-minute latency data
        processed_metrics = []
        for minute, latencies in minute_data.items():
            if latencies:
                percentiles = calculate_percentiles(latencies)
                processed_metrics.append({
                    'time': minute.strftime('%H:%M'),
                    'min': percentiles['min'],  # Min comes first
                    'p90': percentiles['p90'],
                    'p99': percentiles['p99'],
                    'p99.9': percentiles['p99.9'],
                    'p99.99': percentiles['p99.99'],
                    'p99.999': percentiles['p99.999'],
                    'p99.9999': percentiles['p99.9999'],
                    'p99.99999': percentiles['p99.99999'],
                    'max': percentiles['max']
                })

        # Convert processed metrics into a DataFrame
        result_df = pd.DataFrame(processed_metrics)
        
        logger.info(f"Memory usage by process after loading CSV files: {get_memory_usage()} MiB")
        
        if result_df.empty:
            return None

        return result_df
    
    except Exception as e:
        logger.error(f"Error in analyzing metrics: {e}")
        return None

def parse_args():
    parser = argparse.ArgumentParser(description="Analyze metrics from GCS")
    
    parser.add_argument(
        "--metrics-path",
        type=str,
        help="GCS or local path to metrics files",
        default="gs://vipinydv-metrics/slowenvironment-readstall-genericread-1byte/*.csv"
    )
    parser.add_argument(
        "--timestamp-filter",
        action="store_true",
        help="Filter by common timestamps"
    )
    parser.add_argument(
        "--output-file",
        type=str,
        help="Path to save the processed CSV output",
        default="processed_metrics.csv"  # Default file name if not provided
    )
    
    return parser.parse_args()

def main():
    args = parse_args()
    result_df = analyze_metrics(args.metrics_path, args.timestamp_filter)
    
    if result_df is not None:
        output_file = args.output_file  # Get the file path from the argument
        result_df.to_csv(output_file, index=False)
        logger.info(f"Results have been saved to {output_file}")

if __name__ == "__main__":
    main()
