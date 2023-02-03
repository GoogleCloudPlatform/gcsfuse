#!/bin/bash
set -e
echo Running fio test..
fio job_files/seq_rand_read_write_pr.fio --lat_percentiles 1 --output-format=json --output='output_pr.json'
echo Logging fio results
cp output_pr.json gcs/fio-logs/output-$(date '+%Y-%m-%d').json
echo Fetching results..
python3 fetch_metrics_pr.py output_pr.json
