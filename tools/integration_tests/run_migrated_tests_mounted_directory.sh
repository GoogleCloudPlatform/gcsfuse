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

TEST_BUCKET=""
MOUNTED_DIR=""
PACKAGE_NAME="operations" # Default package
RUN_ALL=false

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
CONFIG_FILE="${SCRIPT_DIR}/test_config.yaml"
GCSFUSE_BINARY="${SCRIPT_DIR}/../../gcsfuse"

# --- 1. Argument Parsing ---
usage() {
    echo "Usage: $0 [--bucket <bucket_name>] [--mount-dir <mounted_directory>] [--package <package_name>] [--all]"
    exit 1
}

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --bucket) TEST_BUCKET="$2"; shift 2 ;;
        --mount-dir) MOUNTED_DIR="$2"; shift 2 ;;
        --package) PACKAGE_NAME="$2"; shift 2 ;;
        --all) RUN_ALL=true; shift 1 ;;
        -h|--help) usage ;;
        *) echo "Unknown parameter passed: $1"; usage ;;
    esac
done

# --- 2. Robustness Checks ---  
if ! command -v yq &> /dev/null; then
    echo "Error: 'yq' is not installed. Please install it to parse the YAML config." >&2
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Configuration file not found at $CONFIG_FILE." >&2
    exit 1
fi

# Build gcsfuse binary if it doesn't exist
if [ ! -f "$GCSFUSE_BINARY" ]; then
    echo "⚠️  gcsfuse binary not found at $GCSFUSE_BINARY!"
    echo "🔨 Building it now..."
    (cd "${SCRIPT_DIR}/../.." && go build -o gcsfuse .)
    if [ ! -f "$GCSFUSE_BINARY" ]; then
        echo "Error: Failed to build gcsfuse binary." >&2
        exit 1
    fi
    echo "✅ Successfully built gcsfuse!"
fi

CONFIG_FILE_ABS="$(readlink -f "$CONFIG_FILE")"

# Determine which packages to run
if [ "$RUN_ALL" = true ]; then
    # Extracts all top-level keys from the YAML file
    PACKAGES=$(yq 'keys | .[]' "$CONFIG_FILE")
else
    PACKAGES="$PACKAGE_NAME"
fi

# --- 3. Cleanup Trap ---
# We use a variable so the trap knows exactly which directory is currently mounted
ACTIVE_MOUNT_DIR=""
cleanup() {
    if [ -n "$ACTIVE_MOUNT_DIR" ] && mountpoint -q "$ACTIVE_MOUNT_DIR"; then
        echo "  [Cleanup] Unmounting: fusermount -u $ACTIVE_MOUNT_DIR"
        fusermount -u "$ACTIVE_MOUNT_DIR" || true
    fi
}
trap cleanup EXIT

