#! /bin/bash
# Copyright 2025 Google LLC
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
# Exit immediately if a command exits with a non-zero status.
set -e

# ==============================================================================
# 1. SYSTEM PREPARATION (Root Level)
# ==============================================================================

# Install wget
if command -v apt-get &> /dev/null; then
    sudo apt-get update && sudo apt-get install -y wget
elif command -v yum &> /dev/null; then
    sudo yum install -y wget
else
    exit 1
fi

# Upgrade gcloud
echo "Upgrade gcloud version"
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk

# Conditionally install python3.11 for RHEL 8/Rocky 8
INSTALL_COMMAND="sudo /usr/local/google-cloud-sdk/install.sh --quiet"
if [ -f /etc/os-release ]; then
    # shellcheck source=/dev/null
    . /etc/os-release
    if [[ ($ID == "rhel" || $ID == "rocky") && $VERSION_ID == 8* ]]; then
        sudo yum install -y python311
        INSTALL_COMMAND="sudo env CLOUDSDK_PYTHON=/usr/bin/python3.11 /usr/local/google-cloud-sdk/install.sh --quiet"
    fi
fi
$INSTALL_COMMAND 

export PATH=/usr/local/google-cloud-sdk/bin:$PATH
gcloud version && rm gcloud.tar.gz

# Retrieve Metadata
ZONE=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone)
ZONE_NAME=$(basename "$ZONE")
RUN_ON_ZB_ONLY=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.run-on-zb-only)')
RUN_READ_CACHE_TESTS_ONLY=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.run-read-cache-only)')

echo "Got ZONE=\"${ZONE}\" (Name: ${ZONE_NAME})"
echo "RUN_ON_ZB_ONLY flag set to : \"${RUN_ON_ZB_ONLY}\""
echo "RUN_READ_CACHE_TESTS_ONLY flag set to : \"${RUN_READ_CACHE_TESTS_ONLY}\""

# Download details.txt (Release version info)
gcloud storage cp gs://gcsfuse-release-packages/version-detail/details.txt .
# Append instance name to details.txt
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >>details.txt

# Create User Function
create_user() {
  local USERNAME=$1
  local HOMEDIR=$2
  local DETAILS=$3
  if id "${USERNAME}" &>/dev/null; then
    echo "User ${USERNAME} already exists."
    return 0
  fi

  echo "Creating user ${USERNAME}..."
  if grep -qi -E 'ubuntu|debian' "$DETAILS"; then
    sudo adduser --disabled-password --home "${HOMEDIR}" --gecos "" "${USERNAME}"
  elif grep -qi -E 'rhel|centos|rocky' "$DETAILS"; then
    sudo adduser --home-dir "${HOMEDIR}" "${USERNAME}" && sudo passwd -d "${USERNAME}"
  else
    echo "Unsupported OS type in details file." >&2
    return 1
  fi
}

# Grant Sudo Function
grant_sudo() {
  local USERNAME=$1
  if ! id "${USERNAME}" &>/dev/null; then return 1; fi
  sudo mkdir -p /etc/sudoers.d/
  SUDOERS_FILE="/etc/sudoers.d/${USERNAME}"
  if ! sudo test -f "${SUDOERS_FILE}"; then
    echo "${USERNAME} ALL=(ALL:ALL) NOPASSWD:ALL" | sudo tee "${SUDOERS_FILE}" > /dev/null
    sudo chmod 440 "${SUDOERS_FILE}"
  fi
}

USERNAME=starterscriptuser
HOMEDIR="/home/${USERNAME}"
DETAILS_FILE=$(pwd)/details.txt

create_user "$USERNAME" "$HOMEDIR" "$DETAILS_FILE"
grant_sudo  "$USERNAME"

# ==============================================================================
# 2. PACKAGE INSTALLATION (Root Level)
# We must install the package being tested (the .deb or .rpm) before switching users.
# ==============================================================================

# Logs for installation
touch logs.txt
chmod 666 logs.txt
LOG_FILE=$(pwd)/logs.txt

if grep -q ubuntu details.txt || grep -q debian details.txt; then
    architecture=$(dpkg --print-architecture)
    sudo apt update
    sudo apt install -y fuse wget git build-essential
    
    # Download and install specific GCSFuse version
    # Quoted to prevent word splitting on version numbers or paths
    gcloud storage cp "gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse_$(sed -n 1p details.txt)_${architecture}.deb" .
    sudo dpkg -i "gcsfuse_$(sed -n 1p details.txt)_${architecture}.deb" |& tee -a "${LOG_FILE}"
