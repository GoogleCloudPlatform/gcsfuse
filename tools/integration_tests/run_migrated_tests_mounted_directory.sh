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

# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

RUN_ZONAL=false
PROJECT_ID=""
BUCKET_LOCATION=""
MOUNTED_DIR="/home/cpranjal_google_com/mnt"
PACKAGE_NAME="" # Default is empty to trigger run-all
RUN_ALL=false

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
CONFIG_FILE="${SCRIPT_DIR}/test_config.yaml"
GCSFUSE_BINARY="${SCRIPT_DIR}/../../gcsfuse"

# --- 0. Modular Utility Functions ---

log_info() {
    echo "[INFO] $(date +"%Y-%m-%d %H:%M:%S"): $1" >&2
}

log_error() {
    echo "[ERROR] $(date +"%Y-%m-%d %H:%M:%S"): $1" >&2
}

fetch_gce_metadata() {
    if [ -z "${PROJECT_ID:-}" ]; then
        PROJECT_ID=$(curl -s --connect-timeout 2 -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/project/project-id || echo "")
    fi
    if [ -z "${BUCKET_LOCATION:-}" ]; then
        local zone
        zone=$(curl -s --connect-timeout 2 -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone || echo "")
        if [ -n "$zone" ]; then
            local zone_name=$(basename "$zone")
            BUCKET_LOCATION="${zone_name%-*}"
        fi
    fi
    
    # Cloudtop specific fallback
    if [[ "$PROJECT_ID" == *"cloudtop"* ]]; then
        PROJECT_ID="gcs-fuse-test"
    fi

    # Final fallbacks
    PROJECT_ID="${PROJECT_ID:-gcs-fuse-test}"
    BUCKET_LOCATION="${BUCKET_LOCATION:-us-central1}"
}

create_bucket() {
    local bucket_type="$1"
    local package_name="$2"
    local project_id="$3"
    local location="$4"
    
    local safe_pkg="${package_name//_/-}"
    # Shorten the components to ensure length strictly <= 63 characters!
    local bucket_name="gcsfuse-mounted-tests-${safe_pkg:0:20}-${bucket_type}-$(date +%s)"
    local cmd=("gcloud" "alpha" "storage" "buckets" "create" "gs://${bucket_name}" "--project=${project_id}" "--location=${location}" "--uniform-bucket-level-access")

    if [[ "$bucket_type" == "hns" ]]; then
        cmd+=("--enable-hierarchical-namespace")
    elif [[ "$bucket_type" == "zonal" ]]; then
        cmd+=("--enable-hierarchical-namespace" "--placement=${location}-a" "--default-storage-class=RAPID")
    elif [[ "$bucket_type" != "flat" ]]; then
        log_error "Invalid bucket type: $bucket_type"
        return 1
    fi

    log_info "Creating $bucket_type bucket: $bucket_name..."
    if ! "${cmd[@]}" > /dev/null; then
        log_error "Failed to create bucket $bucket_name"
        return 1
    fi
    
    # ONLY echo the raw bucket name to stdout so $(create_bucket) stays unpolluted
    echo "$bucket_name"
}

delete_bucket() {
    local bucket_name="$1"
    log_info "Deleting bucket: $bucket_name..."
    gcloud storage rm -r "gs://${bucket_name}" --no-user-output-enabled --verbosity=error || true
}

# --- 1. Argument Parsing ---
usage() {
    echo "Usage: $0 [--zonal] [--project-id <project_id>] [--location <location>] [--mount-dir <mounted_directory>] [--package <package_name>] [--all]" >&2
    exit 1
}

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --zonal) RUN_ZONAL=true; shift 1 ;;
        --project-id) PROJECT_ID="$2"; shift 2 ;;
        --location) BUCKET_LOCATION="$2"; shift 2 ;;
        --mount-dir) MOUNTED_DIR="$2"; shift 2 ;;
        --package) PACKAGE_NAME="$2"; shift 2 ;;
        --all) RUN_ALL=true; shift 1 ;;
        -h|--help) usage ;;
        *) echo "Unknown parameter passed: $1" >&2; usage ;;
    esac
done

# If package name is missing or RUN_ALL is passed, run everything!
if [ -z "$PACKAGE_NAME" ] || [ "$RUN_ALL" = true ]; then
    RUN_ALL=true
fi

# --- 2. Robustness Checks ---  
if ! command -v yq &> /dev/null; then
    log_error "'yq' is not installed. Please install it to parse the YAML config."
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    log_error "Configuration file not found at $CONFIG_FILE."
    exit 1
