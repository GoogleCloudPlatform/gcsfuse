#!/bin/bash

#To run the script
#>> ./populate_metrics.sh <start_time> <end_time>

set -e

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://periodic-perf-tests/creds.json ./gsheet
echo Fetching results..
python3 populate_vm_metrics.py $1 $2
