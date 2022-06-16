#!/bin/bash
set -e
echo Running fio test..
fio job_files/job_1.fio --output-format=json --output='output.json'
echo Installing requirements..
pip install -r requirements.txt --user
echo Adding pytest to PATH:
export PATH=/home/kbuilder/.local/bin:$PATH
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json ./gsheet
echo Running tests..
echo Fetching results..
python3 fetch_metrics.py output.json
