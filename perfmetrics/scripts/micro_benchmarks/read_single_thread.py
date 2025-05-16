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
import sys
import subprocess
import time
import argparse
from google.cloud import storage
import helper

MOUNT_DIR = "gcs"
FILE_PREFIX = "testfile_read"

def check_and_create_files(bucket_name: str, total_files: int, file_size_gb: int):
  """
  Ensures that the specified number of files exist in the given GCS bucket.
  If a file is missing or its not given size, it is (re)created and uploaded.

  Args:
      bucket_name (str): Name of the GCS bucket.
      total_files (int): Number of files to check or create.
      file_size_gb (int): Expected size of each file in gigabytes (GB, base-10).
  """
  client = storage.Client()
  bucket = client.get_bucket(bucket_name)
  expected_size = file_size_gb * (10 ** 9)  # Use base-10 GB

  print(f"Ensuring all {total_files} files exist in gs://{bucket_name}...")

  for i in range(total_files):
    fname = f"{FILE_PREFIX}_{file_size_gb}_{i}.bin"
    blob = bucket.blob(fname)

    blob_exists = blob.exists()
    # Use a default value for blob.size if it's None (e.g., if blob doesn't exist)
    if blob_exists:
      blob.reload()
    current_blob_size = blob.size if blob_exists and blob.size is not None else 0

    if not blob_exists or current_blob_size != expected_size:
      if blob_exists:
        print(f"{fname} exists but has size {current_blob_size} bytes (expected ~{expected_size}). Re-uploading...")

      print(f" Creating {file_size_gb}GB dummy file {fname}...")
      local_path = f"/tmp/{fname}"

      try:
        # Use subprocess.run to execute fallocate
        gb_in_bytes = file_size_gb * 10**9
        subprocess.run(f"fallocate -l {gb_in_bytes} {local_path}", shell=True, check=True)

      except subprocess.CalledProcessError as e:
        print(f"Error creating dummy file {local_path}: {e}")
        continue # Skip to the next file if creation fails

      try:
        blob.upload_from_filename(local_path)
        print(f"Uploaded {fname} to gs://{bucket_name}/{fname}")
      except Exception as e: # Catch broader exceptions for upload issues
        print(f"Error uploading {fname} to GCS: {e}")
      finally:
        # Ensure local file is removed even if upload fails
        if os.path.exists(local_path):
          os.remove(local_path)
    else:
      print(f"{fname} already exists with acceptable size.")

def read_all_files(total_files: int, file_size_gb: int) -> int:
  """
   Reads a specified number of files, calculates, and returns the total number
   of bytes read across all files.

   The files are expected to be named with a common prefix and index suffix:
   {FILE_PREFIX}_{file_size_gb}_{i}.bin, located inside the directory MOUNT_DIR.

   Args:
       total_files (int): The number of files to read.
       file_size_gb (int): file size in gb
   Returns:
       int: The total number of bytes read from all files.

   Raises:
       RuntimeError: If any error occurs while reading any file, including
                     file not found, permission issues, or other IO errors.
   """
  total_bytes = 0
  for i in range(total_files):
    path = os.path.join(MOUNT_DIR, f"{FILE_PREFIX}_{file_size_gb}_{i}.bin")
    try:
      with open(path, "rb") as f:
        total_bytes += len(f.read())
    except Exception as e:  # catch all exceptions
      print(f"Error reading file {path}: {e}")
      raise RuntimeError(f"Failed to read file: {path}") from e

  return total_bytes

def main():
  parser = argparse.ArgumentParser(description="Measure GCS read bandwidth via gcsfuse.")
  parser.add_argument("--bucket", required=True, help="GCS bucket name")
  parser.add_argument("--gcsfuse-config", default="--implicit-dirs", help="GCSFuse mount flags")
  parser.add_argument("--total-files", type=int, default=10, help="Number of files to read")
  parser.add_argument("--file-size-gb", type=int, default=15, help="Size of each file in GB")

  workflow_type = "READ_{args.total_files}_{args.file_size_gb}GB_SINGLE_THREAD"
  args = parser.parse_args()

  # Mount the bucket
  helper.mount_bucket(MOUNT_DIR, args.bucket, args.gcsfuse_config)

  # Ensure test files exist
  check_and_create_files(args.bucket, args.total_files, args.file_size_gb)

  print(f"Starting read of {args.total_files} files...")
  start = time.time()
  try:
    total_bytes = read_all_files(args.total_files, args.file_size_gb)
    print(f"Total bytes read from {args.total_files} files: {total_bytes}")
  except RuntimeError as e:
    print(f"Failed during file read: {e}")
    helper.unmount_gcs_directory(MOUNT_DIR)
    sys.exit(1)  # Exit with error status
  duration = time.time() - start

  # Unmount after test
  helper.unmount_gcs_directory(MOUNT_DIR)

  # Log to BigQuery
  helper.log_to_bigquery(
      duration_sec=duration,
      total_bytes=total_bytes,
      gcsfuse_config=args.gcsfuse_config,
      workload_type=workflow_type,
  )

if __name__ == "__main__":
  main()
