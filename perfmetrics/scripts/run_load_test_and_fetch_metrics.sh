#!/bin/bash
set -e
echo Running fio test..
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo Installing requirements..
pip install -r requirements.txt --user
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json ./gsheet
echo Fetching results..
python3 fetch_metrics.py output.json
