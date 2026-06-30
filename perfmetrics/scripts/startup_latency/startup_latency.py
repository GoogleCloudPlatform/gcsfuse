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

"""Script to measure the startup latency of gcsfuse.

This script starts gcsfuse, waits until the mount point's device ID changes,
measures the elapsed time, and then unmounts gcsfuse.
"""

import argparse
import os
import subprocess
import sys
import time


def measure_startup_latency(gcsfuse_path, bucket_name, mount_point, extra_flags):
    # Ensure mount point exists
    if not os.path.exists(mount_point):
        os.makedirs(mount_point)

    # Check st_dev before mount
    try:
        stat_before = os.stat(mount_point)
        dev_before = stat_before.st_dev
    except Exception as e:
        print(f"Error stating mount point {mount_point} before mount: {e}", file=sys.stderr)
        return None

    # Construct command. We always enforce --foreground to keep process management clean
    # and avoid daemonization hangs in subprocess/wrapper environments.
    cmd = [gcsfuse_path]
    if "--foreground" not in extra_flags:
        cmd.append("--foreground")
    if extra_flags:
        cmd.extend(extra_flags)
    cmd.extend([bucket_name, mount_point])

    print(f"Starting gcsfuse with command: {' '.join(cmd)}")

    # Redirect stdout/stderr to a temporary log file to diagnose failures
    log_file_path = "gcsfuse_exec.log"
    log_file = None
    process = None
    mounted = False
    end_time = None

    try:
        log_file = open(log_file_path, "w")
        start_time = time.perf_counter()

        # Start gcsfuse process safely
        try:
            process = subprocess.Popen(cmd, stdout=log_file, stderr=log_file)
        except Exception as e:
            print(f"Error starting gcsfuse subprocess: {e}", file=sys.stderr)
            return None

        timeout = 15.0  # 15 seconds timeout
        while time.perf_counter() - start_time < timeout:
            try:
                stat_after = os.stat(mount_point)
                if stat_after.st_dev != dev_before:
                    end_time = time.perf_counter()
                    mounted = True
                    break
            except Exception:
                # Ignore stat errors while mounting is in progress
                pass
            # 1ms sleep to prevent 100% CPU usage
            time.sleep(0.001)
    finally:
        if log_file:
            log_file.close()

        # Always clean up by unmounting
        print("Unmounting...")
        try:
            subprocess.call(
                ["fusermount", "-u", mount_point], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
            )
        except Exception as e:
            print(f"Warning: fusermount failed with error: {e}", file=sys.stderr)

        # Safely wait for the process to exit with a timeout
        if process:
            try:
                # Give it a short timeout to exit after unmount
                process.wait(timeout=2)
            except subprocess.TimeoutExpired:
                # If it doesn't exit, terminate/kill it
                print("gcsfuse process did not exit after unmount. Terminating...", file=sys.stderr)
                process.terminate()
                try:
                    process.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    print("gcsfuse process did not terminate. Killing...", file=sys.stderr)
                    process.kill()
                    process.wait()

    if mounted and end_time is not None:
        latency_ms = (end_time - start_time) * 1000.0
        # Clean up log file on success
        if os.path.exists(log_file_path):
            os.remove(log_file_path)
        return latency_ms
    else:
        print("Timed out waiting for mount to be ready.", file=sys.stderr)
        # Print the log file contents to stdout for troubleshooting
        if os.path.exists(log_file_path):
            print("\n--- GCSFuse Logs ---", file=sys.stderr)
            try:
                with open(log_file_path, "r") as f:
                    print(f.read(), file=sys.stderr)
            except Exception as e:
                print(f"Error reading log file: {e}", file=sys.stderr)
            print("---------------------\n", file=sys.stderr)
            try:
                os.remove(log_file_path)
            except Exception:
                pass
        return None


def main():
    parser = argparse.ArgumentParser(description="Measure startup latency of gcsfuse.")
    parser.add_argument("--gcsfuse-path", default="./gcsfuse", help="Path to gcsfuse binary")
    parser.add_argument("--bucket-name", required=True, help="GCS bucket name to mount")
    parser.add_argument("--mount-point", default="./mnt", help="Directory to mount to")
    parser.add_argument("--flags", default="", help="Extra flags to pass to gcsfuse (space-separated)")

    args = parser.parse_args()

    extra_flags = args.flags.split() if args.flags else []

    # Make sure we use absolute path for mount point
    mount_point = os.path.abspath(args.mount_point)

    # Check if gcsfuse exists
    if not os.path.exists(args.gcsfuse_path):
        print(f"Error: gcsfuse binary not found at {args.gcsfuse_path}", file=sys.stderr)
        sys.exit(1)

    latency = measure_startup_latency(args.gcsfuse_path, args.bucket_name, mount_point, extra_flags)
    if latency is not None:
        print(f"Startup latency: {latency:.2f} ms")
    else:
        print("Failed to measure startup latency.")
        sys.exit(1)


if __name__ == "__main__":
    main()
