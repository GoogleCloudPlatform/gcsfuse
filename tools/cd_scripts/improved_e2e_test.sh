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
    # For Debian/Ubuntu-based systems
    sudo apt-get update && sudo apt-get install -y wget
elif command -v yum &> /dev/null; then
    # For RHEL/CentOS-based systems
    sudo yum install -y wget
else
    exit 1
fi

# Upgrade gcloud
echo "Upgrade gcloud version"
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk

# Conditionally install python3.11 and run gcloud installer with it for RHEL 8 and Rocky 8
INSTALL_COMMAND="sudo /usr/local/google-cloud-sdk/install.sh --quiet"
if [ -f /etc/os-release ]; then
    . /etc/os-release
    if [[ ($ID == "rhel" || $ID == "rocky") ]]; then
        sudo yum install -y python311
        export CLOUDSDK_PYTHON=/usr/bin/python3.11
        INSTALL_COMMAND="sudo env CLOUDSDK_PYTHON=/usr/bin/python3.11 /usr/local/google-cloud-sdk/install.sh --quiet"
    fi
fi
$INSTALL_COMMAND

export PATH=/usr/local/google-cloud-sdk/bin:$PATH
gcloud version && rm gcloud.tar.gz

# Extract the metadata parameters passed, for which we need the zone of the GCE VM
# on which the tests are supposed to run.
ZONE=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone)
echo "Got ZONE=\"${ZONE}\" from metadata server."
# The format for the above extracted zone is projects/{project-id}/zones/{zone}, thus, from this
# need extracted zone name.
ZONE_NAME=$(basename "$ZONE")

CUSTOM_BUCKET=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.custom_bucket)')
RUN_ON_ZB_ONLY=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.run-on-zb-only)')

# If CUSTOM_BUCKET is empty, use the release-packages bucket as default. When a custom bucket is provided, this script
# will use the provided bucket to fetch details.txt file for the runa nd will upload the results to that bucket,
# specially useful for testing.
BUCKET_NAME_TO_USE=${CUSTOM_BUCKET:-"gcsfuse-release-packages"}
echo "BUCKET_NAME_TO_USE set to: \"${BUCKET_NAME_TO_USE}\""
echo "RUN_ON_ZB_ONLY flag set to : \"${RUN_ON_ZB_ONLY}\""

#details.txt file contains the release version and commit hash of the current release.
# Using dynamic bucket.
gcloud storage cp gs://${BUCKET_NAME_TO_USE}/version-detail/details.txt .
# Writing VM instance name to details.txt (Format: release-test-<os-name>)
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >>details.txt

# Function to create the local user
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
    # For Ubuntu and Debian
    sudo adduser --disabled-password --home "${HOMEDIR}" --gecos "" "${USERNAME}"
  elif grep -qi -E 'rhel|centos|rocky' "$DETAILS"; then
    # For RHEL, CentOS, Rocky Linux
    sudo adduser --home-dir "${HOMEDIR}" "${USERNAME}" && sudo passwd -d "${USERNAME}"
  else
    echo "Unsupported OS type in details file." >&2
    return 1
  fi
  local exit_code=$?

  if [ ${exit_code} -eq 0 ]; then
    echo "User ${USERNAME} created successfully."
  else
    echo "Failed to create user ${USERNAME}." >&2
  fi
  return ${exit_code}
}

