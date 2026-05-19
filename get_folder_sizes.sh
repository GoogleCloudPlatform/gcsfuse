#!/usr/bin/env bash

# Exit on error
set -e

BUCKET="mohits-checkpoint-deletion-data-main"
echo "Fetching top-level prefixes for bucket: gs://$BUCKET"

# 1. Get list of top-level prefixes
prefixes=$(gcloud storage ls "gs://$BUCKET/" | grep -E 'gs://[^/]+/[^/]+/' || true)

if [[ -z "$prefixes" ]]; then
  echo "No top-level folders found."
  exit 0
fi

echo "Calculating sizes in parallel..."
echo "-------------------------------------------"
printf "%-45s %s\n" "FOLDER" "SIZE"
echo "-------------------------------------------"

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

# 2. Query sizes in parallel using background jobs
declare -A pids

for prefix_url in $prefixes; do
  folder_name=$(basename "$prefix_url")
  
  # Run gcloud storage du -s in background and output to a temp file
  gcloud storage du -s "$prefix_url" > "$temp_dir/$folder_name" 2>/dev/null &
  pids["$folder_name"]=$!
done

# 3. Wait for all jobs to finish
for folder_name in "${!pids[@]}"; do
  wait "${pids[$folder_name]}" || true
done

# 4. Process and display results
total_bytes=0

# Sort directories naturally
for prefix_url in $(echo "$prefixes" | sort -V); do
  folder_name=$(basename "$prefix_url")
  result_file="$temp_dir/$folder_name"
  
  if [[ -f "$result_file" && -s "$result_file" ]]; then
    bytes=$(sed -E 's/^([0-9]+).*/\1/' "$result_file")
    if [[ "$bytes" =~ ^[0-9]+$ ]]; then
      total_bytes=$((total_bytes + bytes))
      human_size=$(numfmt --to=iec-i --suffix=B "$bytes")
      printf "%-45s %s\n" "$folder_name/" "$human_size"
    else
      printf "%-45s %s\n" "$folder_name/" "Error: Got '$bytes'"
    fi
  else
    printf "%-45s %s\n" "$folder_name/" "Empty or Error"
  fi
done

echo "-------------------------------------------"
total_human=$(numfmt --to=iec-i --suffix=B "$total_bytes")
printf "%-45s %s\n" "TOTAL BUCKET SIZE" "$total_human"
echo "-------------------------------------------"


# =============================================================
# GCS BUCKET SIZE REPORT: gs://mohits-checkpoint-deletion-data-main
# =============================================================

# FOLDER                                        SIZE
# -------------------------------------------------------------
# 100/                                          10 TiB
# 300/                                          10 TiB
# 500/                                          10 TiB
# 600/                                          5.0 TiB
# 700/                                          10 TiB
# 900/                                          10 TiB
# 1100/                                         10 TiB
# 1300/                                         10 TiB
# 1500/                                         10 TiB
# 1700/                                         10 TiB
# 1900/                                         10 TiB
# 2000.orbax-checkpoint-tmp/                    10 TiB
# 2400/                                         10 TiB
# 2600/                                         10 TiB
# 2800/                                         10 TiB
# 3000/                                         10 TiB
# 3200/                                         10 TiB
# 3400/                                         10 TiB
# 3600/                                         10 TiB
# 3800/                                         10 TiB
# 4000/                                         10 TiB
# 4200/                                         10 TiB
# 4400/                                         10 TiB
# 4500/                                         10 TiB
# 4600.orbax-checkpoint-tmp/                    10 TiB
# 4800/                                         10 TiB
# 4900.orbax-checkpoint-tmp/                    10 TiB
# -------------------------------------------------------------
# TOTAL BUCKET SIZE                             264 TiB
# =============================================================
