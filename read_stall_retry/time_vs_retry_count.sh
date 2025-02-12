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

# Function to print messages with timestamps to the terminal
log_message() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Change to the ~/vipinydv-readstall-logs directory
cd ~/readstall-logs || { log_message "Directory ~/readstall-logs does not exist"; exit 1; }
log_message "Changed to directory ~/readstall-logs"

# Directory to store the output CSV files
output_dir="time_vs_retry_count"

# Create the directory if it doesn't exist
mkdir -p "$output_dir"
log_message "Created output directory $output_dir"

# Input CSV files (list the CSV files here)
csv_files=("slowenvironment-readstall-genericread.csv" "fastenvironment-readstall-filecache.csv" "fastenvironment-readstall-paralleldownload.csv")

# Process each CSV file
for file in "${csv_files[@]}"; do
    # Log file being processed
    log_message "Processing file $file"

    # Check if the file exists
    if [[ ! -f "$file" ]]; then
        log_message "File $file does not exist, skipping."
        echo "File $file does not exist, skipping."
        continue
    fi

    # Create an associative array to store the frequency of requests per hour:minute
    declare -A request_count

    # Read the CSV file line by line
    while IFS=, read -r timestamp message; do
        # Extract hour and minute directly using parameter expansion
        hour_minute="${timestamp:11:5}"  # Get the hour and minute (characters 11 to 15 in the format "HH:MM")
        
        # Log if the hour_minute value is empty or malformed
        if [[ -z "$hour_minute" || ! "$hour_minute" =~ ^[0-9]{2}:[0-9]{2}$ ]]; then
            log_message "Invalid hour:minute value found for timestamp: $timestamp"
            continue
        fi
        
        # Increment the request count for that hour:minute
        ((request_count["$hour_minute"]++))
    done < "$file"

    # Prepare the output CSV file
    output_file="$output_dir/$(basename "$file" .csv)_time_vs_retry_count.csv"

    # Log the output file being written
    log_message "Writing output to $output_file"

    # Collect all output into a variable
    output=""
    output+="hour:minute,request_count\n"

    # Generate the request count for each hour:minute
    for hour_minute in "${!request_count[@]}"; do
        output+="$hour_minute,${request_count[$hour_minute]}\n"
    done

    # Write the collected output to the file in one go
    echo -e "$output" > "$output_file"

    # Log after the file has been processed
    log_message "Processed $file and stored results in $output_file"
done

log_message "Script execution completed."
