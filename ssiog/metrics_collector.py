#!/usr/bin/env python3
# Copyright 2024 Google LLC
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

def process_csv(file, fs):
    with fs.open(file, 'r') as f:
        df = pd.read_csv(f)
        if not df.empty:
            return df['timestamp'][0], df['timestamp'][len(df['timestamp']) - 1], df
        else:
            return None, None, df

            

def analyze_metrics(path, timestamp_filter=True):
    """
    Analyzes metrics from CSV files in a Google Cloud Storage bucket or local filesystem.

    Args:
        path (str): The path to the bucket or local containing CSV files, e.g., "gs://my-bucket/path/to/files/*.csv", "/local/*.csv"

    Returns:
        A pandas DataFrame containing the combined latency data, or None if no files are found. Also, timebased filtering which selects
        common entry among all the CSV files if timestamp_filter is set to True.
    """
    try:
        if path.startswith("gs://"): # if gcs path
            fs = gcsfs.GCSFileSystem()
        else: # otherwise assume it a local path
            fs = fsspec.filesystem("local")
        
        # Find all CSV files in the path using glob-like pattern matching.
        csv_files = list(fs.glob(path))
        if not csv_files:
            return None
        
        logger.info(f"Total number of CSV files: {len(csv_files)}")
        systemMemory = get_system_memory()
        logger.info(f"Total system memory: {systemMemory[0]} MiB" )
        logger.info(f"Used system memory: {systemMemory[1]} MiB")
        logger.info(f"Free system memory: {systemMemory[2]} MiB")
        logger.info(f"Memory usage by process before loading CSV files: {get_memory_usage()} MiB")    

        with ThreadPoolExecutor() as pool:
            results = list(tqdm(pool.map(lambda file: process_csv(file, fs), csv_files), total=len(csv_files)))
            
        start_timestamps = []
        end_timestamps = []        
        all_data = []
        for start, end, df in results:
            if start is not None and end is not None:
                start_timestamps.append(start)
                end_timestamps.append(end)
            all_data.append(df)
        
        combined_df = pd.concat(all_data)    
        logger.info(f"Memory usage by process after loading CSV files: {get_memory_usage()} MiB")    
        
        if not start_timestamps or not end_timestamps:
            return None

        min_timestamp = max(start_timestamps)
        max_timestamp = min(end_timestamps)
        
        # Filter which is not recorded b/w min_timestamp and max_timestamp
        if timestamp_filter:
            combined_df['timestamp'] = pd.to_datetime(combined_df['timestamp'], unit='s')
            combined_df = combined_df[(combined_df['timestamp'] >= pd.to_datetime(min_timestamp, unit='s')) & (combined_df['timestamp'] <= pd.to_datetime(max_timestamp, unit='s'))]
        
        if combined_df.empty:
            return None
    
        return combined_df
    
    except Exception as e:
        logger.error(f"Error in analyzing metrics: {e}")
        return None

def parse_args():
    parser = argparse.ArgumentParser(description="Analyze metrics from GCS")
    
    parser.add_argument(
        "--metrics-path",
        type=str,
        help="GCS or local path to metrics files",
        default="gs://princer-ssiog-data-bkt-uc1/test_0_7_0-0/ssiog-training-n69qj/*.csv"
    )
    parser.add_argument(
        "--timestamp-filter",
        action="store_true",
        help="Filter by common timestamps")
    
    return parser.parse_args()


# Create a main executor which provides a hardcoded path to analyze the metrics create a main method instead
def main():
    args = parse_args()
    result_df = analyze_metrics(args.metrics_path, args.timestamp_filter)
    if result_df is not None:
        print(result_df['sample_lat'].describe(percentiles=[0.05, 0.1, 0.25, 0.5, 0.9, 0.99, 0.999]))

if __name__ == "__main__":
    main()
