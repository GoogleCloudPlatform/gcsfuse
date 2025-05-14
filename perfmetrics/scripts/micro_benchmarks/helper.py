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

def mount_bucket(mount_dir: str, bucket_name: str, flags: str) -> None:
  """
  Mounts a Google Cloud Storage bucket using gcsfuse.

  Args:
      mount_dir (str): The local directory where the bucket will be mounted.
      bucket_name (str): The name of the GCS bucket to mount.
      flags (str): Additional gcsfuse mount options (e.g., "implicit-dirs,custom-endpoint=...").

  Raises:
      subprocess.CalledProcessError: If the gcsfuse mount command fails.
  """
  os.makedirs(mount_dir, exist_ok=True)
  cmd = f"gcsfuse {flags} {bucket_name} {mount_dir}"
  print(f"Mounting: {cmd}")
  subprocess.run(cmd, shell=True, check=True)

def unmount_gcs_directory(mount_point: str) -> None:
  """
  Unmounts a GCS bucket that was mounted with gcsfuse.

  Args:
      mount_point (str): The local mount point directory.

  Prints:
      Success or failure message based on the unmount operation.
  """
  try:
    subprocess.run(["fusermount", "-u", mount_point], check=True)
    print(f"✅ Successfully unmounted {mount_point}")
  except subprocess.CalledProcessError:
    print(f"❌ Failed to unmount {mount_point}. Ensure the directory is correctly mounted.")

def log_to_bigquery(duration_sec: float, total_bytes: int, gcsfuse_config: str, workload_type: str) -> None:
  """Logs performance metrics to a BigQuery table.

  This function calculates bandwidth, creates a pandas DataFrame with the
  provided data, converts the data to the appropriate types, and then inserts
  the data into the specified BigQuery table. If the table does not exist,
  this query can be used to create it:

  CREATE TABLE `your-project-id.benchmark_results.gcsfuse_benchmarks` (
      timestamp TIMESTAMP,
      duration_seconds FLOAT64,
      bandwidth_mbps FLOAT64,
      gcsfuse_config STRING,
      workload_type STRING
  );

  Args:
      duration_sec (float): Duration of the operation in seconds.
      total_bytes (int): Total data processed in bytes.
      gcsfuse_config (str): Configuration flags used with gcsfuse.
      workload_type (str): Type of workload (e.g., "read", "write").

  Prints:
      Performance metrics and confirmation of successful logging.
  """
  bandwidth_mbps = total_bytes / duration_sec / 1000 / 1000
  print(f"✅ Duration: {duration_sec:.2f}s | Data: {total_bytes / (1000 ** 3):.2f} GiB | Bandwidth: {bandwidth_mbps:.2f} MB/s")

  client = bigquery.Client(project=PROJECT_ID)
  table_ref = client.dataset(DATASET_ID).table(TABLE_ID)

  df = pd.DataFrame([{
      "timestamp": datetime.utcnow(),
      "duration_seconds": duration_sec,
      "bandwidth_mbps": bandwidth_mbps,
      "gcsfuse_config": gcsfuse_config,
      "workload_type": workload_type,
  }])

  df['timestamp'] = pd.to_datetime(df['timestamp'])
  df['duration_seconds'] = df['duration_seconds'].astype(float)
  df['bandwidth_mbps'] = df['bandwidth_mbps'].astype(float)

  client.load_table_from_dataframe(df, table_ref).result()
  print("✅ Successfully logged data to BigQuery.")
