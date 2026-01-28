#! /bin/bash
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

# Print commands and their arguments as they are executed.
set -x
# -e: Exit on error, -u: Exit on unset vars, -o pipefail: Pipeline error trapping
set -euo pipefail

# Defaults
LOCAL_RUN=false
CUSTOM_BUCKET=""
LOG_FILE=$(pwd)/logs.txt
TEST_USER="starterscriptuser"
HOME_DIR="/home/${TEST_USER}"
ARTIFACTS_DIR="${HOME_DIR}/failure_logs"
DETAILS_FILE="$(pwd)/details.txt"
# Determine the absolute location of THIS script to find repo root
SCRIPT_DIR=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")
REPO_ROOT=$(realpath "${SCRIPT_DIR}/../..")
LOCAL_GO_VERSION=$(cat "${REPO_ROOT}/.go-version")

usage() {
    echo "Usage: $0 [--local-run] [--bucket <bucket_name>]"
    echo "  --local-run    Skips user creation and GCS log uploads."
    echo "  --bucket       Sets a custom bucket name (overrides default/metadata)."
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --local-run)
            LOCAL_RUN=true
            TEST_USER="$USER"
            HOME_DIR="$HOME"
            ARTIFACTS_DIR="$(pwd)/failure_logs"
            DETAILS_FILE="$(pwd)/details.txt"
            echo "Running on LOCAL machine. No new users created, logs won't be uploaded."
            shift
            ;;
        --bucket)
            if [[ -n "${2:-}" ]]; then
                CUSTOM_BUCKET="$2"
                shift
                shift
            else
                echo "ERROR: --bucket requires a non-empty value."
                exit 1
            fi
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "ERROR: Unknown argument: $1"
            usage
            ;;
    esac
done

# Initialize Log File
touch "$LOG_FILE"
chmod 666 "$LOG_FILE"
echo "User: $USER" &>> "${LOG_FILE}"
echo "Current Working Directory: $(pwd)" &>> "${LOG_FILE}"
echo "Repository Root: ${REPO_ROOT}" &>> "${LOG_FILE}"

# ==============================================================================
# 1. HELPER FUNCTIONS
# ==============================================================================

cleanup() {
    echo "Cleaning up temporary script files..."
    rm -f /tmp/test_exit_code.txt /tmp/test_log_filename.txt /tmp/run_tests_logic.sh
    echo "Cleanup complete."
}
trap cleanup EXIT

install_system_deps() {
    echo "Installing system dependencies..."
    
    # Source os_utils.sh
    if [ -f "${REPO_ROOT}/perfmetrics/scripts/os_utils.sh" ]; then
        source "${REPO_ROOT}/perfmetrics/scripts/os_utils.sh"
    else
        echo "Error: os_utils.sh not found."
        exit 1
    fi
    
    local os_id
    os_id=$(get_os_id)
    
    local pkgs=("wget" "fuse" "git")
    if [[ "$os_id" == "ubuntu" || "$os_id" == "debian" ]]; then
        pkgs+=("build-essential")
    else
        pkgs+=("gcc" "gcc-c++" "make")
    fi
    
    install_packages_by_os "$os_id" "${pkgs[@]}"

    # Upgrade gcloud
    echo "Upgrading gcloud..."
    bash "${REPO_ROOT}/perfmetrics/scripts/install_latest_gcloud.sh"
    export PATH="/usr/local/google-cloud-sdk/bin:$PATH"
    export CLOUDSDK_PYTHON="$HOME/.local/python-3.11.9/bin/python3.11"
    export PATH="$HOME/.local/python-3.11.9/bin:$PATH"
}

fetch_metadata() {
    local os_id
    os_id=$(get_os_id)
    if [ "$LOCAL_RUN" = true ]; then
        echo "Local run detected. Setting dummy metadata."
        ZONE="projects/12345/zones/us-west1-b"
        ZONE_NAME="us-west1-b"
        
        RUN_ON_ZB_ONLY="false"
        
        # Recreate details.txt in local mode to include OS type in the instance name field
        # Line 1: VERSION, Line 2: COMMIT_HASH, Line 3: VM_INSTANCE_NAME
        echo "3.5.4" > "$DETAILS_FILE"
        echo "local-commit" >> "$DETAILS_FILE"
        echo "local-instance-${os_id}" >> "$DETAILS_FILE"
    else
        # Real Metadata Fetching
        ZONE=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone)
        ZONE_NAME=$(basename "$ZONE")
        CUSTOM_BUCKET=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.custom_bucket)')
        RUN_ON_ZB_ONLY=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.run-on-zb-only)')
        
        # Fetch details.txt from bucket
        gcloud storage cp "gs://${BUCKET_NAME_TO_USE}/version-detail/details.txt" .
        # Append instance name from metadata server
        curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >> "$DETAILS_FILE"
    fi
    BUCKET_NAME_TO_USE=${CUSTOM_BUCKET:-"gcsfuse-local-run-cd-script"}
    echo "ZONE_NAME: $ZONE_NAME"
    echo "BUCKET_NAME_TO_USE: $BUCKET_NAME_TO_USE"
    
    # Extract version info
    VERSION=$(sed -n 1p "$DETAILS_FILE")
    COMMIT_HASH=$(sed -n 2p "$DETAILS_FILE")
    VM_INSTANCE_NAME=$(sed -n 3p "$DETAILS_FILE")
}

