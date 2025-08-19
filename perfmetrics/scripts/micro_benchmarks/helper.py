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

import sys
from datetime import datetime, timedelta
from google.cloud import bigquery
import os
import subprocess
import pandas as pd

PROJECT_ID = "gcs-fuse-test-ml"
DATASET_ID = "benchmark_results"
TABLE_ID = "gcsfuse_benchmarks"

def mount_bucket(mount_dir: str, bucket_name: str, flags: str) -> bool:
  """
  Mounts a Google Cloud Storage (GCS) bucket using gcsfuse.

  This function attempts to create the mount directory (if it doesn't exist),
  then runs the `gcsfuse` command to mount the specified GCS bucket using the provided flags.

  Args:
      mount_dir (str): The local directory where the GCS bucket should be mounted.
      bucket_name (str): The name of the GCS bucket to mount.
      flags (str): Additional flags or options to pass to the `gcsfuse` command
                   (e.g., "--implicit-dirs").

  Returns:
      bool:
          - True if the mount operation succeeded.
          - False if the `gcsfuse` command failed (e.g., due to permissions, bucket issues, etc.).

  Prints:
      Status messages indicating success or failure of the mount operation.
  """
  os.makedirs(mount_dir, exist_ok=True)
  cmd = f"gcsfuse {flags} {bucket_name} {mount_dir}"
  print(f"Mounting: {cmd}")
  try:
    subprocess.run(cmd, shell=True, check=True)
    print(f"Successfully mounted {bucket_name} at {mount_dir}")
    # Set read_ahead_kb as requested.
    post_mount_cmd = (f"export MOUNT_POINT='{mount_dir}'; "
                      f"echo 1024 | sudo tee /sys/class/bdi/0:$(stat -c '%d' $MOUNT_POINT)/read_ahead_kb")
    print(f"Running post-mount command for {mount_dir}...")
    try:
      # Using /bin/bash to ensure 'export' and command substitution work as expected.
      subprocess.run(post_mount_cmd, shell=True, check=True, executable='/bin/bash')
      print("Post-mount command executed successfully.")
    except subprocess.CalledProcessError as e:
      print(f"Failed to execute post-mount command: {e}")
      # Attempt to unmount before failing to leave a clean state.
      unmount_gcs_directory(mount_dir)
      return False
    return True
  except subprocess.CalledProcessError as e:
    print(f"Failed to mount {bucket_name} at {mount_dir}: {e}")
    return False


def unmount_gcs_directory(mount_point: str) -> bool:
  """
  Unmounts a GCS bucket that was mounted with gcsfuse.

  Args:
      mount_point (str): The local mount point directory.

  Prints:
      Success or failure message based on the unmount operation.

   Returns:
      bool:
          - True if the mount operation succeeded.
          - False if the `fusermount` command failed.
  """
  try:
    subprocess.run(["fusermount", "-u", mount_point], check=True)
    print(f"Successfully unmounted {mount_point}")
    return True
  except subprocess.CalledProcessError as e:
    print(f"Failed to unmount {mount_point}: {e}. Ensure the directory is correctly mounted.")
    return False


