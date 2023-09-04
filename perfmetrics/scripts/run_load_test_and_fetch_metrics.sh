#!/bin/bash
set -e
echo "Installing fio"
sudo apt-get install fio -y
echo "Installing pip"
sudo apt-get install pip -y
echo Print the time when FIO tests start
date
echo Running fio test..
echo "Overall fio start epoch time:" `date +%s`
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='fio-output.json'
echo "Overall fio end epoch time:" `date +%s`

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://periodic-perf-tests/creds.json gsheet
echo Fetching results..
# Upload data to the gsheet only when it runs through kokoro.
if [ "${KOKORO_JOB_TYPE}" != "RELEASE" ] && [ "${KOKORO_JOB_TYPE}" != "CONTINUOUS_INTEGRATION" ] && [ "${KOKORO_JOB_TYPE}" != "PRESUBMIT_GITHUB" ];
then
  python3 fetch_metrics.py fio-output.json
else
  python3 fetch_metrics.py fio-output.json --upload
fi

