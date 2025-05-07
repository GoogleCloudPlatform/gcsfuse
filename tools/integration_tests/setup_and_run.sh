#!/bin/bash

set -euo pipefail

bash "$PWD/tools/integration_tests/run_e2e_tests.sh" false true "us-central1" false false false

# --- Configuration ---
ENV_NAME="$HOME/venv" # Name of the virtual environment directory
rm -rf "$ENV_NAME"
REQUIREMENTS_FILE="$PWD/tools/integration_tests/requirements.txt" # Name of the requirements file

# --- Script Logic ---
echo "--- Python Environment Setup Script ---"

# Check if requirements file exists
if [ ! -f "$REQUIREMENTS_FILE" ]; then
    echo "Error: '$REQUIREMENTS_FILE' not found in the current directory."
    echo "Please create a '$REQUIREMENTS_FILE' file with your dependencies."
    exit 1
fi

# Create a virtual environment
echo "Creating virtual environment '$ENV_NAME'..."
# Use python3 -m venv for modern Python versions
python3 -m venv "$ENV_NAME"

# Check if venv creation was successful
if [ $? -ne 0 ]; then
    echo "Error: Failed to create virtual environment."
    exit 1
fi

echo "Virtual environment created."

# Activate the virtual environment
echo "Activating virtual environment..."
# Source the activate script - use the correct path for your OS
# For Linux/macOS:
if [ -f "$ENV_NAME/bin/activate" ]; then
    source "$ENV_NAME/bin/activate"
    echo "Virtual environment activated."
else
    echo "Error: Could not find activate script."
    exit 1
fi

# Check if activation was successful (optional, but good practice)
if [[ "$VIRTUAL_ENV" == *"$ENV_NAME"* ]]; then
    echo "Activation confirmed."
else
    echo "Warning: Activation may not have been fully successful."
fi


# Install requirements
echo "Installing dependencies from '$REQUIREMENTS_FILE'..."
pip install -r "$REQUIREMENTS_FILE"

# Check if installation was successful
if [ $? -ne 0 ]; then
    echo "Error: Failed to install dependencies."
    echo "Please check your '$REQUIREMENTS_FILE' and internet connection."
    # Note: We don't exit here immediately, allowing sample commands to potentially run
    # if some packages were installed, but it's often better to exit on install failure.
    # For this example, we'll continue but print a warning.
    echo "Continuing script execution, but installation failed."
fi


# Run command in the activated environment
echo "--- Running Gantt Generator ---"
python3 "$PWD/tools/integration_tests/gantt_generator.py"
echo "--- Execution Logic ---"

deactivate
if [ $? -ne 0 ]; then
    echo "Virtual Env Deactivated"
fi