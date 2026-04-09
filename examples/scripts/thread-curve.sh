#!/usr/bin/env bash
# thread-curve.sh — Run gcs-bench bench at a sweep of concurrency levels.
#
# For each concurrency level the script runs a full warm-up + measurement
# phase, captures the results from the TSV output, then prints a consolidated
# summary table so you can see how latency and throughput scale with threads.
#
# The MRD MultiRangeDownloader bidi-gRPC connection cache is preserved across
# levels (it is internal to gcs-bench per-process), so each level starts with
# a cold cache.  To profile steady-state connection reuse, increase --warmup.
#
# Usage
# -----
#   ./thread-curve.sh --config rapid-mrd-8k-example.yaml
#
# Defaults:
#
#   ./thread-curve.sh \
#       --config rapid-mrd-8k-example.yaml \
#       --sweep  8,16,32,48,64,96,128,192,256,384,512 \
#       --duration 2m \
#       --warmup  20s \
#       --output  /tmp/thread-curve-results
#
# Flags
# -----
#   --config  <file>   (required) Path to the gcs-bench YAML config.
#   --sweep   <list>   Comma-separated concurrency levels.
#                      Default: 8,16,32,48,64,96,128,192,256,384
#   --duration <dur>   Override the benchmark measurement duration per level.
#                      Accepts Go duration strings (e.g. 2m, 90s).
#                      Default: use the value in the config file.
#   --warmup   <dur>   Override the warmup duration per level.
#                      Default: use the value in the config file.
#   --output   <dir>   Root directory for all result files.
#                      Each level writes to <dir>/threads-N/bench-*/
#                      Default: thread-curve-$(date +%Y%m%d-%H%M%S)/
#   --dry-run          Print the commands that would run, but do not execute.
#
# Requirements
# ------------
#   • gcs-bench binary on PATH, or GCS_BENCH=/path/to/gcs-bench
#   • awk, column (util-linux) — for table formatting
#
# Output
# ------
#   Per-level results are written under <output>/threads-N/.
#   A consolidated TSV summary is written to <output>/thread-curve.tsv.
#   A human-readable table is printed to stdout at the end.

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
GCS_BENCH="${GCS_BENCH:-gcs-bench}"
CONFIG_FILE=""
SWEEP="8,16,32,48,64,96,128,192,256,384,512"
DURATION_OVERRIDE=""
WARMUP_OVERRIDE=""
OUTPUT_ROOT=""
DRY_RUN=false

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --config)   CONFIG_FILE="$2";       shift 2 ;;
        --sweep)    SWEEP="$2";             shift 2 ;;
        --duration) DURATION_OVERRIDE="$2"; shift 2 ;;
        --warmup)   WARMUP_OVERRIDE="$2";   shift 2 ;;
        --output)   OUTPUT_ROOT="$2";       shift 2 ;;
        --dry-run)  DRY_RUN=true;           shift   ;;
        *)
            echo "Unknown argument: $1" >&2
            echo "Usage: $0 --config <file> [--sweep 8,16,32,...] [--duration 2m] [--warmup 20s] [--output dir] [--dry-run]" >&2
            exit 1
            ;;
    esac
done

if [[ -z "${CONFIG_FILE}" ]]; then
    echo "Error: --config is required" >&2
    exit 1
fi
if [[ ! -f "${CONFIG_FILE}" ]]; then
    echo "Error: config file not found: ${CONFIG_FILE}" >&2
    exit 1
fi

# Default output directory — timestamped so repeated runs don't overwrite.
if [[ -z "${OUTPUT_ROOT}" ]]; then
    OUTPUT_ROOT="thread-curve-$(date +%Y%m%d-%H%M%S)"
fi

# ---------------------------------------------------------------------------
# Resolve gcs-bench binary (skip check in --dry-run mode)
# ---------------------------------------------------------------------------
if [[ "${DRY_RUN}" == "false" ]] && ! command -v "${GCS_BENCH}" &>/dev/null; then
    echo "Error: gcs-bench not found on PATH. Set GCS_BENCH=/path/to/gcs-bench." >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Parse sweep list
# ---------------------------------------------------------------------------
IFS=',' read -ra LEVELS <<< "${SWEEP}"

echo "==================================================="
echo "  gcs-bench thread-curve sweep"
echo "==================================================="
echo "  Config:     ${CONFIG_FILE}"
echo "  Levels:     ${SWEEP}"
[[ -n "${DURATION_OVERRIDE}" ]] && echo "  Duration:   ${DURATION_OVERRIDE} (override)"
[[ -n "${WARMUP_OVERRIDE}"   ]] && echo "  Warmup:     ${WARMUP_OVERRIDE} (override)"
echo "  Output:     ${OUTPUT_ROOT}/"
echo "==================================================="
echo ""