# --- 4. Execution Loop ---
for CURRENT_PACKAGE in $PACKAGES; do
    echo ""
    echo "#################################################################"
    echo " 🚀 TESTING PACKAGE: $CURRENT_PACKAGE"
    echo "#################################################################"

    # Read base config for the current package
    CONFIG_BASE=$(yq ".${CURRENT_PACKAGE}[0]" "$CONFIG_FILE") 
    if [ -z "$CONFIG_BASE" ] || [ "$CONFIG_BASE" == "null" ]; then
        echo "Warning: Could not find '${CURRENT_PACKAGE}[0]' entry in $CONFIG_FILE. Skipping..." >&2
        continue
    fi

    # Resolve Mounted Directory & Test Bucket (CLI overrides evaluated YAML)
    PKG_MOUNTED_DIR="${MOUNTED_DIR:-$(eval echo "$(echo "$CONFIG_BASE" | yq -r '.mounted_directory')")}"
    PKG_TEST_BUCKET="${TEST_BUCKET:-$(eval echo "$(echo "$CONFIG_BASE" | yq -r '.test_bucket')")}"

    if [ -z "$PKG_MOUNTED_DIR" ] || [ "$PKG_MOUNTED_DIR" == "null" ]; then
        echo "Error: Mounted directory not specified for $CURRENT_PACKAGE. Skipping..." >&2
        continue
    fi
    if [ -z "$PKG_TEST_BUCKET" ] || [ "$PKG_TEST_BUCKET" == "null" ]; then
        echo "Error: Test bucket not specified for $CURRENT_PACKAGE. Skipping..." >&2
        continue
    fi

    NUM_CONFIGS=$(echo "$CONFIG_BASE" | yq '.configs | length')
    if [ "$NUM_CONFIGS" == "null" ] || [ "$NUM_CONFIGS" -eq 0 ]; then
        echo "Error: Found 0 test configurations for $CURRENT_PACKAGE. Skipping..." >&2
        continue
    fi

    GO_TEST_DIR="${SCRIPT_DIR}/${CURRENT_PACKAGE}/..."
    ACTIVE_MOUNT_DIR="$PKG_MOUNTED_DIR"
    mkdir -p "$PKG_MOUNTED_DIR"

    for (( i=0; i<$NUM_CONFIGS; i++ )); do
        TEST_NAME=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].run")
        NUM_FLAG_SETS=$(echo "$CONFIG_BASE" | yq ".configs[$i].flags | length")
        RUN_ON_GKE=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].run_on_gke")
        
        DISPLAY_NAME=${TEST_NAME}
        if [ "$TEST_NAME" == "null" ] || [ -z "$TEST_NAME" ]; then
            DISPLAY_NAME="All tests in ${CURRENT_PACKAGE}"
        fi

        # Skip this configuration if run_on_gke is strictly false
        if [ "$RUN_ON_GKE" == "false" ]; then
            echo -e "\n⏭️  Skipping: ${DISPLAY_NAME} (Package: $CURRENT_PACKAGE) - run_on_gke is false"
            continue
        fi

        for (( j=0; j<$NUM_FLAG_SETS; j++ )); do
            # Extract flags and convert commas to spaces
            RAW_FLAGS=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].flags[$j]")
            FLAGS="${RAW_FLAGS//,/ }" 
            
            echo -e "\n--- Running: ${DISPLAY_NAME} (Package: $CURRENT_PACKAGE) ---"
            echo "--- Flags: ${FLAGS} ---"

            # 1. Mount
            echo "  Mount: $GCSFUSE_BINARY $FLAGS $PKG_TEST_BUCKET $PKG_MOUNTED_DIR"
            if mountpoint -q "$PKG_MOUNTED_DIR"; then fusermount -u "$PKG_MOUNTED_DIR"; fi
            
            # We don't quote $FLAGS so bash splits it into separate arguments
            "$GCSFUSE_BINARY" $FLAGS "$PKG_TEST_BUCKET" "$PKG_MOUNTED_DIR"
            
            # 2. Setup Go Test Command
            GO_CMD=(go test "$GO_TEST_DIR" -p 1 --integrationTest -v --config-file="$CONFIG_FILE_ABS")
            
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
            echo "  Test: GODEBUG=asyncpreemptoff=1 MOUNTED_DIR=\"$PKG_MOUNTED_DIR\" TEST_BUCKET=\"$PKG_TEST_BUCKET\" BUCKET_NAME=\"$PKG_TEST_BUCKET\" ${GO_CMD[*]}"
            env GODEBUG=asyncpreemptoff=1 MOUNTED_DIR="$PKG_MOUNTED_DIR" TEST_BUCKET="$PKG_TEST_BUCKET" BUCKET_NAME="$PKG_TEST_BUCKET" "${GO_CMD[@]}"
            
            # 4. Unmount
            echo "  Unmount: fusermount -u $PKG_MOUNTED_DIR"
            fusermount -u "$PKG_MOUNTED_DIR"
        done
    done
done

ACTIVE_MOUNT_DIR="" # Clear active mount dir since we successfully finished

echo -e "\n========================================================"
echo " ✅ All Specified Tests Completed Successfully!"
echo "========================================================"