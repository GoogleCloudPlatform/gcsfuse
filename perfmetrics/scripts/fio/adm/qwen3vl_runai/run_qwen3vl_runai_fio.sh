#!/bin/bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# ADM fio replay: Qwen3-VL-235B-A22B-Instruct on vLLM v0.16 + RunAI Model Streamer 0.15.6 TP=8.
# Replays one or more 40 GB batches against a GCSFuse mount, enforcing the batch barrier between batches.
#
# Usage:
#   sudo ./run_qwen3vl_runai_fio.sh <model_dir_on_mount> <read_ahead_kb> <out_dir> [batch numbers... | all]
#     model_dir_on_mount : e.g. /mnt/gcs/Qwen3-VL-235B (contains model-000NN-of-00096.safetensors; no spaces)
#     read_ahead_kb      : 128 (kernel default, matches full RunAI run), 0 (matches read-ahead-off run), ...
#     batch numbers      : default "1" (~39 GB, ~20 min cross-region — enough to verify GCSFuse Random/5.7x).
#                          Use "all" for all 12 batches (471.34 GB, ~4 h cross-region).
# Env:
#   FIO=/path/to/fio (stock fio >= 3.36 works; no patch needed), PY=python3,
#   METRICS_URL=http://localhost:9091/metrics to snapshot GCSFuse Prometheus counters after every batch,
#   HEADERS_DIR=<dir with hdr_00001/2/95/96.json> (default ./headers).
#
# Mount like the RunAI run (kernel reader off, infinite metadata TTL, no file cache), with wire log + metrics:
#   gcsfuse --wire-log=<out>/wirelog.json --implicit-dirs --metadata-cache-ttl-secs -1 \
#           --file-cache-max-size-mb 0 --prometheus-port 9091 <bucket> /mnt/gcs
set -euo pipefail

MODEL_DIR=${1:?model dir}
RA_KB=${2:?read_ahead_kb}
OUT=${3:?out dir}
shift 3
BATCHES=("$@")
[[ ${#BATCHES[@]} -eq 0 ]] && BATCHES=(1)
[[ "${BATCHES[0]}" == "all" ]] && BATCHES=($(seq 1 12))

HERE=$(cd "$(dirname "$0")" && pwd)
JOB=${JOB:-$HERE/qwen3vl_runai_tp8.fio}
GEN=$HERE/gen_runai_iologs.py
HEADERS_DIR=${HEADERS_DIR:-$HERE/headers}
FIO=${FIO:-fio}
PY=${PY:-python3}
IOLOG_ROOT=$OUT/iolog
mkdir -p "$OUT" "$IOLOG_ROOT"

MNT=$(df --output=target "$MODEL_DIR" | tail -n 1)
BDI=$(mountpoint -d "$MNT")
{
  echo "mount=$MNT bdi=$BDI read_ahead_kb=$RA_KB batches=${BATCHES[*]}"
  gcsfuse --version 2>/dev/null | head -n 1 || true
  "$FIO" --version
  uname -r
} | tee "$OUT/run.txt"

# Generate iologs for the requested batches (<2 MB per batch, <1 s).
BCSV=$(IFS=,; echo "${BATCHES[*]}")
"$PY" "$GEN" --model-dir "$MODEL_DIR" --headers-dir "$HEADERS_DIR" --out "$IOLOG_ROOT" --batches "$BCSV" \
  | tee "$OUT/batches_plan.txt"

# Cold start: drop the page cache once (RunAI does not drop between batches), then set read-ahead.
sync; echo 3 > /proc/sys/vm/drop_caches
echo "$RA_KB" > "/sys/class/bdi/$BDI/read_ahead_kb"
echo "effective read_ahead_kb=$(cat "/sys/class/bdi/$BDI/read_ahead_kb")" | tee -a "$OUT/run.txt"

echo "batch,start_s,end_s" > "$OUT/batches.csv"
[[ -n "${METRICS_URL:-}" ]] && { curl -s "$METRICS_URL" > "$OUT/metrics_start.txt" || true; }
for b in "${BATCHES[@]}"; do
  bb=$(printf %02d "$b")
  t0=$(date +%s.%N)
  BATCH_DIR="$IOLOG_ROOT/batch_${bb}" "$FIO" "$JOB" --output-format=json --output="$OUT/b${bb}.json"
  t1=$(date +%s.%N)
  printf "%d,%.3f,%.3f\n" "$b" "$t0" "$t1" | tee -a "$OUT/batches.csv"
  [[ -n "${METRICS_URL:-}" ]] && { curl -s "$METRICS_URL" > "$OUT/metrics_b${bb}.txt" || true; }
done
echo "done: $OUT"
