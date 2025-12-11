#!/bin/bash
umount ~/gcs1 || true
echo "Prince" > ~/logs.txt
echo 256 | sudo tee /proc/sys/fs/fuse/max_pages_limit
go install . && gcsfuse --max-read-ahead-kb 16384 --max-background=96 --congestion-threshold=96 --log-severity=info --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1

# mkdir -p /home/princer_google_com/gcs1/2G

# fio --name=multi_file_64gb \
#     --directory=/home/princer_google_com/gcs1/2G \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=64 \
#     --filesize=2G \
#     --numjobs=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# 2577 MB/s, 2637 MB/s, 2652 MB/s - Seems too high - needs investigation.

# mkdir -p /home/princer_google_com/gcs1/1G

# fio --name=multi_file_64gb \
#     --directory=/home/princer_google_com/gcs1/1G \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=64 \
#     --filesize=1G \
#     --numjobs=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# 2.4 GB/s 2477 MB/s, 2500 MB/s - Seems too high - needs investigation.

# mkdir -p /home/princer_google_com/gcs1/100M

# fio --name=multi_file_6400mb \
#     --directory=/home/princer_google_com/gcs1/100M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=100 \
#     --filesize=100M \
#     --numjobs=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# 873 MB/s, 942 MB/s, 962 MB/s

# mkdir -p /home/princer_google_com/gcs1/64M

# fio --name=multi_file_12800mb \
#     --directory=/home/princer_google_com/gcs1/64M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=200 \
#     --filesize=64M \
#     --numjobs=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# 718 MB/s, 762 MB/s, 795 MB/s


# mkdir -p /home/princer_google_com/gcs1/5M

# fio --name=multi_file_5000mb \
#     --directory=/home/princer_google_com/gcs1/5M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=1000 \
#     --filesize=5M \
#     --numjobs=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# 206 MB/s, 201 MB/s, 201 MB/s

# mkdir -p /home/princer_google_com/gcs1/1M

# fio --name=multi_file_1000mb \
#     --directory=/home/princer_google_com/gcs1/1M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=1000 \
#     --filesize=1M \
#     --numjobs=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# 44.1 MB/s, 45.0 MB/s, 44 MB/s

# dd if=/home/princer_google_com/gcs1/100GB of=/dev/null count=10000 bs=1M

# 1.7 GB/s, 1.6 GB/s, 1.8 GB/s
umount ~/gcs1 || true