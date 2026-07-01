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
grep -vE 'github.com/googlecloudplatform/gcsfuse/v3/tools/|github.com/googlecloudplatform/gcsfuse/v3/perfmetrics/|github.com/googlecloudplatform/gcsfuse/v3/benchmarks/|github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake|/mock' "$coverage_txt_path" > "${coverage_txt_path}.tmp" && mv "${coverage_txt_path}.tmp" "$coverage_txt_path"

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

# 5. Generate and upload separate reports for each input directory (Unit, Zonal, Regional)
e2e_dirs=()
for d in "${valid_dirs[@]}"; do
  report_name="dir"
  if [[ "$d" == *unit* ]]; then
    report_name="unit"
  elif [[ "$d" == *zonal* ]]; then
    report_name="zonal"
    e2e_dirs+=("$d")
  elif [[ "$d" == *regional* || "$d" == *non_zonal* ]]; then
    report_name="regional"
    e2e_dirs+=("$d")
  else
    report_name=$(basename "$d")
    e2e_dirs+=("$d")
  fi

  if [ ! -d "$d" ] || [ "$(find "$d" -maxdepth 1 -name "covcounters.*" | wc -l)" -eq 0 ]; then
    echo "Skipping report generation for '$report_name' (directory '$d' is empty or missing)."
    continue
  fi

  echo "Generating coverage report for '$report_name'..."
  temp_text_profile=$(mktemp)
  if go tool covdata textfmt -i="$d" -o="$temp_text_profile" 2>/dev/null; then
    grep -vE 'github.com/googlecloudplatform/gcsfuse/v3/tools/|github.com/googlecloudplatform/gcsfuse/v3/perfmetrics/|github.com/googlecloudplatform/gcsfuse/v3/benchmarks/|github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake|/mock' "$temp_text_profile" > "${temp_text_profile}.tmp" && mv "${temp_text_profile}.tmp" "$temp_text_profile"
    target_html="$reports_dir/${report_name}-coverage.html"
    if command -v go-better-html-coverage &> /dev/null; then
      go-better-html-coverage -n -profile "$temp_text_profile" -o "$target_html" || \
        go tool cover -html="$temp_text_profile" -o "$target_html"
    else
      go tool cover -html="$temp_text_profile" -o "$target_html"
    fi
    echo "👉 Local Click (${report_name}): file://$target_html"

    # Copy to Kokoro artifacts if available
    if [[ -n "${KOKORO_ARTIFACTS_DIR-}" ]]; then
      cp "$target_html" "$KOKORO_ARTIFACTS_DIR/${report_name}-coverage.html"
      if [[ -n "${KOKORO_BUILD_ID-}" ]]; then
        echo "👉 Open Interactive ${report_name} Coverage in Fusion UI (Sponge):"
        echo "   https://sponge.corp.google.com/target?id=${KOKORO_BUILD_ID}&tab=artifacts&file=${report_name}-coverage.html"
      fi
    fi

    # Upload to GCS if requested
    if [[ -n "$GCS_UPLOAD_PATH" ]]; then
      gcloud storage cp "$target_html" "gs://${GCS_UPLOAD_PATH}/${report_name}-coverage.html"
      echo "👉 Open Standalone Interactive ${report_name} Coverage served from GCS:"
      echo "   https://storage.cloud.google.com/${GCS_UPLOAD_PATH}/${report_name}-coverage.html"
    fi
  else
    echo "Warning: Failed to convert binary profiles in '$d' to text format."
  fi
  rm -f "$temp_text_profile"
done

# 6. Generate and upload combined Integration (E2E) report (excluding Unit tests)
if [ ${#e2e_dirs[@]} -gt 0 ]; then
  echo "Generating combined integration (E2E) coverage report..."
  temp_integration_merged_dir=$(mktemp -d)
  temp_integration_text_profile=$(mktemp)
  joined_e2e_dirs=$(IFS=,; echo "${e2e_dirs[*]}")

  if go tool covdata merge -i="$joined_e2e_dirs" -o="$temp_integration_merged_dir" 2>/dev/null && \
     go tool covdata textfmt -i="$temp_integration_merged_dir" -o="$temp_integration_text_profile" 2>/dev/null; then
    grep -vE 'github.com/googlecloudplatform/gcsfuse/v3/tools/|github.com/googlecloudplatform/gcsfuse/v3/perfmetrics/|github.com/googlecloudplatform/gcsfuse/v3/benchmarks/|github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake|/mock' "$temp_integration_text_profile" > "${temp_integration_text_profile}.tmp" && mv "${temp_integration_text_profile}.tmp" "$temp_integration_text_profile"
    target_html="$reports_dir/integration-coverage.html"
    if command -v go-better-html-coverage &> /dev/null; then
      go-better-html-coverage -n -profile "$temp_integration_text_profile" -o "$target_html" || \
        go tool cover -html="$temp_integration_text_profile" -o "$target_html"
    else
      go tool cover -html="$temp_integration_text_profile" -o "$target_html"
    fi
    echo "👉 Local Click (Integration): file://$target_html"

    # Copy to Kokoro artifacts if available
    if [[ -n "${KOKORO_ARTIFACTS_DIR-}" ]]; then
      cp "$target_html" "$KOKORO_ARTIFACTS_DIR/integration-coverage.html"
      if [[ -n "${KOKORO_BUILD_ID-}" ]]; then
        echo "👉 Open Interactive Integration Coverage in Fusion UI (Sponge):"
        echo "   https://sponge.corp.google.com/target?id=${KOKORO_BUILD_ID}&tab=artifacts&file=integration-coverage.html"
      fi
    fi

    # Upload to GCS if requested
    if [[ -n "$GCS_UPLOAD_PATH" ]]; then
      gcloud storage cp "$target_html" "gs://${GCS_UPLOAD_PATH}/integration-coverage.html"
      echo "👉 Open Standalone Interactive Integration Coverage served from GCS:"
      echo "   https://storage.cloud.google.com/${GCS_UPLOAD_PATH}/integration-coverage.html"
    fi
  else
    echo "Warning: Failed to generate combined integration report."
  fi
  rm -rf "$temp_integration_merged_dir" "$temp_integration_text_profile"
fi


# 6. Kokoro Artifacts support for combined reports
KOKORO_DIR_AVAILABLE=false
if [[ -n "${KOKORO_ARTIFACTS_DIR-}" ]]; then
  KOKORO_DIR_AVAILABLE=true
fi

if ${KOKORO_DIR_AVAILABLE}; then
  if ${full_coverage_generated}; then
    echo "Kokoro artifacts path detected. Copying coverage dashboard to target artifacts directory..."
    cp "$coverage_html_path" "$KOKORO_ARTIFACTS_DIR/e2e-coverage.html"
    
    # Route 1: Direct Sponge/Fusion UI link
    if [[ -n "${KOKORO_BUILD_ID-}" ]]; then
      echo "👉 Open Interactive Coverage in Fusion UI (Sponge):"
      echo "   https://sponge.corp.google.com/target?id=${KOKORO_BUILD_ID}&tab=artifacts&file=e2e-coverage.html"
    fi

    # Route 2: Direct GCS link
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

# 7. Upload HTML combined dashboards to custom GCS bucket
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
