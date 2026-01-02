#!/bin/bash
umount ~/gcs1 || true
echo "Prince" > ~/logs.txt
echo 256 | sudo tee /proc/sys/fs/fuse/max_pages_limit

sevirity="trace"

go install . && gcsfuse --max-read-ahead-kb 1024 --max-background=96 --congestion-threshold=96 --log-severity=$sevirity --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1

fio --name=multi_file_5000mb \
    --directory=/home/princer_google_com/gcs1/5M \
    --rw=read \
    --bs=4K \
    --nrfiles=1 \
    --filesize=5M \
    --numjobs=1 \
    --openfiles=1 \
    --ioengine=libaio \
    --direct=0 \
    --group_reporting


umount "$MOUNT_POINT" || true