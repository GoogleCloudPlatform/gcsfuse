#!/bin/bash
#
# Helper script to generate gRPC metrics for GCSFuse.
# Usage: ./tools/generate_grpc_metrics.sh <mount_point> [prometheus_port]
# Default prometheus_port is 8082.

MOUNT_POINT=$1
PROM_PORT=${2:-8082}

if [ -z "$MOUNT_POINT" ]; then
  echo "Usage: $0 <mount_point> [prometheus_port]"
  exit 1
fi

if [ ! -d "$MOUNT_POINT" ]; then
  echo "Error: Mount point $MOUNT_POINT does not exist or is not a directory."
  exit 1
fi

echo "Generating traffic on $MOUNT_POINT..."

# Perform some file operations to trigger gRPC calls
echo "Listing directory..."
ls -R "$MOUNT_POINT" > /dev/null 2>&1

echo "Writing a test file..."
TEST_FILE="$MOUNT_POINT/test_grpc_metrics.txt"
dd if=/dev/zero of="$TEST_FILE" bs=1M count=1 > /dev/null 2>&1

echo "Reading the test file..."
cat "$TEST_FILE" > /dev/null 2>&1

echo "Deleting the test file..."
rm "$TEST_FILE" > /dev/null 2>&1

echo "Traffic generation complete."
echo "Fetching metrics from localhost:$PROM_PORT..."

# Fetch metrics
METRICS=$(curl -s "http://localhost:$PROM_PORT/metrics")

if [ -z "$METRICS" ]; then
  echo "Error: Failed to fetch metrics. Is GCSFuse running with --prometheus-port=$PROM_PORT?"
  exit 1
fi

echo "--- Requested Metrics ---"

# Filter for the requested metrics
echo "$METRICS" | grep -E "grpc_client_attempt_started|grpc_client_attempt_duration|grpc_client_attempt_sent_total_compressed_message_size|grpc_client_attempt_rcvd_total_compressed_message_size|grpc_client_call_duration|grpc_lb_rls_default_target_picks|grpc_lb_rls_failed_picks"

echo "-------------------------"
echo "Note: RLS metrics (grpc_lb_rls_*) may not be present if RLS is not active in the current environment."
