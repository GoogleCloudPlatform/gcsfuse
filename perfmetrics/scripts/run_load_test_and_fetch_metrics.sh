#!/bin/bash
set -e
echo Print the time when FIO tests start
date
echo Running fio test..
fio perfmetrics/scripts/job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='fio-output.json'
echo Logging fio results
echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://periodic-perf-tests/creds.json gsheet
echo Fetching results..
