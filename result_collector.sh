#!/bin/bash
# Copyright 2025 Google LLC

# 1. Get the absolute path to the directory where this script lives
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

# 2. Define the output folder relative to the script
OUTPUT_DIR="$SCRIPT_DIR/output"

# 3. CLEANING STEP: Remove the folder if it exists, then recreate it
echo "Cleaning local output directory: $OUTPUT_DIR"
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# 4. Define your GCS variables
BUCKET_NAME="gcsfuse-release-packages"
# This prefix matches everything starting with 'mky-release-test' inside the version folder
PREFIX="v999.999.987/mky-release-test" 

echo "Output dir is ${OUTPUT_DIR}"
echo "Syncing folders from gs://$BUCKET_NAME/$PREFIX* to $OUTPUT_DIR..."

# 5. Use gcloud storage cp with the recursive flag and wildcard
# The wildcard after the prefix ensures it grabs all the sub-folders shown in your screenshot
gcloud storage cp -r "gs://$BUCKET_NAME/$PREFIX*" "$OUTPUT_DIR"


echo "Cleanup complete. Files are located in: $OUTPUT_DIR"

# ==========================================
# 6. PYTHON RUNNER: Create venv and analyze
# ==========================================
# CHANGE THIS PATH TO POINT TO YOUR ACTUAL SCRIPT
PYTHON_SCRIPT1_PATH="analyze_logs.py"
PYTHON_SCRIPT2_PATH="package_bucket_analyzer.py"

echo "Setting up Python virtual environment in /tmp..."
VENV_DIR="/tmp/gcsfuse_log_venv"

# Create a clean virtual environment
rm -rf "$VENV_DIR"
python3 -m venv "$VENV_DIR"

# Activate the virtual environment
source "$VENV_DIR/bin/activate"

echo "Running analysis using script at: $PYTHON_SCRIPT_PATH"
echo  > "summary.txt"
# Run your existing python script, passing the output directory as the argument
python3 "$PYTHON_SCRIPT1_PATH" "$OUTPUT_DIR" >> "summary.txt"

python3 "$PYTHON_SCRIPT2_PATH" "$OUTPUT_DIR" >> "summary.txt"

# Deactivate the virtual environment
deactivate
echo "------------------------------------------------------"
echo "Analysis complete. Virtual environment deactivated."