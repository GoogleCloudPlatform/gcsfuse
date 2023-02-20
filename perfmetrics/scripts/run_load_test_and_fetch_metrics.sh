#!/bin/bash
set -e
echo Running fio test..
fio perfmetrics/scripts/job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo Logging fio results
cp output.json gcs/fio-logs/output-$(date '+%Y-%m-%d').json
python3 perfmetrics/scripts/utils/metrics_util.py gcs/fio-logs/ 10
echo Installing requirements..
pip install -r perfmetrics/scripts/requirements.txt --user
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json perfmetrics/scripts/gsheet
echo Fetching results..
python3 perfmetrics/scripts/fetch_metrics.py output.json
