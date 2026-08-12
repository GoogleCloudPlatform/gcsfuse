#!/bin/bash

BUCKET="repro-bench-mohit"
MNT_DIR="/mnt/gcs"
BUCKET_URL="gs://${BUCKET}"

run_bench() {
  local config_file=$1
  local output_file=$2
  local protocol=$3

  echo "========================================================="
  echo "Running benchmark with config: $config_file ($protocol)"
  echo "========================================================="

  sudo umount $MNT_DIR 2>/dev/null || true
  
  # We use --foreground to easily see if mount fails, but here we want to run it in background as daemon
  sudo gcsfuse \
    --cache-dir=/mnt/disks/lssd \
    --config-file=$config_file \
    $BUCKET $MNT_DIR

  if [ $? -ne 0 ]; then
    echo "Failed to mount gcsfuse with config $config_file"
    return 1
  fi

  echo "Mounted successfully. Running benchmark..."
  sudo python3 stage3_repro.py $MNT_DIR $BUCKET_URL > $output_file 2>&1
  
  echo "Benchmark finished. Unmounting..."
  sudo umount $MNT_DIR
}

run_bench "mount_config_http.yaml" "benchmark_results_http.txt" "HTTP"
run_bench "mount_config_grpc.yaml" "benchmark_results_grpc.txt" "gRPC (Default DP)"
run_bench "mount_config_grpc_dp_only.yaml" "benchmark_results_grpc_dp_only.txt" "gRPC (DP Only)"

echo "All benchmarks completed!"
