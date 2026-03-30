#!/usr/bin/env bash
# run-distributed.sh — Coordinate a multi-host gcs-bench distributed benchmark.
#
# This script runs on each worker host independently. It:
#   1. Reads the list of workers and its own position from environment variables.
#   2. Calculates a synchronized start time ~60 s in the future so all workers
#      begin their warm-up at the same wall-clock instant.
#   3. Runs gcs-bench bench with --worker-id, --num-workers, and --start-at.
#   4. On worker 0, waits for all workers to complete then calls
#      gcs-bench merge-results to produce a combined summary.
#
# Usage (run on each host with correct WORKER_ID set):
#
#   WORKER_ID=0 NUM_WORKERS=4 OUTPUT_DIR=/results \
#     ./run-distributed.sh --config unet3d-like.yaml --duration 120s
#
# All extra arguments are forwarded to gcs-bench bench.
#
# Requirements:
#   • gcs-bench binary must be on PATH or set GCS_BENCH=/path/to/gcs-bench.
#   • Worker hosts must share a common NFS / cloud storage OUTPUT_DIR so that
#     worker 0 can find all result files for merging.
#   • Clocks must be synchronized (NTP) to within ~1 second across hosts.

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration (override via environment variables)
# ---------------------------------------------------------------------------
WORKER_ID="${WORKER_ID:-0}"
NUM_WORKERS="${NUM_WORKERS:-1}"
OUTPUT_DIR="${OUTPUT_DIR:-$(pwd)/results}"
GCS_BENCH="${GCS_BENCH:-gcs-bench}"

# Seconds to wait before starting so all workers have time to launch.
START_DELAY_SECS="${START_DELAY_SECS:-60}"

# ---------------------------------------------------------------------------
# Compute synchronized start timestamp
# ---------------------------------------------------------------------------
START_AT=$(( $(date +%s) + START_DELAY_SECS ))
echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Synchronized start at: $(date -d "@${START_AT}" --utc '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -r "${START_AT}" -u '+%Y-%m-%dT%H:%M:%SZ')"

# ---------------------------------------------------------------------------
# Create per-worker output directory
# ---------------------------------------------------------------------------
WORKER_OUTPUT="${OUTPUT_DIR}/worker-${WORKER_ID}"
mkdir -p "${WORKER_OUTPUT}"

# ---------------------------------------------------------------------------
# Run the benchmark
# ---------------------------------------------------------------------------
echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Starting gcs-bench bench..."
"${GCS_BENCH}" bench \
    --worker-id    "${WORKER_ID}" \
    --num-workers  "${NUM_WORKERS}" \
    --start-at     "${START_AT}" \
    --output-path  "${WORKER_OUTPUT}" \
    --output-format yaml \
    "$@"

echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Done. Results in ${WORKER_OUTPUT}/"

# ---------------------------------------------------------------------------
# Worker 0: wait for all worker result files then merge
# ---------------------------------------------------------------------------
if [[ "${WORKER_ID}" -eq 0 && "${NUM_WORKERS}" -gt 1 ]]; then
    echo "[coordinator] Waiting for all ${NUM_WORKERS} worker result files..."

    TIMEOUT_SECS=600
    POLL_INTERVAL=5
    WAITED=0

    worker_files=()
    while true; do
        worker_files=()
        for i in $(seq 0 $(( NUM_WORKERS - 1 ))); do
            # Find the most-recent bench-*.yaml in each worker's output dir.
            result_file=$(ls -t "${OUTPUT_DIR}/worker-${i}"/bench-*.yaml 2>/dev/null | head -1 || true)
            if [[ -n "${result_file}" ]]; then
                worker_files+=("${result_file}")
            fi
        done

        if [[ "${#worker_files[@]}" -eq "${NUM_WORKERS}" ]]; then
            echo "[coordinator] All ${NUM_WORKERS} result files found."
            break
        fi

        if [[ "${WAITED}" -ge "${TIMEOUT_SECS}" ]]; then
            echo "[coordinator] ERROR: Timed out waiting for worker result files after ${TIMEOUT_SECS}s" >&2
            echo "[coordinator] Found ${#worker_files[@]}/${NUM_WORKERS} files." >&2
            exit 1
        fi

        echo "[coordinator] ${#worker_files[@]}/${NUM_WORKERS} workers complete — waiting ${POLL_INTERVAL}s..."
        sleep "${POLL_INTERVAL}"
        WAITED=$(( WAITED + POLL_INTERVAL ))
    done

    MERGED_OUTPUT="${OUTPUT_DIR}/merged"
    mkdir -p "${MERGED_OUTPUT}"

    echo "[coordinator] Merging ${NUM_WORKERS} worker results..."
    "${GCS_BENCH}" merge-results \
        --output-path   "${MERGED_OUTPUT}" \
        --output-format both \
        "${worker_files[@]}"

    echo "[coordinator] Merged results written to ${MERGED_OUTPUT}/"
fi
