# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from datetime import datetime
from google.cloud import bigquery
import os
import subprocess
import pandas as pd

PROJECT_ID = "gcs-fuse-test-ml"
DATASET_ID = "benchmark_results"
TABLE_ID = "gcsfuse_benchmarks"

def mount_bucket(mount_dir, bucket_name, flags):
  os.makedirs(mount_dir, exist_ok=True)
  cmd = f"gcsfuse {flags} {bucket_name} {mount_dir}"
  print(f"Mounting: {cmd}")
  subprocess.run(cmd, shell=True, check=True)

def unmount_gcs_directory(mount_point):
  try:
    # For Linux
    subprocess.run(["fusermount", "-u", mount_point], check=True)
    print(f"✅ Successfully unmounted {mount_point}")

  except subprocess.CalledProcessError:
    print(f"❌ Failed to unmount {mount_point}. Ensure the directory is correctly mounted.")

def log_to_bigquery(duration_sec, total_bytes, gcsfuse_config, workload_type):
  # Calculate bandwidth in Mbps
  bandwidth_mbps = total_bytes / duration_sec / 1024 / 1024
  print(f"✅ Duration: {duration_sec:.2f}s | Data: {total_bytes / (1024 ** 3):.2f} GiB | Bandwidth: {bandwidth_mbps:.2f} MB/s")

  # Create a BigQuery client
  client = bigquery.Client(project=PROJECT_ID)

  # Reference to the table
  table_ref = client.dataset(DATASET_ID).table(TABLE_ID)

  # Prepare the DataFrame for upload
  df = pd.DataFrame([{
      "timestamp": datetime.utcnow(),  # Store the timestamp as a datetime object
      "duration_seconds": duration_sec,
      "bandwidth_mbps": bandwidth_mbps,
      "gcsfuse_config": gcsfuse_config,  # Include the flag information
      "workload_type": workload_type,  # Include the operation type (e.g., "read", "write")
  }])

  # Ensure the correct types for BigQuery
  df['timestamp'] = pd.to_datetime(df['timestamp'])  # Convert to datetime object if not already
  df['duration_seconds'] = df['duration_seconds'].astype(float)
  df['bandwidth_mbps'] = df['bandwidth_mbps'].astype(float)

  # Load the DataFrame into BigQuery
  client.load_table_from_dataframe(df, table_ref).result()  # Wait for the load to complete
  print("✅ Successfully logged data to BigQuery.")
