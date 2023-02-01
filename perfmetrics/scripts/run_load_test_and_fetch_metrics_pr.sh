#!/bin/bash
set -e
echo Running fio test..
fio job_files/seq_rand_read_write_pr.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo Logging fio results
cp output.json gcs/fio-logs/output-$(date '+%Y-%m-%d').json
python3 utils/metrics_util.py gcs/fio-logs/ 10
echo Installing requirements..
pip install -r requirements.txt --user
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json ./gsheet
echo Fetching results..
python3 fetch_metrics.py output.json
