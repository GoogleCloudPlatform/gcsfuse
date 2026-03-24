#!/bin/bash
# Mount gcsfuse with batch read, read 5GiB file with dd, and unmount
# Usage: ./test_batch_read.sh BUCKET_NAME [FILE_NAME]

set -e

BUCKET="${1:?Error: Bucket name required}"
FILE="${2:-test-5gb.bin}"
MOUNT_DIR="/tmp/gcsfuse_$$"

cleanup() {
    fusermount -u "$MOUNT_DIR" 2>/dev/null || true
    rmdir "$MOUNT_DIR" 2>/dev/null || true
}
trap cleanup EXIT

# Mount with batch read enabled
mkdir -p "$MOUNT_DIR"
echo "Mounting gs://$BUCKET with batch read..."
gcsfuse --enable-batch-read "$BUCKET" "$MOUNT_DIR"

# Create test file if it doesn't exist
if [ ! -f "$MOUNT_DIR/$FILE" ]; then
    echo "Creating test file $FILE (5GiB) - this is a one-time operation..."
    dd if=/dev/zero of="$MOUNT_DIR/$FILE" bs=1M count=5120 status=progress
    echo "Test file created successfully."
fi

# Read the file with dd
echo "Reading $FILE (5GiB)..."
time dd if="$MOUNT_DIR/$FILE" of=/dev/null bs=1M

echo "Done!"
