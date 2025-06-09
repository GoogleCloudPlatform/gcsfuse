#!/bin/bash
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

# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

# This script is used to create runtime tables for integration tests given
# package stats entries in file in the below format:
# `package_name bucket_type exit_code start_time end_time`
usage() {
    echo "Usage: $0 <FILE_PATH>"
    exit 1
}

if [ "$#" -ne 1 ]; then
    log_error "Missing required arguments."
    usage
fi

PACKAGE_RUNTIME_STATS=$1

if [ ! -f "$PACKAGE_RUNTIME_STATS" ]; then
    echo "Error: File '$PACKAGE_RUNTIME_STATS' not found."
    exit 1
fi

# sort pakages in ascending order.
sort -o "$PACKAGE_RUNTIME_STATS" "$PACKAGE_RUNTIME_STATS"



# Print single package stats
print_package_stats() {
    local package_name="$1"
    local bucket_type="$2"
    local exit_code="$3"
    local start_sec="$4"
    local end_sec="$5"
    local wait_min run_min status
    if [ "$exit_code" -eq 0 ]; then
        status="PASSED"
    else
        status="FAILED"
    fi
    wait_min=$((start_sec / 60))
    run_min=$(((end_sec - start_sec + 60) / 60))
    package_stats=$(printf "| %-25s | %-15s | %-8s | %-10s |%-60s|\n" \
        "$package_name" \
        "$bucket_type" \
        "$status" \
        "${run_min}m" \
        "$(printf '%0.s_' $(seq 1 "$wait_min"))$(printf '%0.s>' $(seq 1 "$run_min"))")
    echo "$package_stats"
}

# Display legends
echo ""
echo "Timings for the e2e test packages run are listed below."
echo "_ is 1 min wait"
echo "> is 1 min run"
# Add Table headers
echo "+---------------------------+-----------------+----------+------------+------------------------------------------------------------+"
echo "| Package Name              | Bucket Type     | Status   | Total Time |0 min                      runtime                    60 min|"
# Read the file line by line and print stats.
while IFS= read -r line || [[ -n "$line" ]]; do # Process even if last line has no newline
    echo "+---------------------------+-----------------+----------+------------+------------------------------------------------------------+"
    print_package_stats $line
done <"$PACKAGE_RUNTIME_STATS"
echo "+---------------------------+-----------------+----------+------------+------------------------------------------------------------+"
echo ""
