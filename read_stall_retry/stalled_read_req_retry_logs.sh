#!/bin/bash

# Variables
cluster_name="xpk-large-scale-usc1b-a"
starttime="2025-01-29T17:00:00+05:30"
endtime="2025-01-30T05:00:00+05:30"
output_dir="$HOME/vipinydv-redstall-logs"

# Array of job names
job_names=("fastenvironment-readstall-genericread" "fastenvironment-readstall-filecache" "fastenvironment-readstall-paralleldownload")  # Add more job names as needed

# Ensure output directory exists
mkdir -p "$output_dir"
cd "$output_dir"

# Iterate through the job names and execute the `gcloud` logging command for each job
for job_name in "${job_names[@]}"; do
    echo "Fetching logs for job: $job_name"
    
    # Generate dynamic log file name by appending '-logs.csv' to the job name
    log_filename="$job_name-logs.csv"

    # Run gcloud logging read command and store logs in CSV format
    gcloud logging read "resource.labels.cluster_name=\"$cluster_name\" AND resource.labels.container_name=\"gke-gcsfuse-sidecar\" AND resource.labels.pod_name:\"$job_name\" AND timestamp>=\"$starttime\" AND timestamp<=\"$endtime\" AND \"stalled read-req\"" \
        --order=ASC \
        --format='csv(timestamp,textPayload)' > "$output_dir/$log_filename"
    
    echo "Logs for $job_name saved to $output_dir/$log_filename"
done

echo "Log fetching complete."