else
    # RHEL/CentOS
    uname=$(uname -m)
    if [[ $uname == "x86_64" ]]; then architecture="amd64"; elif [[ $uname == "aarch64" ]]; then architecture="arm64"; fi

    sudo yum makecache
    sudo yum -y update
    sudo yum -y install fuse wget git gcc gcc-c++ make
    
    # Download and install specific GCSFuse version
    # Quoted to prevent word splitting on version numbers or paths
    gcloud storage cp "gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse-$(sed -n 1p details.txt)-1.${uname}.rpm" .
    sudo yum -y localinstall "gcsfuse-$(sed -n 1p details.txt)-1.${uname}.rpm" |& tee -a "${LOG_FILE}"
fi

# Install Go (Required for the test runner script)
wget -O go_tar.tar.gz "https://go.dev/dl/go1.24.10.linux-${architecture}.tar.gz"
sudo tar -C /usr/local -xzf go_tar.tar.gz
export PATH=${PATH}:/usr/local/go/bin

# Log versions
gcsfuse --version |& tee -a "${LOG_FILE}"
go version |& tee -a "${LOG_FILE}"

# ==============================================================================
# 3. TEST EXECUTION (Delegation to starterscriptuser)
# ==============================================================================

# Ensure starterscriptuser can read details.txt
cp details.txt "$HOMEDIR/"
chown "$USERNAME:$USERNAME" "$HOMEDIR/details.txt"

# Run the following as starterscriptuser
# Note: Variables in single quotes are passed literally, so we break out of 
# single quotes to inject the parent shell variables safely: '"$VAR"'
sudo -u starterscriptuser bash -c '
set -e
set -x

export PATH=/usr/local/google-cloud-sdk/bin:/usr/local/go/bin:$PATH
export HOME=/home/'"$USERNAME"'

# Import environment variables
ZONE_NAME='"$ZONE_NAME"'
RUN_ON_ZB_ONLY='"$RUN_ON_ZB_ONLY"'
RUN_READ_CACHE_TESTS_ONLY='"$RUN_READ_CACHE_TESTS_ONLY"'

cd $HOME

# Checkout Repo
git clone https://github.com/googlecloudplatform/gcsfuse
cd gcsfuse
git checkout $(sed -n 2p ~/details.txt)

# ------------------------------------------------------------------
# CONFIGURATION FOR NEW SCRIPT
# ------------------------------------------------------------------

# 1. Determine Region from Zone (e.g., us-west1-b -> us-west1)
# New script takes --bucket-location, usually the region.
REGION=${ZONE_NAME%-*}

# 2. Build the Command
# We rely on "tools/integration_tests/run_e2e_tests.sh" which is the new script.
# We pass --test-installed-package because we installed the deb/rpm above.
# We pass --no-build-binary-in-script because we want to test that installed package.

TEST_SCRIPT="./tools/integration_tests/run_e2e_tests.sh"
chmod +x $TEST_SCRIPT

ARGS="--bucket-location $REGION --test-installed-package --no-build-binary-in-script"

if [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
    ARGS="$ARGS --zonal"
fi

if [[ "$RUN_READ_CACHE_TESTS_ONLY" == "true" ]]; then
    # Note: The new script strictly defines packages in its source. 
    # If specific filtering is needed, it relies on the script internal groups.
    # For now, we log this intent.
    echo "Notice: RUN_READ_CACHE_TESTS_ONLY is set. The new script will run the standard suite for the selected bucket type."
fi

echo "----------------------------------------------------------------"
echo "DELEGATING TO NEW E2E SCRIPT"
echo "Command: $TEST_SCRIPT $ARGS"
echo "----------------------------------------------------------------"

# Capture exit code, but do not exit immediately on failure to allow log upload
set +e
$TEST_SCRIPT $ARGS > ~/e2e_run_logs.txt 2>&1
EXIT_CODE=$?
set -e

echo "E2E Script finished with Exit Code: $EXIT_CODE"

# ------------------------------------------------------------------
# LOG UPLOAD (Preserving Legacy Pipeline Behavior)
# ------------------------------------------------------------------
RELEASE_VERSION=$(sed -n 1p ~/details.txt)
COMMIT_HASH=$(sed -n 3p ~/details.txt)
GCS_DEST="gs://gcsfuse-release-packages/v${RELEASE_VERSION}/${COMMIT_HASH}/"

# Upload the consolidated log
gcloud storage cp ~/e2e_run_logs.txt "${GCS_DEST}combined_e2e_logs.txt"

# If success, create and upload success markers matching old script behavior
if [ $EXIT_CODE -eq 0 ]; then
    touch ~/success.txt
    if [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
        # Legacy script expected success-zonal.txt for zonal runs
        touch ~/success-zonal.txt
        gcloud storage cp ~/success-zonal.txt "${GCS_DEST}"
    else
        # Legacy script expected success.txt for standard runs
        gcloud storage cp ~/success.txt "${GCS_DEST}"
    fi
else
    echo "Tests failed. Check combined_e2e_logs.txt in GCS bucket."
fi

# Exit with the actual test code so the VM/Job fails appropriately
exit $EXIT_CODE
'
