#!/bin/bash

# --- Configuration ---

# 1. GCS and Mount Configuration
# Replace with your actual bucket name
GCS_BUCKET="thrivikramks_fio_test_us_west4"
# Local directory to use as the mount point
MOUNT_POINT="/tmp/mnt"

# 2. Fio Job Configuration
# Path to your Fio job file (e.g., 'randrw_job.fio')
# NOTE: Ensure your job file uses the '$BS' variable for block size.
FIO_JOB_FILE="$HOME/tmp/gcsfuse_sample_job_spec_randread.fio"

# List of block sizes (bs) to test
BLOCK_SIZES=("1M" "10M" "20M" "30M" "50M" "70M" "100M" "120M" "150M")

# Environment Variables (Optional, for advanced gcsfuse/fio logging or tuning)
# These variables will be passed to the environment before running fio.
# Example GCSFUSE debug flag (set to 'true' to enable gcsfuse debugging)
export GCSFUSE_DEBUG="false"
# Example FIO job name prefix
export FIO_JOB_NAME_PREFIX="gcsfuse_test"

# --- Setup ---

echo "🚀 Starting gcsfuse/fio benchmark script..."

# Create mount point if it doesn't exist
mkdir -p "$MOUNT_POINT"

# Function to unmount the directory
cleanup() {
    echo "🧹 Unmounting $MOUNT_POINT..."
    if mountpoint -q "$MOUNT_POINT"; then
        fusermount -u "$MOUNT_POINT"
        if [ $? -eq 0 ]; then
            echo "Unmount successful."
        else
            echo "Unmount failed. You may need to manually run: sudo fusermount -u $MOUNT_POINT"
        fi
    fi
    echo "----------------------------------------"
}

# Ensure cleanup runs on exit/interrupt
trap cleanup EXIT

# --- Main Loop ---

for BS in "${BLOCK_SIZES[@]}"; do
    # 2. Set Fio Environment Variable for Block Size
    # Fio will read this variable inside the job file ($ENV_NAME)
    export BS="$BS"
    
    # 3. Run Fio 5 times for the current block size
    for i in {1..5}; do
        echo "⚙️  Starting run with Block Size (BS): $BS (Run $i of 5)"

        # 1. Mount the GCS bucket for this specific run
        echo "   Mounting GCS bucket '$GCS_BUCKET' to '$MOUNT_POINT'..."
        (cd "$HOME/workspace/gcsfuse" && go run . "$GCS_BUCKET" "$MOUNT_POINT") &
        GCSFUSE_PID=$!
        
        # Wait a moment for the mount to stabilize
        sleep 5
        
        if ! mountpoint -q "$MOUNT_POINT"; then
            echo "   ❌ ERROR: GCS bucket failed to mount. Skipping this run."
            kill $GCSFUSE_PID 2>/dev/null
            cleanup
            continue
        fi
        echo "   GCS bucket mounted successfully."
        
        # The --directory option ensures fio writes/reads in the mounted bucket.
        # The output filename now includes the run number.
        echo "   Running fio job..."
        env READ_TYPE=randread \
            DIR="$MOUNT_POINT" \
            BLOCK_SIZE="$BS" \
            NR_FILES=10 \
            fio "$FIO_JOB_FILE" \
            --output="$HOME/tmp/bs_results/fio_results_${BS}_run${i}.json" \
            --output-format=json
        FIO_EXIT_CODE=$?

        if [ $FIO_EXIT_CODE -ne 0 ]; then
            echo "   ⚠️  WARNING: Fio exited with non-zero status ($FIO_EXIT_CODE) on run $i."
        else
            echo "   ✅ Fio job completed. Results saved to $HOME/tmp/bs_results/fio_results_${BS}_run${i}.json"
        fi
        
        # 4. Unmount the directory and kill gcsfuse process before the next run
        cleanup
        kill $GCSFUSE_PID 2>/dev/null
        wait $GCSFUSE_PID 2>/dev/null
        echo "----------------------------------------"
    done

done

echo "🎉 All benchmark runs completed."
