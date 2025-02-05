#!/bin/bash

# Define the repository URL and the target directory
REPO_URL="https://github.com/GoogleCloudPlatform/gcsfuse-tools.git"
TARGET_DIR="$HOME/gcsfuse-tools/ssiog"

# Define the bucket name and list of job names
BUCKET_NAME="vipinydv-metrics"
JOB_NAMES=("fastenvironment-genericread-1byte" "fastenvironment-readstall-genericread-1byte" "fastenvironment-filecache-1byte" "fastenvironment-readstall-filecache-1byte" "fastenvironment-paralleldownload-1byte" "fastenvironment-readstall-paralleldownload-1byte" "slowenvironment-genericread-1byte" "slowenvironment-readstall-genericread-1byte" "slowenvironment-filecache-1byte" "slowenvironment-readstall-filecache-1byte" "slowenvironment-paralleldownload-1byte" "slowenvironment-readstall-paralleldownload-1byte")

# Clone the repository to the home directory
echo "Cloning repository..."
git clone "$REPO_URL" "$HOME/gcsfuse-tools"

# Navigate to the specified directory
cd "$TARGET_DIR" || { echo "Directory not found: $TARGET_DIR"; exit 1; }

# Loop over each job name and run the python command
for JOB in "${JOB_NAMES[@]}"; do
    echo "Running metrics collector for job: $JOB"
    python3 metrics_collector.py --metrics-path "gs://$BUCKET_NAME/$JOB/*.csv"
done

echo "Finished processing all jobs."
