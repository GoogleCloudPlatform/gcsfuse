#!/usr/bin/env bash
# run-distributed-per-host.sh — Multi-host gcs-bench with per-host corpora.
#
# KEY DIFFERENCES from run-distributed.sh:
#
#   1. EACH host writes its OWN full corpus (no modulo sharding of objects).
#      Objects land under OBJECT_PREFIX_BASE/host-N/ so namespaces never
#      collide.  With 8 workers and 50,176 objects each, the total dataset
#      is 401,408 objects across 8 independent subdirectories.
#
#   2. No shared NFS required.  Each worker uploads its bench.yaml to a GCS
#      path (RESULTS_GCS_PATH) after the run.  Worker 0 polls that path, then
#      downloads all YAMLs and runs merge-results locally.
#
#   3. Output format is "both" (YAML + TSV + bench.txt + .hgrm files written
#      to the local per-worker output directory on every run).
#
# ---------------------------------------------------------------------------
# TWO-PHASE USAGE
# ---------------------------------------------------------------------------
#
# Phase 1 — Prepare (populate data, run on ALL worker hosts simultaneously):
#
#   PHASE=prepare \
#   WORKER_ID=0 \
#   NUM_WORKERS=8 \
#   OBJECT_PREFIX_BASE=unet3d/ \
#   GCS_BENCH=./gcs-bench \
#     ./examples/scripts/run-distributed-per-host.sh \
#     --config examples/benchmark-configs/unet3d-like-prepare.yaml
#
#   (Repeat on each host with WORKER_ID=1 ... WORKER_ID=7.)
#   No synchronization needed — each host writes independently.
#
# Phase 2 — Benchmark (run on ALL worker hosts simultaneously):
#
#   PHASE=bench \
#   WORKER_ID=0 \
#   NUM_WORKERS=8 \
#   OBJECT_PREFIX_BASE=unet3d/ \
#   RESULTS_GCS_PATH=gs://signal65-rapid1/bench-results \
#   GCS_BENCH=./gcs-bench \
#     ./examples/scripts/run-distributed-per-host.sh \
#     --config examples/benchmark-configs/unet3d-like.yaml
#
#   Worker 0 also acts as coordinator: after all workers finish it downloads
#   all YAMLs from GCS and calls gcs-bench merge-results automatically.
#
# ---------------------------------------------------------------------------
# ENVIRONMENT VARIABLES
# ---------------------------------------------------------------------------
#
#   PHASE               "prepare" | "bench" (default: bench)
#   WORKER_ID           0-based index of this host (default: 0)
#   NUM_WORKERS         Total number of worker hosts (default: 8)
#   OUTPUT_DIR          Local directory for result files (default: ./results)
#   GCS_BENCH           Path to the gcs-bench binary (default: gcs-bench)
#   OBJECT_PREFIX_BASE  GCS prefix for test data, without trailing host-N/
#                       (default: unet3d/)
#                       Each host writes to OBJECT_PREFIX_BASE/host-N/
#   RESULTS_GCS_PATH    GCS path for worker result YAML upload/download
#                       No trailing slash. (default: gs://my-bucket/bench-results)
#                       Only used in bench phase. Must be writable by all hosts.
#   START_DELAY_SECS    Seconds ahead to schedule the synchronized start.
#                       Must be long enough for all hosts to launch and pre-fill
#                       the write pool (~8 GiB of random data).
#                       (default: 90)
#
# ---------------------------------------------------------------------------
# REQUIREMENTS
# ---------------------------------------------------------------------------
#   • gcs-bench binary on PATH or set GCS_BENCH=./gcs-bench
#   • gsutil on PATH (bench phase only — for YAML upload/download)
#   • Google Cloud credentials with:
#       - read/write on the benchmark bucket (both phases)
#       - read/write on RESULTS_GCS_PATH bucket (bench phase)
#   • NTP-synchronized clocks across all hosts (~1 s accuracy is sufficient;
#     the warmup period absorbs any remaining skew)
# ---------------------------------------------------------------------------

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PHASE="${PHASE:-gentime}"
WORKER_ID="${WORKER_ID:-0}"
NUM_WORKERS="${NUM_WORKERS:-8}"
OUTPUT_DIR="${OUTPUT_DIR:-$(pwd)/results}"
GCS_BENCH="${GCS_BENCH:-gcs-bench}"
LAYOUT="${LAYOUT:-sharded}"           # "sharded" | "per-host"
OBJECT_PREFIX_BASE="${OBJECT_PREFIX_BASE:-unet3d/}"
RESULTS_GCS_PATH="${RESULTS_GCS_PATH:-gs://my-bucket/bench-results}"
START_DELAY_SECS="${START_DELAY_SECS:-300}"

