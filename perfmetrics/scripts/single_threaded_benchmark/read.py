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

import os
import subprocess
import time
import argparse
from datetime import datetime
from google.cloud import bigquery
import pandas as pd

# CONFIGURATION
MOUNT_DIR = "/mnt/gcs"
GCS_FILE_NAME = "large-file.bin"  # Change this to your 15GiB file name
NUM_ITERATIONS = 10
PROJECT_ID = "gcs-fuse-test-ml"
DATASET_ID = "benchmark_results"
TABLE_ID = "gcs_read_timings"

def mount_bucket(bucket_name, flags):
  os.makedirs(MOUNT_DIR, exist_ok=True)
  mount_cmd = f"gcsfuse {flags} {bucket_name} {MOUNT_DIR}"
  print(f"Mounting GCS bucket with: {mount_cmd}")
  try:
    subprocess.run(mount_cmd, shell=True, check=True)
  except subprocess.CalledProcessError as e:
    print(f"Mount failed: {e}")
    exit(1)


def read_file(path):
  with open(path, "rb") as f:
    while f.read(1024 * 1024 * 100):  # Read in 100 MiB chunks
      pass


def log_to_bigquery(df: pd.DataFrame, project_id: str, dataset_id: str, table_id: str):
  client = bigquery.Client(project=project_id)
  table_ref = client.dataset(dataset_id).table(table_id)
  job = client.load_table_from_dataframe(df, table_ref)
  job.result()
  print(f"Inserted {len(df)} rows into {dataset_id}.{table_id}")


def main():
  parser = argparse.ArgumentParser(description="Mount GCS bucket and time file reads.")
  parser.add_argument("--bucket", required=True, help="Name of the GCS bucket")
  parser.add_argument("--flags", default="--implicit-dirs", help="Flags for gcsfuse mount")
  parser.add_argument("--iterations", type=int, default=NUM_ITERATIONS, help="Number of read iterations")
  parser.add_argument("--file", default=GCS_FILE_NAME, help="File name to read from mounted GCS")
  parser.add_argument("--project", default=PROJECT_ID, help="GCP project ID")
  parser.add_argument("--dataset", default=DATASET_ID, help="BigQuery dataset ID")
  parser.add_argument("--table", default=TABLE_ID, help="BigQuery table ID")

  args = parser.parse_args()

  mount_bucket(args.bucket, args.flags)

  file_path = os.path.join(MOUNT_DIR, args.file)
  timings = []

  for i in range(args.iterations):
    print(f"Reading file iteration {i + 1}...")
    start = time.time()
    read_file(file_path)
    duration = time.time() - start
    print(f"Iteration {i + 1}: {duration:.2f} seconds")
    timings.append({
        "timestamp": datetime.utcnow().isoformat(),
        "iteration": i + 1,
        "duration_seconds": duration,
        "file_path": file_path
    })

  df = pd.DataFrame(timings)
  log_to_bigquery(df, args.project, args.dataset, args.table)


if __name__ == "__main__":
  main()
