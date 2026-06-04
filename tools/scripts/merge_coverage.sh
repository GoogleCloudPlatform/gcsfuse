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

# Helper script to merge different binary code coverage directories, generate
# reports, print Sponge/GCS links on Kokoro, and upload to Codecov.
set -euo pipefail

usage() {
  echo "Usage: $0 --input-dirs=dir1,dir2,... --output-dir=out_dir"
  echo "Options:"
  echo "  --input-dirs          Comma-separated list of directories containing raw binary covcounters and covmeta files."
  echo "  --output-dir          Output directory for final combined text & HTML reports."
  echo "  --gcs-upload-path     Optional. GCS destination path (bucket/prefix) to upload the final HTML reports."
  exit 1
}

INPUT_DIRS=""
OUTPUT_DIR=""
GCS_UPLOAD_PATH=""
CODECOV_TOKEN="${CODECOV_TOKEN:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input-dirs=*)
      INPUT_DIRS="${1#*=}"
      shift
      ;;
    --output-dir=*)
      OUTPUT_DIR="${1#*=}"
      shift
      ;;
    --gcs-upload-path=*)
      GCS_UPLOAD_PATH="${1#*=}"
      shift
      ;;
    *)
      echo "Unknown argument: $1"
      usage
      ;;
  esac
done

if [[ -z "$INPUT_DIRS" || -z "$OUTPUT_DIR" ]]; then
  echo "Error: --input-dirs and --output-dir are required."
  usage
fi

# Split input directories by comma into an array
IFS=',' read -r -a dirs <<< "$INPUT_DIRS"

# Validate that input directories exist and have counters directly inside
valid_dirs=()
for d in "${dirs[@]}"; do
  if [ -d "$d" ] && [ "$(find "$d" -maxdepth 1 -name "covcounters.*" | wc -l)" -gt 0 ]; then
    valid_dirs+=("$d")
  else
    echo "Warning: Directory '$d' does not exist or does not directly contain covcounters.* files. Skipping."
  fi
done