# Parse --config from the script's own arguments (gentime phase needs it to
# embed the path in the generated commands; all other phases pass "$@" directly
# to gcs-bench so --config flows through automatically).
CONFIG_ARG=""
for _arg in "$@"; do
    if [[ "${_arg}" == --config=* ]]; then
        CONFIG_ARG="${_arg}"
        break
    fi
done
# Handle split form: --config path/to/file.yaml
prev=""
for _arg in "$@"; do
    if [[ "${prev}" == "--config" ]]; then
        CONFIG_ARG="--config ${_arg}"
        break
    fi
    prev="${_arg}"
done
unset prev _arg

# ===========================================================================
# GENTIME phase — print a ready-to-copy START_AT epoch, then exit.
# Usage:
#   START_DELAY_SECS=600 PHASE=gentime ./run-distributed-per-host.sh
# The epoch will be exactly START_DELAY_SECS seconds from now (default 90 s,
# so override to e.g. 600 for 10 minutes as shown above).
# ===========================================================================
if [[ "${PHASE}" == "gentime" ]]; then
    epoch=$(( $(date +%s) + START_DELAY_SECS ))
    human=$(date -d "@${epoch}" --utc '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
            || date -r "${epoch}" -u '+%Y-%m-%dT%H:%M:%SZ')
    script=$(realpath "$0")

    # Require --config before printing anything useful.
    if [[ -z "${CONFIG_ARG}" ]]; then
        echo "NOTE: No --config specified."
        echo ""
        echo "Re-run with your YAML config to generate ready-to-paste host commands."
        echo ""
        echo "Sharded prepare (each host writes 1/N of the objects into a shared prefix):"
        echo "  ./$(basename "$0") --config path/to/prepare-config.yaml"
        echo ""
        echo "Per-host prepare (each host writes a full independent corpus under host-N/):"
        echo "  LAYOUT=per-host ./$(basename "$0") --config path/to/prepare-config.yaml"
        echo ""
        echo "Bench (reads — use your bench YAML, same LAYOUT as prepare):"
        echo "  ./$(basename "$0") --config path/to/bench-config.yaml"
        echo "  LAYOUT=per-host ./$(basename "$0") --config path/to/bench-config.yaml"
        exit 0
    fi
    config_arg=" ${CONFIG_ARG}"

    # Extract the config file path from CONFIG_ARG (handles both
    # "--config file.yaml" and "--config=file.yaml" forms).
    config_file="${CONFIG_ARG#--config}"
    config_file="${config_file#=}"
    config_file="${config_file# }"

    # Parse bucket and object-prefix from the YAML (grep + sed; no yq needed).
    # Lines look like:  bucket: "sig65-rapid1"     # optional comment
    #                   object-prefix: "gcsfuse/unet3d/"
    _strip_value() {
        sed 's/^\s*[^:]*:\s*//' \
        | sed 's/\s*#.*//' \
        | tr -d '"' | tr -d "'" | tr -d ' '
    }

    yaml_bucket=$(grep -m1 '^\s*bucket:' "${config_file}" 2>/dev/null | _strip_value || true)
    yaml_prefix=$(grep -m1 '^\s*object-prefix:' "${config_file}" 2>/dev/null | _strip_value || true)
    yaml_mode=$(grep -m1 '^\s*mode:' "${config_file}" 2>/dev/null | _strip_value || true)

    # Derive PHASE from the YAML mode field.
    # mode: prepare  → PHASE=prepare
    # mode: benchmark (or absent) → PHASE=bench
    yaml_phase="bench"
    if [[ "${yaml_mode}" == "prepare" ]]; then
        yaml_phase="prepare"
    fi
    echo "INFO: mode          → PHASE=${yaml_phase} (YAML mode: ${yaml_mode:-benchmark})"
    echo "INFO: layout        → LAYOUT=${LAYOUT}"

    # In per-host mode, OBJECT_PREFIX_BASE is the base that host-N/ is appended
    # to.  Populate it from the YAML if the user has not overridden it.
    if [[ "${LAYOUT}" == "per-host" ]]; then
        if [[ -n "${yaml_prefix}" && "${OBJECT_PREFIX_BASE}" == "unet3d/" ]]; then
            OBJECT_PREFIX_BASE="${yaml_prefix}"
        fi
        echo "INFO: object-prefix → OBJECT_PREFIX_BASE=${OBJECT_PREFIX_BASE} (host-N/ appended per host)"
    else
        echo "INFO: object-prefix → from YAML config (${yaml_prefix:-not found})"
    fi

    # Override RESULTS_GCS_PATH bucket segment from the YAML (if still default).
    if [[ -n "${yaml_bucket}" && "${RESULTS_GCS_PATH}" == "gs://my-bucket/bench-results" ]]; then
        RESULTS_GCS_PATH="gs://${yaml_bucket}/bench-results"
        echo "INFO: bucket        → RESULTS_GCS_PATH=${RESULTS_GCS_PATH} (from YAML)"
    fi
    echo ""

    echo "Scheduled start (UTC): ${human}"
    echo ""
    echo "Run each line on the corresponding host (only WORKER_ID differs):"
    echo ""

    for i in $(seq 0 $(( NUM_WORKERS - 1 ))); do
        echo "  # host ${i}"
        # Build env prefix — only include OBJECT_PREFIX_BASE in per-host mode.
        cmd="START_AT=${epoch} PHASE=${yaml_phase} WORKER_ID=${i} NUM_WORKERS=${NUM_WORKERS} LAYOUT=${LAYOUT}"
        if [[ "${LAYOUT}" == "per-host" ]]; then
            cmd="${cmd} OBJECT_PREFIX_BASE=${OBJECT_PREFIX_BASE}"
        fi
        cmd="${cmd} RESULTS_GCS_PATH=${RESULTS_GCS_PATH} GCS_BENCH=${GCS_BENCH} ${script}${config_arg}"
        echo "  ${cmd}"
        echo ""
    done
    exit 0