# Function to grant sudo access by creating a file in /etc/sudoers.d/
grant_sudo() {
  local USERNAME=$1
  if ! id "${USERNAME}" &>/dev/null; then
    echo "User ${USERNAME} does not exist. Cannot grant sudo."
    return 1
  fi
  
  sudo mkdir -p /etc/sudoers.d/
  SUDOERS_FILE="/etc/sudoers.d/${USERNAME}"
  
  if sudo test -f "${SUDOERS_FILE}"; then
    echo "Sudoers file ${SUDOERS_FILE} already exists."
  else
    echo "Granting ${USERNAME} NOPASSWD sudo access..."
    # Create the sudoers file with the correct content
    if ! echo "${USERNAME} ALL=(ALL:ALL) NOPASSWD:ALL" | sudo tee "${SUDOERS_FILE}" > /dev/null; then
      echo "Failed to create sudoers file." >&2
      return 1
    fi

    # Set the correct permissions on the sudoers file
    if ! sudo chmod 440 "${SUDOERS_FILE}"; then
      echo "Failed to set permissions on sudoers file." >&2
      # Attempt to clean up the partially created file
      sudo rm -f "${SUDOERS_FILE}"
      return 1
    fi
    echo "Sudo access granted to ${USERNAME} via ${SUDOERS_FILE}."
  fi
  return 0
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
echo "User: $USER" &>> ${LOG_FILE}
echo "Current Working Directory: $(pwd)" &>> ${LOG_FILE}

# Root-level parsing for installation
VERSION=$(sed -n 1p details.txt)
COMMIT_HASH=$(sed -n 2p details.txt)
VM_INSTANCE_NAME=$(sed -n 3p details.txt)

if grep -q ubuntu details.txt || grep -q debian details.txt; then
    architecture=$(dpkg --print-architecture)
    sudo apt update
    sudo apt install -y fuse wget git build-essential
    
    # Only download and install the pre-built deb package if using the default release bucket.
    if [[ "${BUCKET_NAME_TO_USE}" == "gcsfuse-release-packages" ]]; then
        echo "Downloading pre-built debian package from release bucket..." &>> ${LOG_FILE}
        gcloud storage cp gs://${BUCKET_NAME_TO_USE}/v${VERSION}/gcsfuse_${VERSION}_${architecture}.deb . &>> ${LOG_FILE}
        sudo dpkg -i gcsfuse_${VERSION}_${architecture}.deb |& tee -a ${LOG_FILE}
    else
        echo "Custom bucket detected (${BUCKET_NAME_TO_USE}); skipping pre-built debian package installation." &>> ${LOG_FILE}
    fi
else
    # RHEL/CentOS
    # Set CLOUDSDK_PYTHON to python3.11 for gcloud commands to work.
    export CLOUDSDK_PYTHON=/usr/bin/python3.11
    uname=$(uname -m)
    if [[ $uname == "x86_64" ]]; then architecture="amd64"; elif [[ $uname == "aarch64" ]]; then architecture="arm64"; fi

    sudo yum makecache
    sudo yum -y update
    sudo yum -y install fuse architecture git gcc gcc-c++ make
    
  # Only download and install the pre-built rpm package if using the default release bucket.
    if [[ "${BUCKET_NAME_TO_USE}" == "gcsfuse-release-packages" ]]; then
        echo "Downloading pre-built rpm package from release bucket..." &>> ${LOG_FILE}
        gcloud storage cp gs://${BUCKET_NAME_TO_USE}/v${VERSION}/gcsfuse-${VERSION}-1.${uname}.rpm . &>> ${LOG_FILE}
        sudo yum -y localinstall gcsfuse-${VERSION}-1.${uname}.rpm &>> ${LOG_FILE}
    else
        echo "Custom bucket detected (${BUCKET_NAME_TO_USE}); skipping pre-built rpm package installation." &>> ${LOG_FILE}
    fi
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
# Print commands and their arguments as they are executed.
set -x

# GCSFuse test suite uses this environment variable to save failure logs at the specified location.
export KOKORO_ARTIFACTS_DIR=/home/starterscriptuser/failure_logs
mkdir -p "$KOKORO_ARTIFACTS_DIR"

export PATH=/usr/local/google-cloud-sdk/bin:/usr/local/go/bin:$PATH
export HOME=/home/'"$USERNAME"'

 # Exporting variables to the sub-shell
export ZONE_NAME='"$ZONE_NAME"'
export LOG_FILE='"$LOG_FILE"'
export RUN_ON_ZB_ONLY='"$RUN_ON_ZB_ONLY"'
export BUCKET_NAME_TO_USE='"$BUCKET_NAME_TO_USE"'
export COMMIT_HASH='"$COMMIT_HASH"'
export VERSION='"$VERSION"'
export architecture='"$architecture"'

cd $HOME

# Checkout Repo
git clone https://github.com/googlecloudplatform/gcsfuse
cd gcsfuse
git checkout ${COMMIT_HASH} |& tee -a ${LOG_FILE}
if [[ "${BUCKET_NAME_TO_USE}" != "gcsfuse-release-packages" ]]; then
    echo "Installing GCSFuse from source..."
    GOOS=linux GOARCH=${architecture} go run tools/build_gcsfuse/main.go . . "${VERSION}"
    sudo cp bin/* /usr/bin/
    sudo cp sbin/* /usr/sbin/
fi

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

TEST_SCRIPT="./tools/integration_tests/improved_run_e2e_tests.sh"
chmod +x $TEST_SCRIPT

ARGS="--bucket-location $REGION --test-installed-package --no-build-binary-in-script"

if [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
    ARGS="$ARGS --zonal"
fi

echo "----------------------------------------------------------------"
echo "DELEGATING TO NEW E2E SCRIPT"
echo "Command: $TEST_SCRIPT $ARGS"
echo "----------------------------------------------------------------"

# Capture exit code, but do not exit immediately on failure to allow log upload
set +e

# Generate timestamped log filename
TIMESTAMP=$(date +%d-%m-%H-%M)
LOG_FILENAME="e2e_run_logs_${TIMESTAMP}.txt"

$TEST_SCRIPT $ARGS 2>&1 | tee -a "$LOG_FILE"
EXIT_CODE=${PIPESTATUS[0]}
set -e

echo "E2E Script finished with Exit Code: $EXIT_CODE"

# ------------------------------------------------------------------
# LOG UPLOAD (Preserving Legacy Pipeline Behavior)
# ------------------------------------------------------------------
GCS_DEST="gs://${BUCKET_NAME_TO_USE}/v${VERSION}/${COMMIT_HASH}/${VM_INSTANCE_NAME}/"

# Upload the consolidated log with fixed name for pipeline compatibility
gcloud storage cp "$LOG_FILE" "${GCS_DEST}_combined_e2e_logs_${TIMESTAMP}.txt"

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
    echo "Tests failed. Check ${LOG_FILENAME} in VM or combined_e2e_logs.txt in GCS bucket."
fi

# Exit with the actual test code so the VM/Job fails appropriately
exit $EXIT_CODE
'