def log_to_bigquery(start_time_sec: float, duration_sec: float, total_bytes: int, gcsfuse_config: str, workload_type: str) -> None:
  """Logs performance metrics to a BigQuery table.

  This function calculates bandwidth, creates a pandas DataFrame with the
  provided data, converts the data to the appropriate types, and then inserts
  the data into the specified BigQuery table. If the table does not exist,
  this query can be used to create it:

  CREATE TABLE `your-project-id.benchmark_results.gcsfuse_benchmarks` (
      start_time TIMESTAMP,
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
  print(f"Duration: {duration_sec:.2f}s | Data: {total_bytes / (1000 ** 3):.2f} GB | Bandwidth: {bandwidth_mbps:.2f} MB/s")

  client = bigquery.Client(project=PROJECT_ID)
  table_ref = client.dataset(DATASET_ID).table(TABLE_ID)

  df = pd.DataFrame([{
      "start_time": datetime.fromtimestamp(start_time_sec),
      "duration_seconds": duration_sec,
      "bandwidth_mbps": bandwidth_mbps,
      "gcsfuse_config": gcsfuse_config,
      "workload_type": workload_type,
  }])

  df['start_time'] = pd.to_datetime(df['start_time'])
  df['duration_seconds'] = df['duration_seconds'].astype(float)
  df['bandwidth_mbps'] = df['bandwidth_mbps'].astype(float)

  # client.load_table_from_dataframe(df, table_ref).result()
  # print("Successfully logged data to BigQuery.")


def get_last_n_days_bandwidth_entries(
    client: bigquery.Client,
    table_ref: bigquery.TableReference,
    workload_type: str,
    days: int = 3
) -> list[float]:
  """
  Fetches bandwidth measurements (in MB/s) for a given workload type
  from the last 'n' days of records in the specified BigQuery table.

  Args:
      client (bigquery.Client): Authenticated BigQuery client instance.
      table_ref (bigquery.TableReference): Reference to the BigQuery table.
      workload_type (str): Type of workload to filter for (e.g., "read" or "write").
      days (int): Number of past days to look back from the current time.

  Returns:
      list[float]: A list of bandwidth values (in MB/s). Returns an empty list if no data is found or an error occurs.
  """
  full_table_name = f"`{table_ref.project}.{table_ref.dataset_id}.{table_ref.table_id}`"
  time_ago = datetime.now() - timedelta(days=days)
  time_ago_str = time_ago.strftime('%Y-%m-%d %H:%M:%S')

  query = f"""
        SELECT bandwidth_mbps
        FROM {full_table_name}
        WHERE start_time >= TIMESTAMP('{time_ago_str}')
          AND workload_type = '{workload_type}'
        ORDER BY start_time DESC
    """

  bandwidths = []
  try:
    query_job = client.query(query)
    rows = query_job.result()
    for row in rows:
      bandwidths.append(row.bandwidth_mbps)
  except Exception as e:
    print(f"Error fetching bandwidth entries for the past {days} days: {e}")

  return bandwidths


def check_and_alert_bandwidth(bandwidth_threshold_mbps: float, workload_type: str) -> None:
  """
  Validates current bandwidth performance for a workload by comparing it against
  a historical average from the past 3 days. If the average bandwidth is below
  the defined threshold, the function prints a warning and exits with status code 1.

  Args:
      bandwidth_threshold_mbps (float): Minimum acceptable bandwidth (in MB/s).
      workload_type (str): The type of workload being evaluated (e.g., "read" or "write").
  """
  client = bigquery.Client(project=PROJECT_ID)
  table_ref = client.dataset(DATASET_ID).table(TABLE_ID)

  print("\n--- Bandwidth Validation: Comparing Against Last 3 Days Average ---")
  last_three_days_bandwidths = get_last_n_days_bandwidth_entries(
      client, table_ref, workload_type, days=3
  )

  if last_three_days_bandwidths:
    avg_past_bandwidth = sum(last_three_days_bandwidths) / len(last_three_days_bandwidths)
    print(f"Workload Type       : {workload_type}")
    print(f"3-Day Average       : {avg_past_bandwidth:.2f} MB/s")
    print(f"Configured Threshold: {bandwidth_threshold_mbps:.2f} MB/s")

    if avg_past_bandwidth < bandwidth_threshold_mbps:
      print("FAILURE: 3-day average bandwidth is below the threshold.")
      print("\n----------------------------------\n")
      sys.exit(1)  # Fail the Kokoro build
    else:
      print("Bandwidth is within acceptable range.")
  else:
    print(f"No recent data available for '{workload_type}' workload in the last 3 days.")

  print("\n----------------------------------\n")
