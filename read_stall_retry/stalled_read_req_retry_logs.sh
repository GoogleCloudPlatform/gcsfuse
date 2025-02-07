#!/bin/bash

# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Variables
cluster_name="xpk-large-scale-usc1f-a"
starttime="2025-02-04T18:00:00+05:30"
endtime="2025-02-05T10:00:00+05:30"
output_dir="$HOME/readstall-logs"

# Array of job names
job_names=("slowenvironment-readstall-genericread" "slowenvironment-readstall-filecache" "slowenvironment-readstall-paralleldownload")

# Ensure output directory exists
mkdir -p "$output_dir"
cd "$output_dir"

# Convert starttime and endtime to Unix timestamps
start_timestamp=$(date -d "$starttime" +%s)
end_timestamp=$(date -d "$endtime" +%s)

# Iterate through the job names
for job_name in "${job_names[@]}"; do
    echo "Fetching logs for job: $job_name"
    
    # Generate dynamic log file name by appending '-logs.csv' to the job name
    log_filename="$job_name-logs.csv"
    
    # Initialize the timestamp for the current interval
    current_start_time=$start_timestamp

    # Loop through the time range and divide it into 30-minute intervals
    while [ $current_start_time -lt $end_timestamp ]; do
        # Calculate the end time for the current half-hour interval
        current_end_time=$((current_start_time + 1800))  # 1800 seconds = 30 minutes
        
        # Convert the timestamps back to the desired format (ISO 8601)
        start_time_formatted=$(date -d @$current_start_time --utc +%FT%T%:z)
        end_time_formatted=$(date -d @$current_end_time --utc +%FT%T%:z)

        # Run gcloud logging read command and append logs for the current half-hour interval
        echo "Fetching logs for interval: $start_time_formatted to $end_time_formatted"
        
        gcloud logging read "resource.labels.cluster_name=\"$cluster_name\" AND resource.labels.container_name=\"gke-gcsfuse-sidecar\" AND resource.labels.pod_name:\"$job_name\" AND timestamp>=\"$start_time_formatted\" AND timestamp<=\"$end_time_formatted\" AND \"stalled read-req\"" --order=ASC --format='csv(timestamp,textPayload)' >> "$output_dir/$log_filename"

        # Move to the next half-hour interval
        current_start_time=$current_end_time
    done
    
    echo "Logs for $job_name saved to $output_dir/$log_filename"
done

echo "Log fetching complete."
