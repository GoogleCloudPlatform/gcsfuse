#!/bin/bash
set -e
echo Running fio test..
fio job_files/presubmit_perf_test.fio --lat_percentiles 1 --output-format=json --output='output.json'

echo showing results..
python3 print_results.py ../output.json
