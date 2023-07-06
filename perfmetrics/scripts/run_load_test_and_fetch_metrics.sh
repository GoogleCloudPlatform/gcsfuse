#!/bin/bash
set -e
echo "Mounting gcs bucket"
mkdir -p gcs
GCSFUSE_FLAGS=$1
UPLOAD_FLAGS=$2
BUCKET_NAME="periodic-perf-tests${EXPERIMENT_NUMBER}"
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

echo Print the time when FIO tests start
date
echo Running fio test..
echo "Overall fio start epoch time:" `date +%s`
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output="fio-output${EXPERIMENT_NUMBER}.json"
echo "Overall fio end epoch time:" `date +%s`
sudo umount $MOUNT_POINT

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
if [ -z "$EXPERIMENT_NUMBER" ]; then
  gsutil cp gs://periodic-perf-tests/creds.json gsheet
else
  gsutil cp gs://periodic-perf-tests1/creds.json gsheet
fi

echo Fetching results..
python3 fetch_metrics.py fio-output.json $UPLOAD_FLAGS
