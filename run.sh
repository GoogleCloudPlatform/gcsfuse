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
go run ./ --implicit-dirs --log-severity=INFO --log-file="$LF" --enable-buffered-read=true --read-global-max-blocks=100000 --metadata-cache-negative-ttl-secs=-1 "$BUCKET_NAME" "$MP" 

sleep 3


# Fio config is exactly same as published benchmarks fo sequential reads.
DIR="${MP}/" \
NUMJOBS="1" \
BS="1M" \
FILESIZE="5G" \
NRFILES="1" fio --group_reporting --output-format=normal ~/repro-8gbps/seq.fio

fusermount -uz "$MP" || true
