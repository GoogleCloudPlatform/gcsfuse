#!/usr/bin/env bash

# Exit on error
set -e

# Constants & Defaults
LOG_FILE="benchmark_delete.log"
MOUNT_DIR="$HOME/mnt"
THREAD_COUNT=256  # Default thread count for parallel deletion
BUCKET_NAME="mohits-checkpoint-deletion-data-test"
DIR_NAME="300"
SOURCE_BUCKET="mohits-checkpoint-deletion-data-main"

# Log file configuration: redirect all stdout/stderr and truncate it
exec > >(tee "$LOG_FILE") 2>&1



usage() {
  echo "Usage: $0 -b <bucket_name> -d <dir_to_delete> -m <sequential|parallel> [-t <thread_count>]"
  echo "  -b: GCS Bucket name"
  echo "  -d: Directory name inside mount to delete (relative path, e.g., '2200')"
  echo "  -m: Deletion mode ('sequential' or 'parallel')"
  echo "  -t: Number of parallel threads for parallel delete (default: 256)"
  exit 1
}

# Parse arguments
while getopts "b:d:m:t:" opt; do
  case ${opt} in
    b ) BUCKET_NAME=$OPTARG ;;
    d ) DIR_NAME=$OPTARG ;;
    m ) MODE=$OPTARG ;;
    t ) THREAD_COUNT=$OPTARG ;;
    * ) usage ;;
  esac
done

if [[ -z "$BUCKET_NAME" || -z "$DIR_NAME" || -z "$MODE" ]]; then
  usage
fi

if [[ "$MODE" != "sequential" && "$MODE" != "parallel" ]]; then
  echo "Error: Mode must be 'sequential' or 'parallel'"
  usage
fi

# 1. Prepare mount directory
mkdir -p "$MOUNT_DIR"

# 2. Unmount if already mounted
if mountpoint -q "$MOUNT_DIR"; then
  echo "Mount point $MOUNT_DIR is already mounted. Unmounting..."
  # Try graceful FUSE unmount first, fallback to standard umount
  fusermount -u "$MOUNT_DIR" || umount -l "$MOUNT_DIR"
  sleep 1
fi


# 2b. Transfer folder from main to test bucket using Storage Transfer Service
echo "Synchronizing data from gs://$SOURCE_BUCKET/$DIR_NAME/ to gs://$BUCKET_NAME/$DIR_NAME/ using Storage Transfer Service..."
gcloud transfer jobs create "gs://$SOURCE_BUCKET" "gs://$BUCKET_NAME" --include-prefixes="$DIR_NAME/" --overwrite-when=different --no-async

# 2c. Log the size of the transferred folder
echo "Calculating and logging the size of the transferred folder gs://$BUCKET_NAME/$DIR_NAME/..."
FOLDER_DU_OUTPUT=$(gcloud storage du -s "gs://$BUCKET_NAME/$DIR_NAME/")
RAW_BYTES=$(echo "$FOLDER_DU_OUTPUT" | awk '{print $1}')
if [[ "$RAW_BYTES" =~ ^[0-9]+$ ]]; then
  HUMAN_SIZE=$(numfmt --to=iec-i --suffix=B "$RAW_BYTES")
  echo "--------------------------------------------------"
  echo "Transferred Folder Size: $HUMAN_SIZE ($RAW_BYTES bytes)"
  echo "--------------------------------------------------"
else
  echo "--------------------------------------------------"
  echo "Transferred Folder Size: $FOLDER_DU_OUTPUT"
  echo "--------------------------------------------------"
fi


# 3. Mount GCS Bucket
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

echo "Clearing old gcsfuse mount logs..."
> "$SCRIPT_DIR/gcsfuse.log"

echo "Mounting bucket '$BUCKET_NAME' to '$MOUNT_DIR' using 'go run .' with trace logging..."
(cd "$SCRIPT_DIR" && go run . --log-severity=trace --log-file="$SCRIPT_DIR/gcsfuse.log" "$BUCKET_NAME" "$MOUNT_DIR")
sleep 2

TARGET_PATH="$MOUNT_DIR/$DIR_NAME"

# 4. Verify target directory exists
if [[ ! -d "$TARGET_PATH" ]]; then
  echo "Error: Target directory '$TARGET_PATH' does not exist."
  exit 1
fi

echo "Target directory to delete: $TARGET_PATH"
echo "Deletion Mode: $MODE"

# Start timer
START_TIME=$(date +%s.%N)

echo "Calculating total starting files in gs://$BUCKET_NAME/$DIR_NAME/..."
TOTAL_FILES=$(gcloud storage ls "gs://$BUCKET_NAME/$DIR_NAME/**" 2>/dev/null | wc -l || echo "0")
echo "Total starting files: $TOTAL_FILES"

# Start background progress monitor
monitor_progress() {
  while true; do
    sleep 60
    local remaining
    remaining=$(gcloud storage ls "gs://$BUCKET_NAME/$DIR_NAME/**" 2>/dev/null | wc -l || echo "0")
    local current_time
    current_time=$(date +"%Y-%m-%d %H:%M:%S")
    echo "[$current_time] Progress Monitor: $remaining / $TOTAL_FILES files remaining."
  done
}

echo "Launching background progress monitor..."
monitor_progress &
MONITOR_PID=$!

# Ensure background monitor is terminated when script exits
cleanup() {
  local exit_code=$?
  kill "$MONITOR_PID" 2>/dev/null || true
  exit $exit_code
}
trap cleanup EXIT SIGINT SIGTERM

# 5. Perform Deletion
if [[ "$MODE" == "sequential" ]]; then
  echo "Starting sequential deletion (rm -rf)..."
  rm -rf "$TARGET_PATH"
else
  echo "Starting parallel deletion using $THREAD_COUNT threads..."
  # Phase 1: Delete all files in parallel across subdirectories
  echo "Phase 1: Deleting files in parallel..."
  find "$TARGET_PATH" -type f -print0 | xargs -0 -P "$THREAD_COUNT" rm -f

  # Phase 2: Clean up remaining empty directories
  echo "Phase 2: Cleaning up empty directories..."
  find "$TARGET_PATH" -depth -type d -empty -delete
fi

# End timer
END_TIME=$(date +%s.%N)

# Calculate elapsed time
ELAPSED_TIME=$(echo "$END_TIME - $START_TIME" | bc -l)

echo "--------------------------------------------------"
printf "Deletion completed successfully!\n"
printf "Total Deletion Time: %.3f seconds\n" "$ELAPSED_TIME"
echo "--------------------------------------------------"
