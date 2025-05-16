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
import argparse
import time
import helper
import sys

MOUNT_DIR = "gcs"
FILE_PREFIX = "testfile"

def delete_existing_file(file_path):
  """
  Deletes the file at the specified path if it exists.

  Args:
      file_path (str): The full path to the file to delete.

  Returns:
      bool: True if file deleted or didn't exist; False on error.
  """
  try:
    if os.path.exists(file_path):
      os.remove(file_path)
      print(f"{file_path} existed and was cleared.")
    return True
  except Exception as e:
    print(f"Error deleting file {file_path}: {e}")
    return False

def write_random_file(file_path, file_size_in_bytes):
  """
  Creates a binary file filled with random data of the specified size.

  Args:
      file_path (str): The full path where the file should be created.
      file_size_in_bytes (int): The size of the file in bytes.

  Returns:
      bool: True on success; False on failure.
  """
  try:
    with open(file_path, 'wb') as f:
      f.write(os.urandom(file_size_in_bytes))
    print(f"Created {file_path} of size {file_size_in_bytes / (1000 ** 3):.4f} GB")
    return True
  except Exception as e:
    print(f"Error writing file {file_path}: {e}")
    return False

def create_files(file_paths, file_size_in_gb):
  """
  Writes random data to specified file paths, each of the given size in GB.

  Args:
      file_paths (list[str]): List of file paths to create.
      file_size_in_gb (float): Size of each file in GB (base 10).

  Returns:
      int | None: Total bytes written on success, None on failure.
  """
  total_bytes_written = 0
  file_size_in_bytes = int(file_size_in_gb * (1000 ** 3))

  for file_path in file_paths:
    try:
      success = write_random_file(file_path, file_size_in_bytes)
      if not success:
        print("Write failed. Exiting.")
        sys.exit(1)
      total_bytes_written += file_size_in_bytes
    except Exception as e:
      print(f"Error creating file {file_path}: {e}")
      return None

  print(f"Total bytes written: {total_bytes_written / (1000**3):.4f} GB")
  return total_bytes_written

def main():
  parser = argparse.ArgumentParser(description="Measure GCS write bandwidth via gcsfuse.")
  parser.add_argument("--bucket", required=True)
  parser.add_argument("--gcsfuse-config", default="--implicit-dirs")
  parser.add_argument("--total-files", type=int, default=1)
  parser.add_argument("--file-size-gb", type=int, default=15, help="Size of each file in GB")
  args = parser.parse_args()

  workflow_type = f"WRITE_{args.total_files}_{args.file_size_gb}GB_SINGLE_THREAD"
  helper.mount_bucket(MOUNT_DIR, args.bucket, args.gcsfuse_config)

  # Prepare file paths
  file_paths = [
      os.path.join(MOUNT_DIR, f"{FILE_PREFIX}_{args.file_size_gb}_{i}.bin")
      for i in range(args.total_files)
  ]

  # Delete files if they already exist
  for path in file_paths:
    success = delete_existing_file(path)
    if not success:
      print("Delete failed. Exiting.")
      sys.exit(1)

  print(f"Starting write of {args.total_files} files...")
  start = time.time()
  try:
    total_bytes = create_files(file_paths, args.file_size_gb)
  except RuntimeError as e:
    print(f"Failed during file write: {e}")
    helper.unmount_gcs_directory(MOUNT_DIR)
    sys.exit(1)  # Exit with error status
  duration = time.time() - start

  helper.unmount_gcs_directory(MOUNT_DIR)

  helper.log_to_bigquery(
     start_time_sec=start,
      duration_sec=duration,
      total_bytes=total_bytes,
      gcsfuse_config=args.gcsfuse_config,
      workload_type=workflow_type,
  )

if __name__ == "__main__":
  main()