# ---------------------------------------------------------------------------
# Summary accumulator file (TSV)
# ---------------------------------------------------------------------------
if [[ "${DRY_RUN}" == "false" ]]; then
    mkdir -p "${OUTPUT_ROOT}"
fi

SUMMARY_TSV="${OUTPUT_ROOT}/thread-curve.tsv"

# Header matches the per-run bench.tsv columns we extract, plus 'concurrency'.
# We write our own summary file rather than cat-ing bench.tsv rows so that
# the concurrency value is always explicit (bench.tsv has goroutines from the
# actual run, which may differ from the requested level if weight splitting
# changed it slightly).
SUMMARY_HEADER="concurrency\ttrack\tgoroutines\tops_per_sec\tthroughput_mib_s\tavg_op_size_bytes\ttotal_p50_us\ttotal_p90_us\ttotal_p95_us\ttotal_p99_us\ttotal_mean_us"

if [[ "${DRY_RUN}" == "false" ]]; then
    printf '%b\n' "${SUMMARY_HEADER}" > "${SUMMARY_TSV}"
fi

# ---------------------------------------------------------------------------
# Run each concurrency level
# ---------------------------------------------------------------------------
declare -A ROW   # keyed by concurrency level for the final table

for THREADS in "${LEVELS[@]}"; do
    THREADS="${THREADS// /}"   # strip any stray spaces
    [[ -z "${THREADS}" ]] && continue

    LEVEL_DIR="${OUTPUT_ROOT}/threads-${THREADS}"

    echo "---------------------------------------------------"
    echo "  Running: concurrency = ${THREADS}"
    echo "  Output:  ${LEVEL_DIR}/"
    echo "---------------------------------------------------"

    # Build gcs-bench command.
    BENCH_CMD=(
        "${GCS_BENCH}" bench
        --config       "${CONFIG_FILE}"
        --concurrency  "${THREADS}"
        --output-path  "${LEVEL_DIR}"
        --output-format both
    )
    [[ -n "${DURATION_OVERRIDE}" ]] && BENCH_CMD+=(--duration "${DURATION_OVERRIDE}")
    [[ -n "${WARMUP_OVERRIDE}"   ]] && BENCH_CMD+=(--warmup   "${WARMUP_OVERRIDE}")

    if [[ "${DRY_RUN}" == "true" ]]; then
        echo "  [dry-run] ${BENCH_CMD[*]}"
        # Populate a placeholder row so the table is still shown.
        ROW["${THREADS}"]="${THREADS}\t(dry-run)\t-\t-\t-\t-\t-\t-\t-\t-\t-"
        continue
    fi

    mkdir -p "${LEVEL_DIR}"

    # Execute the benchmark.  Capture exit code without causing set -e to abort.
    set +e
    "${BENCH_CMD[@]}"
    EXIT_CODE=$?
    set -e

    if [[ ${EXIT_CODE} -ne 0 ]]; then
        echo "  WARNING: gcs-bench exited with code ${EXIT_CODE} for concurrency=${THREADS}" >&2
        ROW["${THREADS}"]="${THREADS}\tERROR\t-\t-\t-\t-\t-\t-\t-\t-\t-"
        continue
    fi

    # Locate the bench.tsv that was just written.
    # The exporter creates: <output-path>/bench-YYYYMMDD-HHMMSS/bench.tsv
    TSV_FILE=$(find "${LEVEL_DIR}" -name "bench.tsv" -type f 2>/dev/null | sort | tail -1)

    if [[ -z "${TSV_FILE}" ]]; then
        echo "  WARNING: bench.tsv not found in ${LEVEL_DIR}" >&2
        ROW["${THREADS}"]="${THREADS}\tNO_RESULTS\t-\t-\t-\t-\t-\t-\t-\t-\t-"
        continue
    fi

    # Extract the first data row (skip header).  Column indices (1-based):
    #   1  track              2  goroutines       3  ops_total
    #   4  errors             5  ops_per_sec      6  throughput_mib_s
    #   7  avg_op_size_bytes  8  ttfb_p50_us      ...
    #  15  total_p50_us      16  total_p90_us     17  total_p95_us
    #  18  total_p99_us      19  total_p999_us    20  total_max_us     21  total_mean_us
    DATA_ROW=$(awk -F'\t' 'NR==2 {print $0}' "${TSV_FILE}")

    if [[ -z "${DATA_ROW}" ]]; then
        echo "  WARNING: bench.tsv has no data row in ${TSV_FILE}" >&2
        ROW["${THREADS}"]="${THREADS}\tEMPTY\t-\t-\t-\t-\t-\t-\t-\t-\t-"
        continue
    fi

    # Extract fields we care about with awk.
    TRACK=$(awk -F'\t' 'NR==2 {print $1}' "${TSV_FILE}")
    GOROUTINES=$(awk -F'\t' 'NR==2 {print $2}' "${TSV_FILE}")
    OPS_PER_SEC=$(awk -F'\t' 'NR==2 {print $5}' "${TSV_FILE}")
    THROUGHPUT=$(awk -F'\t' 'NR==2 {print $6}' "${TSV_FILE}")
    AVG_SIZE=$(awk -F'\t' 'NR==2 {printf "%.0f", $7}' "${TSV_FILE}")
    P50=$(awk -F'\t' 'NR==2 {print $15}' "${TSV_FILE}")
    P90=$(awk -F'\t' 'NR==2 {print $16}' "${TSV_FILE}")
    P95=$(awk -F'\t' 'NR==2 {print $17}' "${TSV_FILE}")
    P99=$(awk -F'\t' 'NR==2 {print $18}' "${TSV_FILE}")
    MEAN=$(awk -F'\t' 'NR==2 {print $21}' "${TSV_FILE}")

    # Append to summary TSV.
    SUMMARY_ROW="${THREADS}\t${TRACK}\t${GOROUTINES}\t${OPS_PER_SEC}\t${THROUGHPUT}\t${AVG_SIZE}\t${P50}\t${P90}\t${P95}\t${P99}\t${MEAN}"
    printf '%b\n' "${SUMMARY_ROW}" >> "${SUMMARY_TSV}"

    # Store for final table.
    ROW["${THREADS}"]="${SUMMARY_ROW}"

    echo ""
