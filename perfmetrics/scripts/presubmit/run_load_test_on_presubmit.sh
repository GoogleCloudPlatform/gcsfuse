#!/bin/bash
set -e
echo Running fio test..
fio ./perfmetrics/scripts/job_files/presubmit_perf_test.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo fetching results..
python3 ./perfmetrics/scripts/presubmit/fetch_results.py output.json
