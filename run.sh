#!/bin/bash
umount ~/gcs1 || true
echo "Prince" > ~/logs.txt
echo 256 | sudo tee /proc/sys/fs/fuse/max_pages_limit

sevirity="info"

if [[ $1 == "BR" ]] ; then
    echo "Running with buffered-read"
    go install . && gcsfuse --max-read-ahead-kb 16384 --enable-buffered-read --read-global-max-blocks 1000 --log-severity=$sevirity --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1
elif [[ $1 == "SR" ]] ; then
    echo "Running with simple reader"
    go install . && gcsfuse --max-read-ahead-kb 16384 --async-read --max-background=96 --congestion-threshold=96 --log-severity=$sevirity --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1
else
    # mrd pool is disabled, kernel settings are default and it goes with NewRangeReader calls.
    echo "Running with default reader"
    go install . && gcsfuse --max-read-ahead-kb 16384 --log-severity=$sevirity --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1
fi

# mkdir -p /home/princer_google_com/gcs1/2G

# fio --name=multi_file_64gb \
#     --directory=/home/princer_google_com/gcs1/2G \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=32 \
#     --filesize=2G \
#     --numjobs=1 \
#     --openfiles=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --iodepth=1 \
#     --group_reporting

# Simple reader: 2577 MB/s, 2637 MB/s, 2652 MB/s - Seems too high - needs investigation. (this was with default openfiles=nrfiles)
# Simple reader (openfiles=1): 2238 MB/s, 2303 MB/s, 2298 MB/s
# Buffered reader: 1225 MB/s, 1356 MB/s, 1386 MB/s
# Default reader (openfiles = 1): 770 MB/s, 756 MB/s, 766 MB/s

# mkdir -p /home/princer_google_com/gcs1/1G

# fio --name=multi_file_64gb \
#     --directory=/home/princer_google_com/gcs1/1G \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=64 \
#     --filesize=1G \
#     --numjobs=1 \
#     --openfiles=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# Simple reader: 2400 MB/s, 2477 MB/s, 2500 MB/s - Seems too high - needs investigation.
# Simple reader (openfiles=1): 2064 MB/s, 2067 MB/s, 2143 MB/s
# Buffered reader: 1318 MB/s, 1383 MB/s, 1319 MB/s
# Default reader (openfiles = 1): 759 MB/s, 781 MB/s, 769 MB/s


# mkdir -p /home/princer_google_com/gcs1/100M

# fio --name=multi_file_6400mb \
#     --directory=/home/princer_google_com/gcs1/100M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=100 \
#     --filesize=100M \
#     --numjobs=1 \
#     --openfiles=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# Simple reader: 873 MB/s, 942 MB/s, 962 MB/s
# Simple reader (openfiles=1): 857 MB/s, 858 MB/s, 887 MB/s
# Buffered reader: 909 MB/s, 901 MB/s, 921 MB/s
# Default reader (openfiles = 1): 647 MB/s, 688 MB/s, 673 MB/s

# mkdir -p /home/princer_google_com/gcs1/64M

# fio --name=multi_file_12800mb \
#     --directory=/home/princer_google_com/gcs1/64M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=200 \
#     --filesize=64M \
#     --numjobs=1 \
#     --openfiles=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# Simple reader: 718 MB/s, 762 MB/s, 795 MB/s
# Simple reader (openfiles=1): 647 MB/s, 686 MB/s, 675 MB/s
# Buffered reader: 791 MB/s, 798 MB/s, 809 MB/s
# Default reader (openfiles = 1): 680 MB/s, 657 MB/s, 683 MB/s

# mkdir -p /home/princer_google_com/gcs1/5M

# fio --name=multi_file_5000mb \
#     --directory=/home/princer_google_com/gcs1/5M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=1000 \
#     --filesize=5M \
#     --numjobs=1 \
#     --openfiles=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# Simple reader: 206 MB/s, 201 MB/s, 201 MB/s
# Simple reader (openfiles=1, reset the pool-size to 1): 184 MB/s, 184 MB/s, 186 MB/s
# Buffered reader: 162 MB/s, 162 MB/s, 162 MB/s
# Default reader (openfiles = 1): 182 MB/s, 184 MB/s, 184 MB/s

# mkdir -p /home/princer_google_com/gcs1/1M

# fio --name=multi_file_1000mb \
#     --directory=/home/princer_google_com/gcs1/1M \
#     --rw=read \
#     --bs=1M \
#     --nrfiles=1000 \
#     --filesize=1M \
#     --numjobs=1 \
#     --openfiles=1 \
#     --ioengine=libaio \
#     --direct=0 \
#     --group_reporting

# Simple reader: 44.1 MB/s, 45.0 MB/s, 44 MB/s
# Simple reader (openfiles=1, reset the pool-size to 1): 52 MB/s, 47 MB/s, 49 MB/s
# Buffered reader: 44.1 MB/s, 45.0 MB/s, 46.0 MB/s
# Default reader (openfiles = 1): 48 MB/s, 49.9 MB/s, 50 MB/s

# mkdir -p /home/princer_google_com/gcs1/5G

# fio --name=mmap_concurrent_read \
#     --directory=/home/princer_google_com/gcs1/5G \
#     --rw=read \
#     --bs=1M \
#     --size=5G \
#     --numjobs=8 \
#     --ioengine=mmap \
#     --group_reporting \
#     --fadvise_hint=1

# dd if=/home/princer_google_com/gcs1/100GB of=/dev/null count=10000 bs=1M

# 1.7 GB/s, 1.6 GB/s, 1.8 GB/s
umount ~/gcs1 || true