setup_test_user() {
    if [ "$LOCAL_RUN" = true ]; then
        echo "Skipping user creation for local run."
        return 0
    fi

    local user=$1
    local home=$2
    local details=$3

    if id "${user}" &>/dev/null; then
        echo "User ${user} already exists."
    else
        echo "Creating user ${user}..."
        if grep -qi -E 'ubuntu|debian' "$details"; then
            sudo adduser --disabled-password --home "${home}" --gecos "" "${user}"
        else
            sudo adduser --home-dir "${home}" "${user}" && sudo passwd -d "${user}"
        fi
    fi

    # Grant Sudo
    sudo mkdir -p /etc/sudoers.d/
    local sudoers_file="/etc/sudoers.d/${user}"
    if ! sudo test -f "${sudoers_file}"; then
        echo "${user} ALL=(ALL:ALL) NOPASSWD:ALL" | sudo tee "${sudoers_file}" > /dev/null
        sudo chmod 440 "${sudoers_file}"
    fi
}

install_go() {
    # Read the Go version from the centralized .go-version file
    local go_version=$(cat "${REPO_ROOT}/.go-version")
    echo "Installing Go version: ${go_version}..."
    
    # Execute the installation script with the retrieved version
    bash "${REPO_ROOT}/perfmetrics/scripts/install_go.sh" "${go_version}"
}


install_gcsfuse_package() {
    if grep -q ubuntu "$DETAILS_FILE" || grep -q debian "$DETAILS_FILE"; then
        architecture=$(dpkg --print-architecture)
    else
        uname=$(uname -m)
        if [[ $uname == "x86_64" ]]; then architecture="amd64"; elif [[ $uname == "aarch64" ]]; then architecture="arm64"; fi
    fi

    install_go

    # CI/CD Logic: Download and install package if using default bucket
    if [[ "${BUCKET_NAME_TO_USE}" == "gcsfuse-release-packages" ]]; then
        echo "Downloading pre-built package..." &>> "${LOG_FILE}"
        if grep -q -E 'ubuntu|debian' "$DETAILS_FILE"; then
            gcloud storage cp "gs://${BUCKET_NAME_TO_USE}/v${VERSION}/gcsfuse_${VERSION}_${architecture}.deb" . &>> "${LOG_FILE}"
            sudo dpkg -i "gcsfuse_${VERSION}_${architecture}.deb" |& tee -a "${LOG_FILE}"
        else
            gcloud storage cp "gs://${BUCKET_NAME_TO_USE}/v${VERSION}/gcsfuse-${VERSION}-1.${uname}.rpm" . &>> "${LOG_FILE}"
            sudo yum -y localinstall "gcsfuse-${VERSION}-1.${uname}.rpm" &>> "${LOG_FILE}"
        fi
    else
        echo "Custom bucket detected; skipping pre-built package installation." &>> "${LOG_FILE}"
    fi
}

