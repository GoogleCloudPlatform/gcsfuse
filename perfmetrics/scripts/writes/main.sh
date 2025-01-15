# Expects GCSFuse, Golang, Git, Fio installed.
# Expects GCSFuse repo cloned and checked out in $HOME/github/gcsfuse directory
set -e

export CGO_ENABLED=0
export KOKORO_ARTIFACTS_DIR=$HOME
BUCKET_NAME="ashmeen-streaming-writes"
SPREADSHEET_ID="1TD6ds2ipkRsU6megYO2AEXHYGeh5phLWjFPKnTc7A88"
mkdir -p $HOME/gcs
MOUNT_POINT=$HOME/gcs

config_length=$(yq '.config | length' input.yaml)
for i in $(seq 0 $((config_length - 1))); do
    # Extract values using yq
    fio_directory=$(yq '.config['$i'].fio_directory' input.yaml)
    fio_directory=${fio_directory//[\"\']}
    fio_block_size=$(yq '.config['$i'].fio_block_size' input.yaml)
    fio_block_size=${fio_block_size//[\"\']}
    fio_file_size=$(yq '.config['$i'].fio_file_size' input.yaml)
    fio_file_size=${fio_file_size//[\"\']}
    gcsfuse_flags=$(yq '.config['$i'].gcsfuse_flags' input.yaml)
    gcsfuse_flags=${gcsfuse_flags//[\"\']}

    # Inner loop for experiments
    for j in {1..3}; do
        experiment_dir="${fio_directory}-exp${j}"
        echo "Mounting gcs bucket"
        gcsfuse $gcsfuse_flags --log-severity=TRACE --log-file=$experiment_dir.log $BUCKET_NAME $MOUNT_POINT

        # Run FIO test...
        echo Running fio test..
        rm -r "${MOUNT_POINT:?}/$experiment_dir" || true
        mkdir -p "${MOUNT_POINT:?}/$experiment_dir"
        fio job.fio --bs=$fio_block_size --directory=$MOUNT_POINT/$experiment_dir --filesize=$fio_file_size --lat_percentiles 1 --output-format=json --output="fio-output.json"

        echo Installing requirements..
        pip install --require-hashes -r ../requirements.txt
        gsutil cp gs://ashmeen-perf-tests/creds.json gsheet/creds.json
        echo Fetching results to sheet..
        START_TIME_BUILD=$(date +%s)
        UPLOAD_FLAGS="--upload_gs --start_time_build $START_TIME_BUILD --spreadsheet_id=$SPREADSHEET_ID"
        python3 "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/fetch_and_upload_metrics.py" "fio-output.json" $UPLOAD_FLAGS

        python3 get_time_from_logs.py $experiment_dir.log $SPREADSHEET_ID
        echo Cleaning up...
        rm fio-output.json
        rm "$experiment_dir".log
        rm "$experiment_dir".log.stderr

        fusermount -uz "$MOUNT_POINT"
        sleep 5s
    done
done