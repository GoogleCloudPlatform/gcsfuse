#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

print_usage() {
  echo "Help/Supported options..."
  printf "./run_read_cache_workload  "
  printf "[-e epoch] [-p pause_after_every_epoch_in_seconds] "
  printf "[-n number_of_files_per_thread] "
  printf "[-t number_of_threads] "
  printf "[-r read_type (read | randread)] "
  printf "[-s file_size (in K, M, G E.g. 10K] "
  printf "[-b block_size (in K, M, G E.g. 20K] "
  printf "[-d workload directory] \n"
}

# Default values:
epoch=2
no_of_files_per_thread=1
read_type=read
pause_in_seconds=2
block_size=1K
file_size=1K
num_of_threads=40

while getopts he:p:n:r:d:s:b:t: flag
do
  case "${flag}" in
    e) epoch=${OPTARG};;
    p) pause_in_seconds=${OPTARG};;
    n) no_of_files_per_thread=${OPTARG};;
    r) read_type=${OPTARG};;
    d) workload_dir=${OPTARG};;
    s) file_size=${OPTARG};;
    b) block_size=${OPTARG};;
    t) number_of_threads=${OPTARG};;
    h) print_usage
        exit 0 ;;
    *) print_usage
        exit 1 ;;
  esac
done

if [ ! -d "$workload_dir" ]; then
  echo "Please pass a valid workload dir with -d options..."
  exit 1
fi

if [[ -z "${WORKING_DIR}" ]]; then
  echo "Please set the working directory..."
  exit 1
fi

if [[ "${read_type}" != "read" && "${read_type}" != "randread" ]]; then
  echo "Please pass a valid read typr -r (read | randread)..."
  exit 1
fi

cd $WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/

for i in $(seq $epoch); do
  free -mh

  echo "[Epoch ${i}] start time:" `date +%s`
  NUMJOBS=$num_of_threads NRFILES=$no_of_files_per_thread FILE_SIZE=$file_size BLOCK_SIZE=$block_size READ_TYPE=$read_type DIR=$workload_dir fio $WORKING_DIR/gcsfuse/perfmetrics/scripts/job_files/read_cache_load_test.fio ----alloc-size=1048576
  echo "[Epoch ${i}] end time:" `date +%s`

  free -mh
  sudo ./clean_kernel_cache.sh
  sleep $pause_in_seconds
done

cd -
