#!/bin/bash
set -euo pipefail

BUCKET_NAME="$1"

DETAIL_DIR="$HOME" # Here we create bucket mount point and gcsfuse.log

mkdir -p "$DETAIL_DIR" || true

print() {
    sleep 3
    cat "$MP/print" || true
}

reset() {
    sleep 3
    cat "$MP/reset" || true
}

MP="$DETAIL_DIR/b"
LF="gcsfuse.log"
fusermount -uz "$MP" || true
rm -rf "$MP" || true
mkdir "$MP" || true

echo "" > "$LF" 
go run ./ \
    --implicit-dirs \
    --log-severity=info \
    --log-file="$LF" \
    --enable-buffered-read=true \
    --metadata-cache-negative-ttl-secs=-1 \
    "$BUCKET_NAME" "$MP"

sleep 6

ls -R "$MP"

cat << 'EOF' > seq.fio
[global]
ioengine=libaio
direct=1
fadvise_hint=0
iodepth=64
invalidate=1
thread=1
openfiles=1
group_reporting=1
create_serialize=0
allrandrepeat=0
file_service_type=random
rw=read
filename_format=$jobname.$jobnum.$filenum.size-${FILESIZE}

[seq_read]
directory=${DIR}
filesize=${FILESIZE}
bs=${BS}
numjobs=${NUMJOBS}
nrfiles=${NRFILES}
EOF

# Fio config is exactly same as published benchmarks fo sequential reads.
DIR="${MP}/" \
NUMJOBS="1" \
BS="1M" \
FILESIZE="20G" \
NRFILES="1" fio --group_reporting --output-format=normal seq.fio

reset

# Fio config is exactly same as published benchmarks fo sequential reads.
DIR="${MP}/" \
NUMJOBS="1" \
BS="1M" \
FILESIZE="20G" \
NRFILES="1" fio --group_reporting --output-format=normal seq.fio

print
reset

fusermount -uz "$MP" || true

