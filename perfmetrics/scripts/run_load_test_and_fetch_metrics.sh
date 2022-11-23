#!/bin/bash
set -e
echo Running fio test..
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo Logging fio results
<<<<<<< HEAD
cp output.json gcs/Fio-logs/output-$(date '+%Y-%m-%d').json
python3 utils/metrics_util.py gcs/Fio-logs/ 10
=======
cp output.json gcs/fio-logs/output-$(date '+%Y-%m-%d').json
python3 utils/metrics_util.py gcs/fio-logs/ 10
>>>>>>> 5bd63dc73957a617d8179e1016d74c111158ceed
echo Installing requirements..
pip install -r requirements.txt --user
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json ./gsheet
echo Fetching results..
python3 fetch_metrics.py output.json
