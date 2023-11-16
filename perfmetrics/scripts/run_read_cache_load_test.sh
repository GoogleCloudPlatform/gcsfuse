#!/bin/bash
epoch=$1
pause_in_second=$2
number_of_files_per_thread=$3
read_type=$4
fio_job_path=$5

if [ "$read_type" != "read" && "$read_type" != "readread" ]; then
  echo "Invalid read type"
  exit 1
fi

for i in $(seq $epoch); do
    fio --nrfiles $number_of_files_per_thread --rw $read_type $fio_job_path
    sleep $pause_in_second
done