#!/bin/bash
set -e
echo Print the time when FIO tests start
date
echo Running fio test..
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo Logging fio results
cp output.json gcs/fio-logs/output-$(date '+%Y-%m-%d').json
python3 utils/metrics_util.py gcs/fio-logs/ 10
echo Installing requirements..
pip install -r requirements.txt --user
gsutil cp gs://gcs-fuse-dashboard-fio/creds.json gsheet
echo Fetching results..
# Upload data to the gsheet only when it runs through kokoro.
if [ "${KOKORO_JOB_TYPE}" != "RELEASE" ] && [ "${KOKORO_JOB_TYPE}" != "CONTINUOUS_INTEGRATION" ] && [ "${KOKORO_JOB_TYPE}" != "PRESUBMIT_GITHUB" ];
then
  python3 fetch_metrics.py output.json
else
  python3 fetch_metrics.py output.json --upload
fi
