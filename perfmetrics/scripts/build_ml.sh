#!/bin/bash

#To run the script
#>> ./build.sh <start_time> <end_time>

set -e

echo Installing requirements..
pip install -r requirements.txt --user
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json ./gsheet
echo Fetching results..
python3 fetch_metrics_ml.py $1 $2
