#!/usr/bin/env python3
# Copyright 2026 Google LLC
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

import argparse
import os
import shutil
import subprocess
import sys
import tempfile
import uuid

def main():
  parser = argparse.ArgumentParser(description="GCSFuse Write IO Experiment Script")
  parser.add_argument("bucket_name", help="GCS Bucket Name")
  parser.add_argument("io_size", type=int, help="IO size in MiB")
  args = parser.parse_args()

  # Paths
  script_dir = os.path.dirname(os.path.abspath(__file__))
  repo_root = os.path.dirname(script_dir)
  log_file = os.path.join(script_dir, "gcsfuse.log")

  print(f"Building GCSFuse from repo root: {repo_root}")
  # Build GCSFuse binary
  build_res = subprocess.run(
      ["go", "build", "-o", os.path.join(repo_root, "gcsfuse"), "."],
      cwd=repo_root,
      capture_output=True,
      text=True
  )
  if build_res.returncode != 0:
    print(f"Failed to build GCSFuse:\n{build_res.stderr}", file=sys.stderr)
    sys.exit(1)
  
  gcsfuse_binary = os.path.join(repo_root, "gcsfuse")
  print("GCSFuse binary built successfully.")


  # Truncate/create the current log file to ensure it's clean for this run
  with open(log_file, "w") as f:
    f.truncate(0)

  # Create temp mount directory
  mount_dir = tempfile.mkdtemp(prefix="gcsfuse_mount_")
  print(f"Created temporary mount directory: {mount_dir}")

  try:
    print(f"Mounting bucket {args.bucket_name} to {mount_dir}")
    # Mount using built binary
    mount_cmd = [
        gcsfuse_binary,
        "--log-file", log_file,
        "--log-severity", "trace",
        args.bucket_name,
        mount_dir
    ]
    
    # Run mounting command
    mount_res = subprocess.run(mount_cmd, capture_output=True, text=True)
    if mount_res.returncode != 0:
      print(f"Failed to mount GCSFuse:\n{mount_res.stderr}", file=sys.stderr)
      sys.exit(1)
    
    print("Mount successful. Writing random file...")
    
    # Write random name file with provided IO size at once
    filename = f"exp_{uuid.uuid4().hex}.dat"
    filepath = os.path.join(mount_dir, filename)
    bytes_to_write = args.io_size * 1024 * 1024
    
    print(f"Writing {args.io_size} MiB to {filepath}")
    with open(filepath, "wb") as f:
      f.write(os.urandom(bytes_to_write))
      f.flush()
      os.fsync(f.fileno())
    
    print("Write complete.")

  finally:
    print(f"Cleaning up mount directory {mount_dir}")
    # Cleanup mount forcefully if needed
    unmount_res = subprocess.run(
        ["fusermount", "-uz", mount_dir],
        capture_output=True,
        text=True
    )
    if unmount_res.returncode != 0:
      print(f"Warning: fusermount returned {unmount_res.returncode}: {unmount_res.stderr}")
      # Try fallback umount -l if fusermount fails or is not available
      subprocess.run(["umount", "-l", mount_dir], capture_output=True)
    
    # Remove the directory
    if os.path.exists(mount_dir):
      try:
        shutil.rmtree(mount_dir)
        print("Temporary mount directory removed.")
      except Exception as e:
        print(f"Warning: Failed to remove temporary directory: {e}")

if __name__ == "__main__":
  main()