fi

# Build gcsfuse binary if it doesn't exist
if [ ! -f "$GCSFUSE_BINARY" ]; then
    log_info "gcsfuse binary not found at $GCSFUSE_BINARY. Building it now..."
    (cd "${SCRIPT_DIR}/../.." && go build -o gcsfuse .)
    if [ ! -f "$GCSFUSE_BINARY" ]; then
        log_error "Failed to build gcsfuse binary."
        exit 1
    fi
    log_info "Successfully built gcsfuse."
fi

CONFIG_FILE_ABS="$(readlink -f "$CONFIG_FILE")"

# Determine which packages to run
if [ "$RUN_ALL" = true ]; then
    PACKAGES=$(yq 'keys | .[]' "$CONFIG_FILE")
else
    PACKAGES="$PACKAGE_NAME"
fi

# Set bucket types based on flags
if [ "$RUN_ZONAL" = true ]; then
    BUCKET_TYPES=("zonal")
else
    BUCKET_TYPES=("hns" "flat")
fi

# Fetch metadata for bucket creation
fetch_gce_metadata
log_info "Using Project ID: $PROJECT_ID"
log_info "Using Bucket Location: $BUCKET_LOCATION"

# --- 3. Cleanup Trap ---
ACTIVE_MOUNT_DIR=""
ACTIVE_SEC_MOUNT_DIR=""
CREATED_BUCKETS=()
PACKAGE_RUNTIME_STATS=$(mktemp)
OVERALL_EXIT_CODE=0

cleanup() {
    set +e
    if [ -n "$ACTIVE_MOUNT_DIR" ] && mountpoint -q "$ACTIVE_MOUNT_DIR"; then
        log_info "Unmounting: fusermount -u -z $ACTIVE_MOUNT_DIR"
        fusermount -u -z "$ACTIVE_MOUNT_DIR" || umount -l "$ACTIVE_MOUNT_DIR" || true
    fi
    if [ -n "$ACTIVE_SEC_MOUNT_DIR" ] && mountpoint -q "$ACTIVE_SEC_MOUNT_DIR"; then
        log_info "Unmounting secondary: fusermount -u -z $ACTIVE_SEC_MOUNT_DIR"
        fusermount -u -z "$ACTIVE_SEC_MOUNT_DIR" || umount -l "$ACTIVE_SEC_MOUNT_DIR" || true
    fi
    for b in "${CREATED_BUCKETS[@]}"; do
        if [ -n "$b" ]; then
            delete_bucket "$b"
        fi
    done
    rm -f "$PACKAGE_RUNTIME_STATS"
}
trap cleanup EXIT

# Reset SECONDS timer
SECONDS=0

