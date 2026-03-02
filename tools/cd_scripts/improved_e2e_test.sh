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

# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

# Logging Helpers
log_info() {
    echo "[INFO] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

log_error() {
    echo "[ERROR] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

# Constants
readonly REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT="5.1"
readonly INSTALL_BASH_VERSION="5.3" # Using 5.3 for installation as bash 5.1 has an installation bug.
readonly RELEASE_PACKAGE_BUCKET="gcsfuse-release-packages"

# Defaults
LOCAL_RUN=false
RELEASE_VERSION=""
RUN_TESTS_WITH_ZONAL_BUCKET=false

usage() {
    echo "Usage: $0 [--local-run] "
    echo "  --local-run                     Pass this flag to run this script for local runs. If this flag is passed then gcsfuse is built" 
    echo "                                  locally instead of getting installed by pre-built package from release bucket."
    echo "  --release-version <3.0.0>       Release version determines from which directory the pre-built package is used from release bucket."
    echo "                                  Release version is required if not running using --local-run"
    echo "  --zonal                         Should run tests for zonal bucket (Default: false)"
    exit "$1"
}

# Define options for getopt
# A long option name followed by a colon indicates it requires an argument.
LONG=local-run,zonal,release-version:,help

# Parse the options using getopt
# --options "" specifies that there are no short options.
PARSED=$(getopt --options "" --longoptions "$LONG" --name "$0" -- "$@")
if [[ $? -ne 0 ]]; then
    # getopt will have already printed an error message
    usage 1
fi

# Read the parsed options back into the positional parameters.
eval set -- "$PARSED"

# Loop through the options and assign values to our variables
while (( $# >= 1 )); do
    case "$1" in
        --release-version)
            RELEASE_VERSION="$2"
            shift 2
            # Regex breakdown:
            # ^      : Start of string
            # [0-9]+ : One or more digits
            #  \.     : A literal dot
            # $      : End of string
            RE="^[0-9]+\.[0-9]+\.[0-9]+$"
            if [[ ! $RELEASE_VERSION =~ $RE ]]; then
                log_error "--release-version is incorrectly formatted."
                usage 1
            fi
            ;;
        --zonal)
            RUN_TESTS_WITH_ZONAL_BUCKET=true
            shift
            ;;
        --local-run)
            LOCAL_RUN=true
            shift
            ;;
        --help)
            usage 0
            ;;
        --)
            shift
            break
            ;;
        *)
            log_error "Unrecognized arguments [$*]."
            usage 1
            ;;
    esac
done

# Argument validation
if [[ "$LOCAL_RUN" == "false" ]] && [[ -z "$RELEASE_VERSION" ]]; then
    log_error "--release-version required if not running with --local-run"
    usage 1
fi

# Check and install required bash version for e2e script.
BASH_EXECUTABLE="bash"
REQUIRED_BASH_MAJOR=$(echo "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT" | cut -d'.' -f1)
REQUIRED_BASH_MINOR=$(echo "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT" | cut -d'.' -f2)

log_info "Current Bash version: ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"
log_info "Required Bash version for e2e script: ${REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT}"

if (( BASH_VERSINFO[0] < REQUIRED_BASH_MAJOR || ( BASH_VERSINFO[0] == REQUIRED_BASH_MAJOR && BASH_VERSINFO[1] < REQUIRED_BASH_MINOR ) )); then
    log_info "Current Bash version is older than the required version. Installing Bash ${INSTALL_BASH_VERSION}..."
    ./perfmetrics/scripts/install_bash.sh "$INSTALL_BASH_VERSION"
    BASH_EXECUTABLE="/usr/local/bin/bash"
else
    log_info "Current Bash version (${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}) meets or exceeds the required version (${REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT}). Skipping Bash installation."
fi

# Build args for the e2e script 
ARGS=()

# If Get Region from ZONE
ZONE=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone)
ZONE_NAME=$(basename "$ZONE")
REGION="${ZONE_NAME%-*}"
ARGS+=("--bucket-location=${REGION}")

# Local Run Validation and gcsfuse package installation.
if ${LOCAL_RUN}; then
    log_info "Running script in local mode gcsfuse binary would be built in script from current repository."
else
    log_info "Running script with release version package from release bucket for version $RELEASE_VERSION will be installed."
    # Identify the OS and Architecture
    if [ -f /etc/os-release ]; then
        # We source in a subshell to prevent variable pollution, 
        # then capture only the ID and ID_LIKE fields.
        DISTRO_DATA=$( (source /etc/os-release; echo "${ID:-} ${ID_LIKE:-}") )
        # Check for debian or ubuntu in the ID or the ID_LIKE chain
        if [[ "$DISTRO_DATA" == *"debian"* ]] || [[ "$DISTRO_DATA" == *"ubuntu"* ]]; then
            ARCH=$(dpkg --print-architecture)
            ARGS+=(
                "--install-package-from-path=gs://${RELEASE_PACKAGE_BUCKET}/v${RELEASE_VERSION}/gcsfuse_${RELEASE_VERSION}_${ARCH}.deb"
            )
        elif [[ "$DISTRO_DATA" == *"rhel"* ]] || [[ "$DISTRO_DATA" == *"centos"* ]]; then
            # On RHEL-based systems, we use 'arch' or 'uname -m' 
            # as dpkg is usually not present.
            ARCH=$(uname -m)
            ARGS+=(
                "--install-package-from-path=gs://${RELEASE_PACKAGE_BUCKET}/v${RELEASE_VERSION}/gcsfuse-${RELEASE_VERSION}-1.${ARCH}.rpm"
            )
        else
            log_error "This script only supports Debian/Ubuntu/rhel/centos based distributions."
            log_info "Your distribution is:"
            cat /etc/os-release
            exit 1
        fi
    else
        log_error "/etc/os-release not found. Unable to determine distribution"
        exit 1
    fi
fi

# Set parallelism to 3
ARGS+=( "--package-level-parallelism=3")

# Set --zonal arg if required
if ${RUN_TESTS_WITH_ZONAL_BUCKET}; then
    log_info "Running zonal e2e tests."
    ARGS+=("--zonal")
else
    log_info "Running regional e2e tests."
fi

# Run the main e2e script
"${BASH_EXECUTABLE}" ./tools/integration_tests/improved_run_e2e_tests.sh "${ARGS[@]}"
