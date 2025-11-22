#!/bin/bash
set -euo pipefail

BUCKET_NAME="$1"

DETAIL_DIR="$HOME/repro-8gbps" # Here we create bucket mount point and gcsfuse.log

mkdir -p "$DETAIL_DIR" || true

MP="$DETAIL_DIR/b"
LF="$DETAIL_DIR/gcsfuse.log"
fusermount -uz "$MP" || true
rm -rf "$MP" || true
mkdir "$MP" || true

echo "" > "$LF" 
go run ./ \
    --implicit-dirs \
    --log-severity=INFO \
    --log-file="$LF" \
    --enable-buffered-read=true \
    --metadata-cache-negative-ttl-secs=-1 \
    "$BUCKET_NAME" "$MP"

sleep 3

ls -R "$MP"

COUNT=1

for ((i=1; i<=COUNT; i++)) 
do
# Fio config is exactly same as published benchmarks fo sequential reads.
DIR="${MP}/" \
NUMJOBS="1" \
BS="1M" \
FILESIZE="20G" \
NRFILES="1" fio --group_reporting --output-format=normal ~/repro-8gbps/seq.fio
done

fusermount -uz "$MP" || true