if [ ${#valid_dirs[@]} -eq 0 ]; then
  echo "Error: No valid coverage data directories with files were found!"
  exit 1
fi

joined_dirs=$(IFS=,; echo "${valid_dirs[*]}")
echo "Merging coverage from: $joined_dirs"

# Create output directories
mkdir -p "$OUTPUT_DIR/merged" "$OUTPUT_DIR/reports"
local_merged_dir="$OUTPUT_DIR/merged"
reports_dir="$OUTPUT_DIR/reports"
coverage_txt_path="$reports_dir/combined_coverage.out"
coverage_html_path="$reports_dir/e2e-coverage.html"
diff_html_path="$reports_dir/e2e-diff-coverage.html"

# 1. Merge binary profiles
go tool covdata merge -i="$joined_dirs" -o="$local_merged_dir"

# 2. Convert to text coverprofile format
go tool covdata textfmt -i="$local_merged_dir" -o="$coverage_txt_path"

# 3. Print functional summary to stdout
echo "------------------------------------------"
echo "Functional Coverage Summary:"
go tool cover -func="$coverage_txt_path"
echo "------------------------------------------"

# 4. Generate visual interactive HTML dashboard
full_coverage_generated=false
diff_coverage_generated=false

# Dynamically install go-better-html-coverage if missing
if ! command -v go-better-html-coverage &> /dev/null; then
  echo "go-better-html-coverage not found. Attempting to install..."
  go install github.com/chmouel/go-better-html-coverage@latest || {
    echo "Warning: Failed to install go-better-html-coverage. Fallback to standard cover tool."
  }
fi

if command -v go-better-html-coverage &> /dev/null; then
  echo "Generating visual interactive HTML browser dashboard..."
  if go-better-html-coverage -n -profile "$coverage_txt_path" -o "$coverage_html_path"; then
    full_coverage_generated=true
    echo "Functional coverage dashboard compiled successfully!"
    echo "👉 Local Click (Full): file://$coverage_html_path"
  else
    echo "Failed to generate visual HTML dashboard using go-better-html-coverage."
  fi

  # Generate diff-coverage dashboard
  base_ref=""
  if git rev-parse --verify master &>/dev/null; then
    base_ref="master"
  elif git rev-parse --verify origin/master &>/dev/null; then
    base_ref="origin/master"
  fi

  if [[ -n "$base_ref" ]]; then
    echo "Generating git diff-coverage dashboard against '$base_ref'..."
    if go-better-html-coverage -n -ref "$base_ref" -profile "$coverage_txt_path" -o "$diff_html_path"; then
      diff_coverage_generated=true
      echo "👉 Local Click (Diff): file://$diff_html_path"
    else
      echo "Failed to generate diff-coverage report."
    fi
  fi
else
  # Fallback to standard go tool cover
  go tool cover -html="$coverage_txt_path" -o "$coverage_html_path"
  full_coverage_generated=true
  echo "👉 Local Click (Full, Standard): file://$coverage_html_path"
fi

# 5. Kokoro Artifacts support
KOKORO_DIR_AVAILABLE=false
if [[ -n "${KOKORO_ARTIFACTS_DIR-}" ]]; then
  KOKORO_DIR_AVAILABLE=true
fi

if ${KOKORO_DIR_AVAILABLE}; then
  if ${full_coverage_generated}; then
    echo "Kokoro artifacts path detected. Copying coverage dashboard to target artifacts directory..."
    cp "$coverage_html_path" "$KOKORO_ARTIFACTS_DIR/e2e-coverage.html"
    
    # Route 1: Direct Sponge/Fusion UI dynamic link matching Kokoro Run UUID
    if [[ -n "${KOKORO_BUILD_ID-}" ]]; then
      echo "👉 Open Interactive Coverage in Fusion UI (Sponge):"
      echo "   https://sponge.corp.google.com/target?id=${KOKORO_BUILD_ID}&tab=artifacts&file=e2e-coverage.html"
    fi

    # Route 2: Direct GCS Corp-authenticated static dynamic link
    if [[ -n "${KOKORO_ARTIFACTS_GCS_PATH-}" ]]; then
      gcs_http_path="${KOKORO_ARTIFACTS_GCS_PATH#gs://}"
      echo "👉 Open Standalone Coverage served from Google Cloud Storage:"
      echo "   https://storage.cloud.google.com/${gcs_http_path}/e2e-coverage.html"
    fi
  fi

  if ${diff_coverage_generated}; then
    echo "Copying diff-coverage dashboard to target artifacts directory..."
    cp "$diff_html_path" "$KOKORO_ARTIFACTS_DIR/e2e-diff-coverage.html"
    
    if [[ -n "${KOKORO_BUILD_ID-}" ]]; then
      echo "👉 Open Interactive Diff-Coverage in Fusion UI (Sponge):"
      echo "   https://sponge.corp.google.com/target?id=${KOKORO_BUILD_ID}&tab=artifacts&file=e2e-diff-coverage.html"
    fi

    if [[ -n "${KOKORO_ARTIFACTS_GCS_PATH-}" ]]; then
      gcs_http_path="${KOKORO_ARTIFACTS_GCS_PATH#gs://}"
      echo "👉 Open Standalone Diff-Coverage served from Google Cloud Storage:"
      echo "   https://storage.cloud.google.com/${gcs_http_path}/e2e-diff-coverage.html"
    fi
  fi
fi

# 6. Upload HTML dashboard to custom GCS bucket
if [[ -n "$GCS_UPLOAD_PATH" ]]; then
  echo "Uploading HTML dashboards to Google Cloud Storage..."
  if ${full_coverage_generated}; then
    gcloud storage cp "$coverage_html_path" "gs://${GCS_UPLOAD_PATH}/e2e-coverage.html"
    echo "👉 Open Standalone Interactive Coverage served from GCS:"
    echo "   https://storage.cloud.google.com/${GCS_UPLOAD_PATH}/e2e-coverage.html"
  fi
  if ${diff_coverage_generated}; then
    gcloud storage cp "$diff_html_path" "gs://${GCS_UPLOAD_PATH}/e2e-diff-coverage.html"
    echo "👉 Open Standalone Interactive Diff-Coverage served from GCS:"
    echo "   https://storage.cloud.google.com/${GCS_UPLOAD_PATH}/e2e-diff-coverage.html"
  fi
fi

# 7. Upload to Codecov
if [[ -n "$CODECOV_TOKEN" ]]; then
  echo "Uploading coverage profile to Codecov..."
  curl -Os https://uploader.codecov.io/latest/linux/codecov
  chmod +x codecov
  ./codecov -t "$CODECOV_TOKEN" -f "$coverage_txt_path" -F combined_suite
  rm -f codecov
fi

echo "Coverage merge process complete!"
