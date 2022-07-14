#!/bin/bash
set -e
echo Running fio test..
fio job_files/job_6.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo Installing requirements..
pip install -r requirements.txt --user
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json ./gsheet
echo Fetching results..
python3 fetch_metrics.py output.json