fi

# ---------------------------------------------------------------------------
# Derived values
# ---------------------------------------------------------------------------
# Each host owns a unique subdirectory so object names never collide.
HOST_PREFIX="${OBJECT_PREFIX_BASE}host-${WORKER_ID}/"

WORKER_OUTPUT="${OUTPUT_DIR}/worker-${WORKER_ID}"
mkdir -p "${WORKER_OUTPUT}"

# LAYOUT controls how objects are distributed across hosts:
#   sharded   — all hosts share the YAML object-prefix; gcs-bench distributes
#               objects via "i % NumWorkers == WorkerID" during prepare, and
#               reads from the shared corpus during bench.
#   per-host  — each host writes/reads its own HOST_PREFIX subdirectory with
#               a complete (unsharded) object list.
if [[ "${LAYOUT}" == "per-host" ]]; then
    EFFECTIVE_PREFIX="${HOST_PREFIX}"
    LAYOUT_LABEL="per-host corpus → ${HOST_PREFIX}"
else
    EFFECTIVE_PREFIX=""   # do not override; YAML object-prefix flows through
    LAYOUT_LABEL="sharded (shared prefix from YAML)"
fi

echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Phase:         ${PHASE}"
echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Layout:        ${LAYOUT_LABEL}"
echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Local output:  ${WORKER_OUTPUT}"

# ===========================================================================
# PREPARE phase
# ===========================================================================
# sharded:   pass --worker-id / --num-workers so the engine writes only this
#            host's shard (i % NumWorkers == WorkerID) into the shared prefix.
# per-host:  no sharding flags; this host writes the full object list under
#            its own HOST_PREFIX.
#
# Neither mode uses --start-at; prepare hosts run at their own pace.
# ===========================================================================
if [[ "${PHASE}" == "prepare" ]]; then
    echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Starting prepare — ${LAYOUT_LABEL} ..."

    # Build prepare arguments based on layout.
    prepare_args=(
        --output-path   "${WORKER_OUTPUT}"
        --output-format both
    )
    if [[ "${LAYOUT}" == "per-host" ]]; then
        # Full corpus under this host's unique prefix; no worker sharding.
        prepare_args+=( --object-prefix "${EFFECTIVE_PREFIX}" )
        echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Writing full corpus to: ${EFFECTIVE_PREFIX}"
    else
        # Sharded: engine writes only objects where i % NumWorkers == WorkerID.
        prepare_args+=( --worker-id "${WORKER_ID}" --num-workers "${NUM_WORKERS}" )
        echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Writing shard ${WORKER_ID}/${NUM_WORKERS} to prefix from YAML config"
    fi

    "${GCS_BENCH}" bench "${prepare_args[@]}" "$@"

    echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Prepare complete."
    exit 0
fi

# ===========================================================================
# BENCH phase
# ===========================================================================
# Each host reads from its own HOST_PREFIX corpus.  --worker-id / --num-workers
# are passed so the engine embeds raw HDR histogram snapshots in bench.yaml,
# enabling statistically correct merge across all workers.
#
# Start time is computed locally.  If all hosts are launched within a few
# seconds of each other (normal for scripted deployment), the START_DELAY_SECS
# gap ensures every host is running before the measurement window opens.
# The warmup period absorbs any remaining clock skew.
# ===========================================================================

