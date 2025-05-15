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
import time
import argparse
import helper

MOUNT_DIR = "gcs"
FILE_PREFIX = "testfile_write"

def create_files(num_files, file_size_in_gb, file_prefix="file"):
  """
  Creates a specified number of binary files with random data of a given size in GB.

  Args:
      num_files (int): Number of files to create.
      file_size_in_gb (float): Size of each file in GB (base 10).
      file_prefix (str): Prefix for the filenames.

  Returns:
      int | None: Total bytes written on success, None on failure.
  """
  total_bytes_written = 0
  file_size_in_bytes = int(file_size_in_gb * (1000 ** 3))

  for i in range(num_files):
    file_path = os.path.join(MOUNT_DIR, f"{file_prefix}_{file_size_in_gb}_{i}.bin")

    try:
      if os.path.exists(file_path):
        os.remove(file_path)
        print(f"{file_path} existed and was cleared.")

      with open(file_path, 'wb') as f:
        f.write(os.urandom(file_size_in_bytes))
        total_bytes_written += file_size_in_bytes

      print(f"Created {file_path} of size {file_size_in_gb:.4f} GB")

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
  parser.add_argument("--file-size-gb", type=float)
  args = parser.parse_args()

  workflow_type = "WRITE_{args.total_files}_{args.file_size_gb}GB_SINGLE_THREAD"
  helper.mount_bucket(MOUNT_DIR, args.bucket, args.gcsfuse_config)

  print(f"Starting write of {args.total_files} files...")
  start = time.time()
  try:
    total_bytes = create_files(args.total_files, args.file_size_gb, FILE_PREFIX)
  except RuntimeError as e:
    print(f"Failed during file write: {e}")
  duration = time.time() - start

  helper.unmount_gcs_directory(MOUNT_DIR)

  helper.log_to_bigquery(
      duration_sec=duration,
      total_bytes=total_bytes,
      gcsfuse_config=args.gcsfuse_config,
      workload_type=workflow_type,
  )

if __name__ == "__main__":
  main()
