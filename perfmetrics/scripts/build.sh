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
#pytest gsheet/gsheet_test.py
pytest vmmetrics/vmmetrics_test.py
echo Fetching results..
python3 execute_codes.py output.json
