#!/bin/bash

set -x

# Set the Python interpreter to use (optional, but recommended)
PYTHON_INTERPRETER="python3"

# Set the path to your Python script
SCRIPT_PATH="training.py"  # Replace with the actual path

# Set any command-line arguments you want to pass to the script
ARGS=(
  "--prefix" "/usr/local/google/home/princer/gcs/"
  "--epochs" "4"
  "--steps" "6"
  "--sample-size" "1048576"
  "--batch-size" "20"
  "--log-level" "INFO"
 "--background-threads" "8"
#  "--group-size" "2"
  "--read-order" "FullRandom"
  # "--exporter-type" "cloud"
  "--exporter-type" "console"
)

# Invoke the Python script with the specified interpreter and arguments
$PYTHON_INTERPRETER "$SCRIPT_PATH" "${ARGS[@]}"