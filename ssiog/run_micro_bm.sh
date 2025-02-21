#!/bin/bash

set -x

# Set the Python interpreter to use (optional, but recommended)
PYTHON_INTERPRETER="python3.9"

# Set the path to your Python script
SCRIPT_PATH="training.py"  # Replace with the actual path

# Set any command-line arguments you want to pass to the script
ARGS=(
  "--prefix" "~/gcs/12G"
  "--epochs" "1"
  "--steps" "10"
  "--sample-size" "131072"
  "--batch-size" "98304"
  "--log-level" "INFO"
 "--background-threads" "32"
#  "--group-size" "2"
  "--read-order" "FullRandom"
  # "--exporter-type" "cloud"
  "--exporter-type" "console"
)

# Invoke the Python script with the specified interpreter and arguments
$PYTHON_INTERPRETER "$SCRIPT_PATH" "${ARGS[@]}"