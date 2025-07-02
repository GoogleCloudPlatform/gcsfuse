#!/bin/bash

# Define variables
BUCKET_NAME="princer-ckpt"
MOUNT_POINT="/home/princer_google_com/bucket"
LOG_FILE="/home/princer_google_com/logs.txt"
DATA_FILE="/home/princer_google_com/bucket/data10G.txt"
PYTHON_SCRIPT="/home/princer_google_com/dev/gcsfuse/read.py"

# Unmount GCS bucket if it's mounted
echo "Unmounting GCS bucket: $MOUNT_POINT"
fusermount -u "$MOUNT_POINT"

# Create mount point if it doesn't exist
mkdir -p "$MOUNT_POINT"

# Truncate log-file.
echo "" >"$LOG_FILE"

# Mount GCS bucket using gcsfuse
echo "Mounting GCS bucket: gs://$BUCKET_NAME to $MOUNT_POINT"
gcsfuse --enable-buffered-read --read-block-size-mb=16 --read-max-blocks-per-handle=20 --read-global-max-blocks=100 --log-file "$LOG_FILE" --log-severity info --log-format text "$BUCKET_NAME" "$MOUNT_POINT"

# Check if mount was successful
if [ $? -eq 0 ]; then
    echo "GCS bucket mounted successfully."

    #   echo "Generating 10GiB file: $DATA_FILE"
    #   dd if=/dev/zero of="$DATA_FILE" bs=1M count=10240

    # echo "Executing Python script: $PYTHON_SCRIPT"
    # time python3 "$PYTHON_SCRIPT"

    echo "Reading 10GiB using dd command:"
    time dd if="$DATA_FILE" of=/dev/null bs=1M count=10240 iflag=direct

    # Unmount GCS bucket
    echo "Unmounting GCS bucket: $MOUNT_POINT"
    fusermount -u "$MOUNT_POINT"

    # Check if unmount was successful
    if [ $? -eq 0 ]; then
        echo "GCS bucket unmounted successfully."
    else
        echo "Error unmounting GCS bucket."
    fi

else
    echo "Error mounting GCS bucket."
fi