run_tests() {
    # Prepare the test script content dynamically
    local script_runner_path="/tmp/run_tests_logic.sh"
    
    cat << EOF > "$script_runner_path"
#!/bin/bash
set -e
set -x

export PATH=/usr/local/google-cloud-sdk/bin:/usr/local/go/bin:\$PATH
export HOME="${HOME_DIR}"
export KOKORO_ARTIFACTS_DIR="${ARTIFACTS_DIR}"
export ZONE_NAME="${ZONE_NAME}"
export RUN_ON_ZB_ONLY="${RUN_ON_ZB_ONLY}"

mkdir -p "\$KOKORO_ARTIFACTS_DIR"

# Repository Setup
if [ "${LOCAL_RUN}" = "true" ]; then
    # Fix for sudo changing HOME to /root:
    # Explicitly change to the directory where the user called the script
    echo "Local Run: Using existing repository at ${REPO_ROOT}"
    cd "${REPO_ROOT}"
else
    # CI Mode: Clone the repo
    cd "\$HOME"
    git clone https://github.com/googlecloudplatform/gcsfuse
    cd gcsfuse
    git checkout "${COMMIT_HASH}"
fi

# Always perform cleanup before building or executing tests
echo "Cleaning up any previous GCSFuse mounts, processes, and artifacts..."
# Kill any remaining gcsfuse processes to prevent mount conflicts
sudo pkill -f gcsfuse || true
# CRITICAL: Clean up bin/sbin to prevent "mkdir: file exists" errors in the build tool
rm -rf bin sbin

# Build GCSFuse from source if it's a local run or requires a custom build
if [ "${LOCAL_RUN}" = "true" ] || [[ "${BUCKET_NAME_TO_USE}" != "gcsfuse-release-packages" ]]; then
    echo "Building GCSFuse from source (Version: ${VERSION})..."
    go run tools/build_gcsfuse/main.go . . "${VERSION}"
    sudo cp bin/* /usr/bin/
    sudo cp sbin/* /usr/sbin/
fi


# Determine Region
REGION=\${ZONE_NAME%-*}

# Execute Test Wrapper
TEST_SCRIPT="./tools/integration_tests/improved_run_e2e_tests.sh"
chmod +x \$TEST_SCRIPT

ARGS="--bucket-location \$REGION --test-installed-package --no-build-binary-in-script --package-level-parallelism=5"

if [[ "\$RUN_ON_ZB_ONLY" == "true" ]]; then
    ARGS="\$ARGS --zonal"
fi

echo "----------------------------------------------------------------"
echo "EXECUTING TEST SCRIPT: \$TEST_SCRIPT \$ARGS"
echo "----------------------------------------------------------------"

# Capture exit code
set +e
TIMESTAMP=\$(date +%d-%m-%H-%M)
LOG_FILENAME="e2e_run_logs_\${TIMESTAMP}.txt"

# Run tests and capture output
\$TEST_SCRIPT \$ARGS 2>&1 | tee -a "${LOG_FILE}"
EXIT_CODE=\${PIPESTATUS[0]}
set -e

# Export for the parent script to see (via file, since we are in subshell/sudo)
echo \$EXIT_CODE > /tmp/test_exit_code.txt
echo \$LOG_FILENAME > /tmp/test_log_filename.txt
EOF

    chmod +x "$script_runner_path"

    # Execute the runner
    if [ "$LOCAL_RUN" = true ]; then
        # Run directly as current user (preserving env)
        /bin/bash "$script_runner_path"
    else
        # Ensure user owns details file before running
        cp "$DETAILS_FILE" "${HOME_DIR}/"
        chown "${TEST_USER}:${TEST_USER}" "${HOME_DIR}/details.txt"
        
        # Run as test user
        sudo -u "$TEST_USER" /bin/bash "$script_runner_path"
    fi
}

upload_logs() {
    if [ "$LOCAL_RUN" = true ]; then
        EXIT_CODE=$(cat /tmp/test_exit_code.txt)
        echo "Local run: Skipping GCS upload."
        if [ "$EXIT_CODE" -ne 0 ]; then
            echo "Tests failed. Failure logs are located in: ${ARTIFACTS_DIR}"
        else
            echo "Tests passed. Cleaning up failure logs directory."
            rm -rf "${ARTIFACTS_DIR}"
        fi
        exit "$EXIT_CODE"
    else
        # Retrieve values from the test run
        EXIT_CODE=$(cat /tmp/test_exit_code.txt)
        LOG_FILENAME=$(cat /tmp/test_log_filename.txt)
        GCS_DEST="gs://${BUCKET_NAME_TO_USE}/v${VERSION}/${COMMIT_HASH}/${VM_INSTANCE_NAME}/"
        TIMESTAMP=$(date +%d-%m-%H-%M)

        echo "Uploading logs to $GCS_DEST..."
        gcloud storage cp "$LOG_FILE" "${GCS_DEST}_combined_e2e_logs_${TIMESTAMP}.txt"
        echo "Logfile for this run: ${GCS_DEST}_combined_e2e_logs_${TIMESTAMP}.txt"
        if [ "$EXIT_CODE" -eq 0 ]; then
            if [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
                touch ~/success-zonal.txt
                gcloud storage cp ~/success-zonal.txt "${GCS_DEST}"
            else
                touch ~/success.txt
                gcloud storage cp ~/success.txt "${GCS_DEST}"
            fi
        else
            echo "Tests failed. Check logs."
        fi
        exit "$EXIT_CODE"
    fi
}

# ==============================================================================
# 2. MAIN EXECUTION FLOW
# ==============================================================================

install_system_deps
fetch_metadata
setup_test_user "$TEST_USER" "$HOME_DIR" "$DETAILS_FILE"
install_gcsfuse_package
run_tests
upload_logs