# --- 4. Execution Loop ---
for CURRENT_PACKAGE in $PACKAGES; do
    # Read base config for the current package
    CONFIG_BASE=$(yq ".${CURRENT_PACKAGE}[0]" "$CONFIG_FILE") 
    if [ -z "$CONFIG_BASE" ] || [ "$CONFIG_BASE" == "null" ]; then
        log_error "Could not find '${CURRENT_PACKAGE}[0]' entry in $CONFIG_FILE. Skipping..."
        continue
    fi

    # Resolve Mounted Directory
    PKG_MOUNTED_DIR="${MOUNTED_DIR:-$(echo "$CONFIG_BASE" | yq -r '.mounted_directory' | envsubst)}"
    if [ -z "$PKG_MOUNTED_DIR" ] || [ "$PKG_MOUNTED_DIR" == "null" ]; then
        log_error "Mounted directory not specified for $CURRENT_PACKAGE. Skipping..."
        continue
    fi

    NUM_CONFIGS=$(echo "$CONFIG_BASE" | yq '.configs | length')
    if [ "$NUM_CONFIGS" == "null" ] || [ "$NUM_CONFIGS" -eq 0 ]; then
        log_error "Found 0 test configurations for $CURRENT_PACKAGE. Skipping..."
        continue
    fi

    GO_TEST_DIR="${SCRIPT_DIR}/${CURRENT_PACKAGE}/..."
    ACTIVE_MOUNT_DIR="$PKG_MOUNTED_DIR"
    mkdir -p "$PKG_MOUNTED_DIR"

    for CURRENT_BUCKET_TYPE in "${BUCKET_TYPES[@]}"; do
        # Create temporary bucket for this package and bucket type
        PKG_TEST_BUCKET=""
        BUCKET_CREATED_FLAG=false

        for (( i=0; i<$NUM_CONFIGS; i++ )); do
            # 1. Check compatibility first!
            IS_COMPATIBLE=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].compatible.${CURRENT_BUCKET_TYPE}")
            if [ "$IS_COMPATIBLE" == "false" ]; then
                continue # Skip this entire configuration set
            fi

            # Ensure bucket is created lazily (only if there are compatible tests)
            if [ "$BUCKET_CREATED_FLAG" = false ]; then
                echo ""
                echo "#################################################################"
                echo " TESTING PACKAGE: $CURRENT_PACKAGE | BUCKET TYPE: $CURRENT_BUCKET_TYPE"
                echo "#################################################################"
                PKG_TEST_BUCKET=$(create_bucket "$CURRENT_BUCKET_TYPE" "$CURRENT_PACKAGE" "$PROJECT_ID" "$BUCKET_LOCATION")
                CREATED_BUCKETS+=("$PKG_TEST_BUCKET")
                BUCKET_CREATED_FLAG=true
            fi

            TEST_NAME=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].run")
            NUM_FLAG_SETS=$(echo "$CONFIG_BASE" | yq ".configs[$i].flags | length") 
            RUN_ON_GKE=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].run_on_gke")
            
            # 2. Dual-mount verification
            NUM_SEC_FLAGS=$(echo "$CONFIG_BASE" | yq ".configs[$i].secondary_flags | length")
            if [ "$NUM_SEC_FLAGS" == "null" ]; then NUM_SEC_FLAGS=0; fi

            DISPLAY_NAME=${TEST_NAME}
            if [ "$TEST_NAME" == "null" ] || [ -z "$TEST_NAME" ]; then
                DISPLAY_NAME="All tests in ${CURRENT_PACKAGE}"
            fi

            if [ "$RUN_ON_GKE" == "false" ]; then
                echo -e "\nSkipping: ${DISPLAY_NAME} (Package: $CURRENT_PACKAGE) - run_on_gke is false"
                continue
            fi

            for (( j=0; j<$NUM_FLAG_SETS; j++ )); do
                RAW_FLAGS=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].flags[$j]")
                FLAGS="${RAW_FLAGS//,/ }" 
                
                # Secondary flags
                if [ "$NUM_SEC_FLAGS" -gt "$j" ]; then
                    RAW_SEC_FLAGS=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].secondary_flags[$j]")
                    SEC_FLAGS="${RAW_SEC_FLAGS//,/ }"
                    IS_DUAL_MOUNT=true
                else
                    SEC_FLAGS=""
                    IS_DUAL_MOUNT=false
                fi

                # Enforce trace logging
                FLAGS=$(echo "$FLAGS" | sed -E 's/--log-severity(=| )[a-zA-Z0-9]+//gi')
                FLAGS="$FLAGS --log-severity=trace"
                
                # Check if only_dir is requested
                ONLY_DIR_KEY=$(echo "$CONFIG_BASE" | yq -r '.only_dir')
                if [ "$ONLY_DIR_KEY" != "null" ] && [ -n "$ONLY_DIR_KEY" ]; then
                    export ONLY_DIR="only-dir-mnt"
                    if [[ "$FLAGS" != *"--only-dir"* ]]; then
                        FLAGS="$FLAGS --only-dir=${ONLY_DIR}"
                    fi
                else
                    export ONLY_DIR=""
                fi

                # Add zonal flag if applicable
                if [ "$CURRENT_BUCKET_TYPE" == "zonal" ]; then
                    GO_ZONAL_FLAG="--zonal"
                else
                    GO_ZONAL_FLAG=""
                fi

                echo -e "\n--- Running: ${DISPLAY_NAME} (Package: $CURRENT_PACKAGE) ---"
                echo "--- Flags: ${FLAGS} ---"

                # 1. Mount Primary
                echo "  Mount Primary: $GCSFUSE_BINARY $FLAGS $PKG_TEST_BUCKET $PKG_MOUNTED_DIR"
                if mountpoint -q "$PKG_MOUNTED_DIR"; then fusermount -u "$PKG_MOUNTED_DIR"; fi
                "$GCSFUSE_BINARY" $FLAGS "$PKG_TEST_BUCKET" "$PKG_MOUNTED_DIR" > /dev/null 2>&1
                
                # 1.5. Mount Secondary
                PKG_MOUNTED_DIR_SECONDARY="${PKG_MOUNTED_DIR}_sec"
                if [ "$IS_DUAL_MOUNT" = true ]; then
                    ACTIVE_SEC_MOUNT_DIR="$PKG_MOUNTED_DIR_SECONDARY"
                    mkdir -p "$PKG_MOUNTED_DIR_SECONDARY"
                    if mountpoint -q "$PKG_MOUNTED_DIR_SECONDARY"; then fusermount -u "$PKG_MOUNTED_DIR_SECONDARY"; fi
                    echo "  Mount Secondary: $GCSFUSE_BINARY $SEC_FLAGS $PKG_TEST_BUCKET $PKG_MOUNTED_DIR_SECONDARY"
                    "$GCSFUSE_BINARY" $SEC_FLAGS "$PKG_TEST_BUCKET" "$PKG_MOUNTED_DIR_SECONDARY" > /dev/null 2>&1
                fi

                # 2. Setup Go Test Command
                GO_CMD=(go test "$GO_TEST_DIR" -p 1 --integrationTest -v --config-file="$CONFIG_FILE_ABS")
                if [ -n "$GO_ZONAL_FLAG" ]; then GO_CMD+=("$GO_ZONAL_FLAG"); fi

                if [ "$TEST_NAME" != "null" ] && [ -n "$TEST_NAME" ]; then
                    if [[ "$TEST_NAME" == Benchmark_* ]]; then
                        GO_CMD+=(-bench "^${TEST_NAME}$" -run "^$")
                    else
                        GO_CMD+=(-run "^${TEST_NAME}$")
                    fi
                else
                    GO_CMD+=(-bench .) 
                fi
                
                # 3. Run Test
                echo "  Test: GODEBUG=asyncpreemptoff=1 MOUNTED_DIR=\"$PKG_MOUNTED_DIR\" ${IS_DUAL_MOUNT:+MOUNTED_DIR_SECONDARY=\"$PKG_MOUNTED_DIR_SECONDARY\"} ${ONLY_DIR:+ONLY_DIR=\"$ONLY_DIR\"} TEST_BUCKET=\"$PKG_TEST_BUCKET\" BUCKET_NAME=\"$PKG_TEST_BUCKET\" ${GO_CMD[*]}"
                
                start=$SECONDS
                exit_code=0
                if ! env GODEBUG=asyncpreemptoff=1 MOUNTED_DIR="$PKG_MOUNTED_DIR" MOUNTED_DIR_SECONDARY="${IS_DUAL_MOUNT:+$PKG_MOUNTED_DIR_SECONDARY}" ONLY_DIR="$ONLY_DIR" TEST_BUCKET="$PKG_TEST_BUCKET" BUCKET_NAME="$PKG_TEST_BUCKET" "${GO_CMD[@]}"; then
                    exit_code=1
                    OVERALL_EXIT_CODE=1
                    log_error "Tests failed in ${CURRENT_PACKAGE} on bucket type ${CURRENT_BUCKET_TYPE}!"
                fi
                end=$SECONDS
                
                # Append stats for visualizer
                echo "${CURRENT_PACKAGE} ${CURRENT_BUCKET_TYPE} ${exit_code} ${start} ${end}" >> "$PACKAGE_RUNTIME_STATS"
                
                # 4. Unmount Primary
                echo "  Unmount: fusermount -u $PKG_MOUNTED_DIR"
                fusermount -u "$PKG_MOUNTED_DIR"

                # 4.5. Unmount Secondary
                if [ "$IS_DUAL_MOUNT" = true ]; then
                    echo "  Unmount Secondary: fusermount -u $PKG_MOUNTED_DIR_SECONDARY"
                    fusermount -u "$PKG_MOUNTED_DIR_SECONDARY" || true
                    ACTIVE_SEC_MOUNT_DIR=""
                fi
            done
        done
    done
done

ACTIVE_MOUNT_DIR=""

# Print beautiful rich-based summary table
if [ -f "$PACKAGE_RUNTIME_STATS" ] && [ -s "$PACKAGE_RUNTIME_STATS" ]; then
    echo -e "\n------ Test Summary Table ------"
    "${SCRIPT_DIR}/create_package_runtime_table.sh" "$PACKAGE_RUNTIME_STATS" || true
fi

if [ "$OVERALL_EXIT_CODE" -eq 0 ]; then
    echo -e "\n========================================================"
    echo " All Specified Tests Completed Successfully!"
    echo "========================================================"
else
    echo -e "\n========================================================"
    echo " Some tests failed. Check summary above."
    echo "========================================================"
fi

exit "$OVERALL_EXIT_CODE"