# Compute (or accept) a synchronized start time.
# If START_AT is already set in the environment (Unix epoch), use it directly so
# every host starts at the exact same wall-clock time regardless of launch order.
# Otherwise fall back to NOW + START_DELAY_SECS.
if [[ -z "${START_AT:-}" ]]; then
    START_AT=$(( $(date +%s) + START_DELAY_SECS ))
fi
echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Synchronized start at: $(
    date -d "@${START_AT}" --utc '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
    || date -r "${START_AT}" -u '+%Y-%m-%dT%H:%M:%SZ'
)"

echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Starting benchmark ..."

bench_args=(
    --worker-id     "${WORKER_ID}"
    --num-workers   "${NUM_WORKERS}"
    --start-at      "${START_AT}"
    --output-path   "${WORKER_OUTPUT}"
    --output-format both
)
if [[ "${LAYOUT}" == "per-host" ]]; then
    # Override prefix so each host reads from its own corpus.
    bench_args+=( --object-prefix "${EFFECTIVE_PREFIX}" )
fi

"${GCS_BENCH}" bench "${bench_args[@]}" "$@"

echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Benchmark complete. Local results in ${WORKER_OUTPUT}/"

# ---------------------------------------------------------------------------
# Upload this worker's bench.yaml to GCS so the coordinator can merge without
# needing a shared NFS mount.  Only the YAML is needed for merging; the full
# set of local output files (bench.txt, bench.tsv, .hgrm) remains on disk.
# ---------------------------------------------------------------------------
result_yaml=$(ls -t "${WORKER_OUTPUT}"/bench-*/bench.yaml 2>/dev/null | head -1 || true)
if [[ -z "${result_yaml}" ]]; then
    echo "[worker ${WORKER_ID}/${NUM_WORKERS}] WARNING: no bench.yaml found in ${WORKER_OUTPUT} — skipping upload" >&2
else
    gcs_dest="${RESULTS_GCS_PATH}/worker-${WORKER_ID}.yaml"
    echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Uploading ${result_yaml} → ${gcs_dest}"
    gsutil cp "${result_yaml}" "${gcs_dest}"
    echo "[worker ${WORKER_ID}/${NUM_WORKERS}] Upload complete."
fi

# ===========================================================================
# Coordinator (worker 0 only): wait for all workers, download YAMLs, merge.
# ===========================================================================
if [[ "${WORKER_ID}" -eq 0 && "${NUM_WORKERS}" -gt 1 ]]; then
    echo "[coordinator] Waiting for all ${NUM_WORKERS} worker YAMLs at ${RESULTS_GCS_PATH}/ ..."

    TIMEOUT_SECS=600
    POLL_INTERVAL=10
    WAITED=0

    while true; do
        found=0
        for i in $(seq 0 $(( NUM_WORKERS - 1 ))); do
            if gsutil -q stat "${RESULTS_GCS_PATH}/worker-${i}.yaml" 2>/dev/null; then
                found=$(( found + 1 ))
            fi
        done

        if [[ "${found}" -eq "${NUM_WORKERS}" ]]; then
            echo "[coordinator] All ${NUM_WORKERS} result files present."
            break
        fi

        if [[ "${WAITED}" -ge "${TIMEOUT_SECS}" ]]; then
            echo "[coordinator] ERROR: timed out after ${TIMEOUT_SECS}s — only ${found}/${NUM_WORKERS} files found." >&2
            exit 1
        fi

        echo "[coordinator] ${found}/${NUM_WORKERS} workers done — waiting ${POLL_INTERVAL}s ..."
        sleep "${POLL_INTERVAL}"
        WAITED=$(( WAITED + POLL_INTERVAL ))
    done

    # Download all worker YAMLs to a local staging directory.
    LOCAL_MERGE_DIR="${OUTPUT_DIR}/merge-inputs"
    mkdir -p "${LOCAL_MERGE_DIR}"
    echo "[coordinator] Downloading worker YAMLs from GCS ..."
    gsutil -m cp "${RESULTS_GCS_PATH}/worker-*.yaml" "${LOCAL_MERGE_DIR}/"

    # Produce the merged summary.
    MERGED_OUTPUT="${OUTPUT_DIR}/merged"
    mkdir -p "${MERGED_OUTPUT}"
    echo "[coordinator] Merging ${NUM_WORKERS} worker results ..."
    "${GCS_BENCH}" merge-results \
        --output-path   "${MERGED_OUTPUT}" \
        --output-format both \
        "${LOCAL_MERGE_DIR}"/worker-*.yaml

    echo "[coordinator] Merged results written to ${MERGED_OUTPUT}/"
    echo "[coordinator] Plot combined latency distribution with:"
    echo "  ${GCS_BENCH} plot-hgrm ${MERGED_OUTPUT}/bench-*/*-total-latency.hgrm"
fi
