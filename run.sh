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
go run ./ --config-file=~/repro-8gbps/gcsfuse.yaml  "$BUCKET_NAME" "$MP" 

sleep 3

DIR="${MP}/" \
NUMJOBS="1" \
BS="1M" \
FILESIZE="5G" \
NRFILES="1" fio --group_reporting --output-format=normal ~/repro-8gbps/seq.fio
