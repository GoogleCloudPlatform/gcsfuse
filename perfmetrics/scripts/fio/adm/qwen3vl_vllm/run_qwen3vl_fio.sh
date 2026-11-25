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

# ADM fio replay v3: Qwen3-VL-235B-A22B-Instruct on vanilla vLLM v0.16 TP=8.
# Replays the vLLM weight load against a GCSFuse mount, one model file at a time in load order. For each file,
# 8 fio jobs (= TP ranks) replay per-rank iologs made by gen_vllm_iologs.py from the safetensors header.
#
# Usage:
#   sudo ./run_qwen3vl_fio.sh <model_dir_on_mount> <read_ahead_kb> <out_dir> [file numbers... | all]
#     model_dir_on_mount : e.g. /mnt/gcs/Qwen3-VL-235B (contains model-000NN-of-00096.safetensors)
#     read_ahead_kb      : 0 (the GPU trace setting; 1 also rounds to 0 pages), 128 (kernel default), ...
#     file numbers       : default "1 2 3 4 95 96"; "all" = the full 96-file load (~22 h at read_ahead 0).
# Env:
#   FIO=/path/to/fio (stock fio >= 3.36 is fine for v3), PY=python3 with numpy,
#   METRICS_URL=http://localhost:9091/metrics to snapshot GCSFuse Prometheus counters after every file,
#   WORK=<dir for iologs> (default <out_dir>/iolog; ~0.7 GB per layer file, deleted after use).
#
# Mount like the GPU run (kernel reader off, infinite metadata TTL, no file cache) plus a wire log:
#   gcsfuse --wire-log=<out>/wirelog.json --implicit-dirs --metadata-cache-ttl-secs -1 \
#           --file-cache-max-size-mb 0 [--prometheus-port 9091] <bucket> /mnt/gcs
# GCSFuse version: the GPU trace used v3.11.2 (inferred). Pin whatever you compare against.
set -euo pipefail

MODEL_DIR=${1:?model dir}
RA_KB=${2:?read_ahead_kb}
OUT=${3:?out dir}
shift 3
FILES=("$@")
[[ ${#FILES[@]} -eq 0 ]] && FILES=(1 2 3 4 95 96)
[[ "${FILES[0]}" == "all" ]] && FILES=($(seq 1 96))

HERE=$(cd "$(dirname "$0")" && pwd)
JOB=${JOB:-$HERE/qwen3vl_vllm_tp8.fio}
GEN=$HERE/gen_vllm_iologs.py
HDRS=$HERE/headers
FIO=${FIO:-fio}
PY=${PY:-python3}
WORK=${WORK:-$OUT/iolog}
mkdir -p "$OUT" "$WORK"

fpath() { printf "%s/model-%05d-of-00096.safetensors" "$MODEL_DIR" "$1"; }
hdr_for() {  # files 2..94 share one layout (checked against the real file size below)
  local h; h=$(printf "%s/hdr_%05d.json" "$HDRS" "$1")
  if [[ -f $h ]]; then echo "$h"; elif (( $1 >= 2 && $1 <= 94 )); then echo "$HDRS/hdr_00002.json"; else
    echo "no header for file $1" >&2; return 1; fi
}
gen() {  # <file_no>: iologs into $WORK/<n>, and a size check against the file on the mount (stat only)
  local n=$1 d="$WORK/$1"
  "$PY" "$GEN" --header "$(hdr_for "$n")" --path "$(fpath "$n")" --out "$d" > "$d.gen.log" 2>&1
  local want got
  want=$("$PY" -c "import json,sys;print(json.load(open(sys.argv[1]))['file_end'])" "$d/summary.json")
  got=$(stat -c %s "$(fpath "$n")")
  if [[ $want != "$got" ]]; then echo "file $n: size $got != header layout end $want" >&2; return 1; fi
}

MNT=$(df --output=target "$MODEL_DIR" | tail -n 1)
BDI=$(mountpoint -d "$MNT")
{
  echo "mount=$MNT bdi=$BDI read_ahead_kb=$RA_KB files=${FILES[*]}"
  gcsfuse --version 2>/dev/null | head -n 1 || true
  "$FIO" --version
  uname -r
} | tee "$OUT/run.txt"

# Cold start: drop the page cache once (vLLM does not drop between files), then set read-ahead.
sync; echo 3 > /proc/sys/vm/drop_caches
echo "$RA_KB" > "/sys/class/bdi/$BDI/read_ahead_kb"
echo "effective read_ahead_kb=$(cat "/sys/class/bdi/$BDI/read_ahead_kb")" | tee -a "$OUT/run.txt"

echo "file,start_s,end_s" > "$OUT/files.csv"
[[ -n "${METRICS_URL:-}" ]] && { curl -s "$METRICS_URL" > "$OUT/metrics_start.txt" || true; }
gen "${FILES[0]}"
for i in "${!FILES[@]}"; do
  n=${FILES[$i]}
  next=${FILES[$((i + 1))]:-}
  gpid=""
  if [[ -n $next ]]; then gen "$next" & gpid=$!; fi        # generate the next file's iologs meanwhile
  nn=$(printf %02d "$n")
  t0=$(date +%s.%N)
  IOLOG_DIR="$WORK/$n" "$FIO" "$JOB" --output-format=json --output="$OUT/f${nn}.json"
  t1=$(date +%s.%N)
  printf "%d,%.3f,%.3f\n" "$n" "$t0" "$t1" | tee -a "$OUT/files.csv"
  [[ -n "${METRICS_URL:-}" ]] && { curl -s "$METRICS_URL" > "$OUT/metrics_f${nn}.txt" || true; }
  rm -rf "${WORK:?}/$n"
  [[ -n $gpid ]] && wait "$gpid"
done
echo "done: $OUT"
