#!/bin/bash

# set -x

file_name=${1:-"out.txt"}
current_dir=$(pwd)
file_name=$current_dir/$file_name

# take a default value of 2nd argument, in a single line
protocol=${2:-"http"}


set +e # Don't fail the script in case of failure.
umount ~/bucket
set -e # Fail the script in case of failure.

if [[ "$protocol" == "grpc" ]]; then
    gcsfuse --implicit-dirs --client-protocol grpc princer-grpc-read-test-uc1a ~/bucket | tee -a $file_name
else
    gcsfuse --implicit-dirs princer-grpc-read-test-uc1a ~/bucket | tee $file_name
fi

cd ~/bucket

patterns=("read" "randread")
# jobs=(16 48 96)
jobs=(16)
for job in ${jobs[@]}; do
    for pattern in ${patterns[@]}; do
        echo "Running for $pattern with $job jobs..."
        BLOCK_SIZE=128K FILE_SIZE=1G MODE=$pattern NUMJOBS=$job fio ~/dev/gcsfuse-tools/read-test/read.fio | tee -a $file_name
        echo "Running for $pattern with $job jobs completed."
        sleep 300s
    done
done

cd -

umount ~/bucket