done

# ---------------------------------------------------------------------------
# Final summary table
# ---------------------------------------------------------------------------
echo ""
echo "==================================================="
echo "  Thread-Curve Summary"
echo "==================================================="
echo ""

# Print header + rows as a fixed-width table using column(1).
# All latency values from bench.tsv are in microseconds; convert to ms for
# readability when the value is >= 1000 µs (which is typical at 96 threads).
print_table() {
    printf '%-12s  %-20s  %10s  %12s  %12s  %10s  %10s  %10s  %10s\n' \
        "Concurrency" "Track" "Ops/sec" "Throughput" "AvgSize" \
        "P50(µs)" "P90(µs)" "P99(µs)" "Mean(µs)"
    printf '%-12s  %-20s  %10s  %12s  %12s  %10s  %10s  %10s  %10s\n' \
        "------------" "--------------------" "----------" "------------" "------------" \
        "----------" "----------" "----------" "----------"

    for THREADS in "${LEVELS[@]}"; do
        THREADS="${THREADS// /}"
        [[ -z "${THREADS}" || -z "${ROW[${THREADS}]+x}" ]] && continue

        IFS=$'\t' read -r CONC TRACK GOROUTINES OPS TPUT AVGSZ P50 P90 P95 P99 MEAN <<< "${ROW[${THREADS}]}"

        # Format throughput with unit.
        TPUT_FMT="${TPUT} MiB/s"

        # Format avg size in KiB.
        if [[ "${AVGSZ}" =~ ^[0-9]+$ ]]; then
            AVGSZ_FMT="$(( AVGSZ / 1024 )) KiB"
        else
            AVGSZ_FMT="${AVGSZ}"
        fi

        printf '%-12s  %-20s  %10s  %12s  %12s  %10s  %10s  %10s  %10s\n' \
            "${CONC}" "${TRACK}" "${OPS}" "${TPUT_FMT}" "${AVGSZ_FMT}" \
            "${P50}" "${P90}" "${P99}" "${MEAN}"
    done
}

if [[ "${DRY_RUN}" == "true" ]]; then
    echo "  [dry-run] — no benchmark data collected."
    echo ""
    echo "  Commands that would be run:"
    for THREADS in "${LEVELS[@]}"; do
        THREADS="${THREADS// /}"
        [[ -z "${THREADS}" ]] && continue
        echo "    ${GCS_BENCH} bench --config ${CONFIG_FILE} --concurrency ${THREADS} ..."
    done
else
    print_table
    echo ""
    echo "Full results:   ${OUTPUT_ROOT}/"
    echo "Summary TSV:    ${SUMMARY_TSV}"
fi

echo ""
