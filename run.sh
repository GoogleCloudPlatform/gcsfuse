#!/bin/bash
umount ~/gcs1 || true
echo "Prince" > ~/logs.txt
echo 256 | sudo tee /proc/sys/fs/fuse/max_pages_limit
go install . && gcsfuse --max-read-ahead-kb 16384 --max-background=96 --congestion-threshold=96 --log-severity=info --log-format=text --log-file ~/logs.txt princer-gcsfuse-test-zonal-us-west4a ~/gcs1
dd if=/home/princer_google_com/gcs1/100GB of=/dev/null count=10000 bs=1M
umount ~/gcs1 || true