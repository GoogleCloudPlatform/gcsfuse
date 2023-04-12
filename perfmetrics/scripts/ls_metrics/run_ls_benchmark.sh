#!/bin/bash
set -e
# Refer for env var: https://stackoverflow.com/questions/72441758/typeerror-descriptors-cannot-not-be-created-directly
export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python
echo Installing pip and fuse..
sudo apt-get install fuse -y
sudo apt-get install pip -y
echo Installing requirements..
pip install -r  perfmetrics/scripts/ls_metrics/requirements.txt --user
echo Running script..
python3  perfmetrics/scripts/ls_metrics/listing_benchmark.py perfmetrics/scripts/ls_metrics/config.json --command "ls -R" --num_samples 30 --upload --message "Testing CT setup."
