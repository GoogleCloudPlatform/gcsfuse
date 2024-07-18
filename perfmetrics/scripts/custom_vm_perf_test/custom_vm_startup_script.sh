#! /bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
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

# This script serves as the startup script for the vm instance created by
# custom_vm_perf_script.py

# Print commands and their arguments as they are executed.
set -x
# Exit immediately if a command exits with a non-zero status.
set -e

VM_INSTANCE=$(hostname)
VM_ZONE=$(curl http://metadata.google.internal/computeMetadata/v1/instance/zone -H Metadata-Flavor:Google | cut '-d/' -f4)
FIO_READ_DIRS=("128kb_read" "256kb_read" "1mb_read" "5mb_read" "10mb_read" "50mb_read" "100mb_read" "200mb_read" "1gb_read")
FIO_WRITE_DIRS=("256kb_write" "1mb_write" "50mb_write" "100mb_write" "1gb_write")

# Function to fetch metadata value of the key.
function fetch_meta_data_value() {
  metadata_key=$1
  # Fetch metadata value of the key
  gcloud compute instances describe $VM_INSTANCE --zone $VM_ZONE --flatten="metadata[$metadata_key]" >>  metadata.txt
  # cat metadata.txt.txt
  # ---
  #   value
  x=$(sed '2!d' metadata.txt)
  #   value(contains preceding spaces)
  # Remove spaces
  # value
  value=$(echo "$x" | sed 's/[[:space:]]//g')
  # echo $value
  # value
  rm metadata.txt
  echo $value
}

echo "Disabling gce-cert-workload refresh timer"
systemctl disable --now gce-workload-cert-refresh.timer

echo "Running update"
sudo apt update

echo "Installing pip"
sudo apt-get install pip -y

#Install fuse
echo "Installing fuse"
sudo apt install -y fuse

sudo apt install -y wget

echo "Installing git"
sudo apt install -y git

echo "Installing python3-setuptools tools"
sudo apt-get install -y gcc python3-dev python3-setuptools
# Downloading composite object requires integrity checking with CRC32c in gsutil.
# it requires to install crcmod.
sudo apt install -y python3-crcmod

echo "Intalling build-essential"
sudo apt install -y build-essential

# install go
echo "Installing Go"
architecture=$(dpkg --print-architecture)
wget -O go_tar.tar.gz https://go.dev/dl/go1.22.4.linux-${architecture}.tar.gz
sudo tar -C /usr/local -xzf go_tar.tar.gz
export PATH=${PATH}:/usr/local/go/bin
echo "Write gcsfuse and go version to log file"
gcsfuse --version |& tee -a ~/logs.txt
go version |& tee -a ~/logs.txt

echo "Clone and checkout gcsfuse repo"
export PATH=${PATH}:/usr/local/go/bin
git clone https://github.com/googlecloudplatform/gcsfuse |& tee -a ~/logs.txt
cd gcsfuse

echo "Installing fio"
"./perfmetrics/scripts/fio/install_fio.sh" .
fio --version

echo "Building and installing gcsfuse"
commitId=$(fetch_meta_data_value "COMMIT_ID")
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

git checkout $commitId
cd perfmetrics/scripts
echo "Mounting gcsfuse"
mkdir -p gcs

UPLOAD_FLAGS="--upload_gs"
GCSFUSE_FIO_FLAGS="--stackdriver-export-interval=30s"
BUCKET_NAME="anushkadhn-perf-tests"
SPREADSHEET_ID="1vFbBhVQ46KclpdOTr2iFZVavAqpOzJgHVjrpmeRZF6A"
MOUNT_POINT=gcs

# The VM will itself exit if the gcsfuse mount fails.sudo cat /var/log/syslog | grep -i -A 20  “Success”
gcsfuse  $GCSFUSE_FIO_FLAGS $BUCKET_NAME $MOUNT_POINT

#deleting files in the read directories if non empty
for dir in "${FIO_READ_DIRS[@]}"; do
  gcloud storage rm gs://$BUCKET_NAME/$dir/?* || echo "No files under directory ${dir}"
done

#deleting files in the write directories if non empty
for dir in "${FIO_WRITE_DIRS[@]}"; do
  gcloud storage rm gs://$BUCKET_NAME/$dir/?* || echo "No files under directory ${dir}"
done

#clearing out the read write directories
echo Print the time when FIO tests start
date
echo Running fio test..
echo "Overall fio start epoch time:" `date +%s`
fio  job_files/custom_vm_perf_test_read_write.fio --lat_percentiles 1 --output-format=json --output="fio-output.json"
echo "Overall fio end epoch time:" `date +%s`
sudo umount $MOUNT_POINT

pip install --upgrade google-cloud
pip install --upgrade google-cloud-bigquery
pip install --upgrade google-cloud-storage

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://anushkadhn-perf-tests/creds.json gsheet
echo Fetching results..
python3 fetch_and_upload_metrics.py "fio-output.json" $UPLOAD_FLAGS --spreadsheet_id=$SPREADSHEET_ID

echo "Success!"