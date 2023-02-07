#!/bin/bash
set -e
echo Running fio test..
fio job_files/seq_rand_read_write_presubmit.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo showing results..
python3 show_results.py output.json
