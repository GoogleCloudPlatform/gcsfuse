#!/bin/bash
set -e
# Installing requirements
echo Installing python3-pip
sudo apt-get -y install python3-pip
echo Installing libraries to run python script
pip install google-cloud
pip install google-cloud-vision
pip install google-api-python-client
pip install prettytable
echo Installing fio
sudo apt-get install fio -y

echo Running fio test..
fio ./perfmetrics/scripts/job_files/presubmit_perf_test.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo fetching results..
python3 ./perfmetrics/scripts/presubmit/fetch_results.py output.json
