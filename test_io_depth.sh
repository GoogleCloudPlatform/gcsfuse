#!/bin/bash
umount ~/gcs1 || true
echo "Prince" > ~/logs.txt
echo 256 | sudo tee /proc/sys/fs/fuse/max_pages_limit

sevirity="trace"

go install . && gcsfuse --log-severity=$sevirity --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1

# mkdir -p /home/princer_google_com/gcs1/5M
# fio --name=multi_file_5000mb \
#     --directory=/home/princer_google_com/gcs1/5M \
#     --rw=read \
#     --bs=4K \
#     --nrfiles=1 \
#     --filesize=5M \
#     --numjobs=1 \
#     --openfiles=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting


 mkdir -p /home/princer_google_com/gcs1/2G
fio --name=multi_file_64gb \
    --directory=/home/princer_google_com/gcs1/2G \
    --rw=read \
    --bs=1M \
    --nrfiles=2 \
    --filesize=2G \
    --numjobs=1 \
    --openfiles=1 \
    --ioengine=libaio \
    --direct=0 \
    --iodepth=1 \
    --group_reporting

umount "$MOUNT_POINT" || true