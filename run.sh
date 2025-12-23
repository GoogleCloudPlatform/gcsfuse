#!/bin/bash

if [ "$1" == "" ]; then
    umount ~/gcs1 || true
    echo "Prince" > ~/logs.txt

    sevirity="info"
    go install . && gcsfuse --mrd-cache-max-instances=5 --log-severity=$sevirity --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1
fi  

mkdir -p /home/princer_google_com/gcs1/100M

fio --name=multi_file_6400mb \
    --directory=/home/princer_google_com/gcs1/100M \
    --rw=randread \
    --bs=1M \
    --nrfiles=5 \
    --filesize=100M \
    --numjobs=1 \
    --openfiles=1 \
    --ioengine=libaio \
    --direct=1 \
    --group_reporting