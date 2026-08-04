#!/bin/bash
# Manually creates HNS RCU buckets with gcsfuse-test-hns-pirlo-<package_name> prefix for all GCSFuse integration test packages.
# Example usage:
#   ./tools/integration_tests/setup_all_pirlo_buckets.sh /google/src/cloud/avoidnull/b-504681452/google3

set -e

GOOGLE3_ROOT="${1:-/google/src/cloud/avoidnull/b-504681452/google3}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RCU_SCRIPT="${SCRIPT_DIR}/create_rcu_bucket.sh"

if [[ ! -f "${RCU_SCRIPT}" ]]; then
  echo "Error: Could not find ${RCU_SCRIPT}"
  exit 1
fi

# Complete list of all integration test packages in tools/integration_tests/
TEST_PACKAGES=(
  "benchmarking"
  "buffered_read"
  "cloud_profiler"
  "concurrent_operations"
  "dentry_cache"
  "explicit_dir"
  "flag_optimizations"
  "grpc_validation"
  "gzip"
  "implicit_dir"
  "inactive_stream_timeout"
  "interrupt"
  "kernel_list_cache"
  "list_large_dir"
  "local_file"
  "log_rotation"
  "managed_folders"
  "monitoring"
  "mount_timeout"
  "mounting"
  "negative_stat_cache"
  "operations"
  "rapid_operations"
  "read_cache"
  "read_gcs_algo"
  "read_large_files"
  "readdirplus"
  "readonly"
  "readonly_creds"
  "release_version"
  "rename_dir_limit"
  "requester_pays_bucket"
  "shared_chunk_cache"
  "stale_handle"
  "streaming_writes"
  "symlink_handling"
  "unfinalized_object"
  "unsupported_path"
  "write_large_files"
)

echo "Setting up HNS Pirlo buckets for ${#TEST_PACKAGES[@]} test packages using ${RCU_SCRIPT}..."
for pkg in "${TEST_PACKAGES[@]}"; do
  package_slug="${pkg//_/-}"
  bucket_name="gcsfuse-test-hns-pirlo-${package_slug}"
  echo "=== Creating/Verifying HNS Pirlo bucket: gs://${bucket_name} (package: ${pkg}) ==="
  "${RCU_SCRIPT}" "${bucket_name}" "${GOOGLE3_ROOT}"
done

echo "All ${#TEST_PACKAGES[@]} HNS Pirlo buckets created/verified successfully!"
