#!/bin/bash
#
# Copyright 2024 Google LLC
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

# This script is for running fio/dlio tests using GKE.
# This is a stand-alone script, and can be invoked directly by a user.
# It takes in parameters through environment variables. For learning about them, run this script with `--help` argument.
# For debugging, pass argument `--debug` which will print all the shell commands that runs.
# It fetches GCSFuse and GKE GCSFuse CSI driver code from github, if you don't provide it pre-existing clones of them.
# It installs all the necessary dependencies on its own.
# It creates a GKE cluster and other GCP resources (as needed), based on a number of configuration parameters e.g. gcp-project-name/number, cluster-name, zone (for resource location), machine-type (of node), number of local SSDs.
# It creates fio/dlio tests as helm charts, based on the provided JSON workload configuration file and deploys them on the GKE cluster.
# A sample workload-configuration file is available at https://github.com/GoogleCloudPlatform/gcsfuse/blob/b2286ec3466dd285b2d5ea3be8636a809efbfb1b/perfmetrics/scripts/testing_on_gke/examples/workloads.json#L2 .

# Fail script if any of the commands fail.
set -e

# Print all the shell commands if the user passes argument `--debug`. This is
# useful for debugging the script.
if ([ $# -gt 0 ] && ([ "$1" == "-debug" ] || [ "$1" == "--debug" ])); then
  set -x
fi

# Utilities
function exitWithSuccess() { exit 0; }
function exitWithFailure() { exit 1; }

# Default values, to be used for parameters in case user does not specify them.
# GCP related
readonly DEFAULT_PROJECT_ID="gcs-fuse-test"
readonly DEFAULT_PROJECT_NUMBER=927584127901
readonly DEFAULT_ZONE="us-west1-b"
# GKE cluster related
readonly DEFAULT_CLUSTER_NAME="${USER}-testing-us-west1-1"
readonly DEFAULT_NODE_POOL=default-pool
readonly DEFAULT_MACHINE_TYPE="n2-standard-96"
readonly DEFAULT_NUM_NODES=8
readonly DEFAULT_NUM_SSD=16
readonly DEFAULT_APPNAMESPACE=default
readonly DEFAULT_KSA=default
readonly DEFAULT_USE_CUSTOM_CSI_DRIVER=false
# GCSFuse/GKE GCSFuse CSI Driver source code related
readonly DEFAULT_SRC_DIR="$(realpath .)/src"
readonly csi_driver_github_path=https://github.com/googlecloudplatform/gcs-fuse-csi-driver
readonly csi_driver_branch=main
readonly gcsfuse_github_path=https://github.com/googlecloudplatform/gcsfuse
readonly gcsfuse_branch=garnitin/add-gke-load-testing/v1
# Test runtime configuration
readonly DEFAULT_INSTANCE_ID=${USER}-$(date +%Y%m%d-%H%M%S)
# 5 minutes
readonly DEFAULT_POD_WAIT_TIME_IN_SECONDS=300
# 1 week
readonly DEFAULT_POD_TIMEOUT_IN_SECONDS=604800

function printHelp() {
  echo "Usage guide: "
  echo "[ENV_OPTIONS] "${0}" [ARGS]"
  echo ""
  echo "ENV_OPTIONS (all are optional): "
  echo ""
  # GCP related
  echo "project_id=<project-id default=\"${DEFAULT_PROJECT_ID}\">"
  echo "project_number=<number default=\"${DEFAULT_PROJECT_NUMBER}\">"
  echo "zone=<region-zone default=\"${DEFAULT_ZONE}\">"
  # GKE cluster related
  echo "cluster_name=<cluster-name default=\"${DEFAULT_CLUSTER_NAME}\">"
  echo "node_pool=<pool-name default=\"${DEFAULT_NODE_POOL}\">"
  echo "machine_type=<machine-type default=\"${DEFAULT_MACHINE_TYPE}\">"
  echo "num_nodes=<number from 1-8, default=\"${DEFAULT_NUM_NODES}\">"
  echo "num_ssd=<number from 0-16, default=\"${DEFAULT_NUM_SSD}\">"
  echo "use_custom_csi_driver=<true|false, true means build and use a new custom csi driver using gcsfuse code, default=\"${DEFAULT_USE_CUSTOM_CSI_DRIVER}\">"
  # GCSFuse/GKE GCSFuse CSI Driver source code related
  echo "src_dir=<\"directory/to/clone/github/repos/if/needed\", default=\"${DEFAULT_SRC_DIR}\">"
  echo "gcsfuse_src_dir=<\"/path/of/gcsfuse/src/to/use/if/available\", default=\"${DEFAULT_SRC_DIR}/gcsfuse\">"
  echo "csi_src_dir=<\"/path/of/gcs-fuse-csi-driver/to/use/if/available\", default=\"${DEFAULT_SRC_DIR}\"/gcs-fuse-csi-driver>"
  # Test runtime configuration
  echo "pod_wait_time_in_seconds=<number e.g. 60 for checking pod status every 1 min, default=\"${DEFAULT_POD_WAIT_TIME_IN_SECONDS}\">"
  echo "pod_timeout_in_seconds=<number e.g. 3600 for timing out pod runs, should be more than the value of pod_wait_time_in_seconds, default=\"${DEFAULT_POD_TIMEOUT_IN_SECONDS}\">"
  echo "instance_id=<string, not containing spaces, representing unique id for particular test-run e.g. \"${DEFAULT_INSTANCE_ID}\""
  echo "workload_config=<path/to/workload/configuration/file e.g. /a/b/c.json >"
  echo "output_dir=</absolute/path/to/output/dir, output files will be written at output_dir/fio/output.csv and output_dir/dlio/output.csv>"
  echo ""
  echo ""
  echo ""
  echo "ARGS (all are optional) : "
  echo ""
  echo "--debug     Print out shell commands for debugging. Aliases: -debug "
  echo "--help      Print out this help. Aliases: help , -help , -h , --h"
}

# Print out help if user passes argument `--help`
if ([ $# -gt 0 ] && ([ "$1" == "-help" ] || [ "$1" == "--help" ] || [ "$1" == "-h" ])); then
  printHelp
  exitWithSuccess
fi

# Set environment variables.
# GCP related
test -n "${project_id}" || export project_id=${DEFAULT_PROJECT_ID}
test -n "${project_number}" || export project_number=${DEFAULT_PROJECT_NUMBER}
test -n "${zone}" || export zone=${DEFAULT_ZONE}
# GKE cluster related
test -n "${cluster_name}" || export cluster_name=${DEFAULT_CLUSTER_NAME}
test -n "${node_pool}" || export node_pool=${DEFAULT_NODE_POOL}
test -n "${machine_type}" || export machine_type=${DEFAULT_MACHINE_TYPE}
test -n "${num_nodes}" || export num_nodes=${DEFAULT_NUM_NODES}
test -n "${num_ssd}" || export num_ssd=${DEFAULT_NUM_SSD}
# test -n "${appnamespace}" ||
export appnamespace=${DEFAULT_APPNAMESPACE}
# test -n "${ksa}" ||
export ksa=${DEFAULT_KSA}
test -n "${use_custom_csi_driver}" || export use_custom_csi_driver="${DEFAULT_USE_CUSTOM_CSI_DRIVER}"
# GCSFuse/GKE GCSFuse CSI Driver source code related
(test -n "${src_dir}" && src_dir="$(realpath "${src_dir}")") || export src_dir=${DEFAULT_SRC_DIR}
test -d "${src_dir}" || mkdir -pv "${src_dir}"
(test -n "${gcsfuse_src_dir}" && gcsfuse_src_dir="$(realpath "${gcsfuse_src_dir}")") || export gcsfuse_src_dir="${src_dir}"/gcsfuse
export gke_testing_dir="${gcsfuse_src_dir}"/perfmetrics/scripts/testing_on_gke
(test -n "${csi_src_dir}" && csi_src_dir="$(realpath "${csi_src_dir}")") || export csi_src_dir="${src_dir}"/gcs-fuse-csi-driver
# GCSFuse configuration related - deprecated. Will cause error.
test -z "${gcsfuse_mount_options}" || (echo "gcsfuse_mount_options set by user is a deprecated option. Please set gcsfuseMountOptions in workload objects in workload configuration file in its place." && exit 1)
# Test runtime configuration
test -n "${pod_wait_time_in_seconds}" || export pod_wait_time_in_seconds="${DEFAULT_POD_WAIT_TIME_IN_SECONDS}"
test -n "${pod_timeout_in_seconds}" || export pod_timeout_in_seconds="${DEFAULT_POD_TIMEOUT_IN_SECONDS}"
test -n "${instance_id}" || export instance_id="${DEFAULT_INSTANCE_ID}"

if [[ ${pod_timeout_in_seconds} -le ${pod_wait_time_in_seconds} ]]; then
  echo "pod_timeout_in_seconds (${pod_timeout_in_seconds}) <= pod_wait_time_in_seconds (${pod_wait_time_in_seconds})"
  exitWithFailure
fi

if test -n "${workload_config}"; then
  workload_config="$(realpath "${workload_config}")"
  test -f "${workload_config}"
else
    export workload_config="${gke_testing_dir}"/examples/workloads.json
fi

if test -n "${output_dir}"; then
  output_dir="$(realpath "${output_dir}")"
  test -d "${output_dir}"
else
  export output_dir="${gke_testing_dir}"/examples
fi

function printRunParameters() {
  echo "Running $0 with following parameters:"
  echo ""
  # GCP related
  echo "project_id=\"${project_id}\""
  echo "project_number=\"${project_number}\""
  echo "zone=\"${zone}\""
  # GKE cluster related
  echo "cluster_name=\"${cluster_name}\""
  echo "node_pool=\"${node_pool}\""
  echo "machine_type=\"${machine_type}\""
  echo "num_nodes=\"${num_nodes}\""
  echo "num_ssd=\"${num_ssd}\""
  echo "appnamespace=\"${appnamespace}\""
  echo "ksa=\"${ksa}\""
  echo "use_custom_csi_driver=\"${use_custom_csi_driver}\""
  # GCSFuse/GKE GCSFuse CSI Driver source code related
  echo "src_dir=\"${src_dir}\""
  echo "gcsfuse_src_dir=\"${gcsfuse_src_dir}\""
  echo "csi_src_dir=\"${csi_src_dir}\""
  # Test runtime configuration
  echo "pod_wait_time_in_seconds=\"${pod_wait_time_in_seconds}\""
  echo "pod_timeout_in_seconds=\"${pod_timeout_in_seconds}\""
  echo "instance_id=\"${instance_id}\""
  echo "workload_config=\"${workload_config}\""
  echo "output_dir=\"${output_dir}\""
  echo ""
  echo ""
  echo ""
}

# Install dependencies.
function installDependencies() {
  printf "\nInstalling dependencies ...\n\n"
  # Refresh software repositories.
  sudo apt-get update
  # Get some common software dependencies.
  sudo apt-get install -y apt-transport-https ca-certificates gnupg curl
  # Ensure that realpath is installed.
  which realpath
  # Ensure that make is installed.
  which make || ( sudo apt-get install -y make time && which make )
  # Ensure that go is installed.
  which go || (version=1.22.4 && wget -O go_tar.tar.gz https://go.dev/dl/go${version}.linux-amd64.tar.gz 1>/dev/null && sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz 1>/dev/null && sudo mv go /usr/local && echo $PATH && export PATH=$PATH:/usr/local/go/bin && echo $PATH && echo 'export PATH=$PATH:/usr/local/go/bin'>>~/.bashrc && go version)
  # for some reason, the above is unable to update the value of $PATH, so doing it explicitly below.
  export PATH=$PATH:/usr/local/go/bin
  which go
  # Ensure that python3 is installed.
  which python3 || ( sudo apt-get install -y python3 && which python3 )
  # Install more python tools.
  sudo apt-get -y install python3-dev python3-venv python3-pip
  # Enable python virtual environment.
  python3 -m venv .venv
  source .venv/bin/activate
  # Ensure that pip is installed.
  sudo apt-get install -y pip
  # python3 -m pip install --upgrade pip
  # python3 -m pip --version
  # Ensure that python-absl is installed.
  pip install absl-py
  # Ensure that helm is installed
  which helm || (cd "${src_dir}" && (test -d "./helm" || git clone https://github.com/helm/helm.git) && cd helm && make && ls -lh bin/ && mkdir -pv ~/bin && cp -fv bin/helm ~/bin/ && chmod +x ~/bin/helm && export PATH=$PATH:$HOME/bin && echo $PATH && which helm && cd - && cd -)
  # for some reason, the above is unable to update the value of $PATH, so doing it explicitly below.
  export PATH=$PATH:$HOME/bin
  which helm
  # Ensure that kubectl is installed
  if ! which kubectl; then
    # Install the latest gcloud cli. Find full instructions at https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl .
    # Import the Google Cloud public key (Debian 9+ or Ubuntu 18.04+)
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --yes --dearmor -o /usr/share/keyrings/cloud.google.gpg
    # Add the gcloud CLI distribution URI as a package source (Debian 9+ or Ubuntu 18.04+)
    echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
    # Update and install the gcloud CLI
    sudo apt-get update
    sudo apt-get install -y google-cloud-cli
    # install kubectl
    gcloud components install kubectl || sudo apt-get install -y kubectl
    kubectl version --client
  fi
  # Ensure that gke-gcloud-auth-plugin is installed.
  gke-gcloud-auth-plugin --version || ((gcloud components install gke-gcloud-auth-plugin || sudo apt-get install -y google-cloud-cli-gke-gcloud-auth-plugin) && gke-gcloud-auth-plugin --version)
  # Ensure that docker is installed.
  if ! which docker ; then
    sudo apt install apt-transport-https ca-certificates curl software-properties-common -y
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
    sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
    apt-cache policy docker-ce
    sudo apt install docker-ce -y
  fi
  # Install mash, as it is needed for fetching cpu/memory values for test runs
  # in cloudtop. Even if mash install fails, don't panic, go ahead and install
  # google-cloud-monitoring as an alternative.
  which mash || sudo apt-get install -y monarch-tools || true
  # Ensure that gcloud monitoring tools are installed. This is alternative to
  # mash on gce vm.
  # pip install --upgrade google-cloud-storage
  # pip install --ignore-installed --upgrade google-api-python-client
  # pip install --ignore-installed --upgrade google-cloud
  pip install --upgrade google-cloud-monitoring
  # Ensure that jq is installed.
  which jq || sudo apt-get install -y jq
  # # Ensure sudoless docker is installed.
  # docker ps >/dev/null || (sudo addgroup docker && sudo usermod -aG docker $USER && newgrp docker)
}

# Make sure you have access to the necessary GCP resources. The easiest way to enable it is to use <your-ldap>@google.com as active auth.
function ensureGcpAuthsAndConfig() {
  if ! $(gcloud auth list | grep -q ${USER}); then
    gcloud auth application-default login --no-launch-browser && (gcloud auth list | grep -q ${USER})
  fi
  gcloud config set project ${project_id} && gcloud config list
}

# Verify that the passed machine configuration parameters (machine-type, num-nodes, num-ssd) are compatible.
# This is to fail fast, right at the start of the script, rather than failing at
# cluster/nodepool creation, which takes a lot longer.
# Source of constraints:
# https://cloud.google.com/compute/docs/disks/local-ssd#lssd_disk_options .
function validateMachineConfig() {
  echo "Validating input machine configuration ..."
  local machine_type=${1}
  local num_nodes=${2}
  local num_ssd=${3}

  if test ${num_nodes} -le 0; then
    echo "num_nodes is too low (minimium=1) at "${num_nodes}
  fi

  case "${machine_type}" in
  "n2-standard-96")
    if [ ${num_ssd} -ne 0 -a ${num_ssd} -ne 16 -a ${num_ssd} -ne 24 ]; then
      echo "Unsupported num-ssd "${num_ssd}" with given machine-type "${machine_type}". It should be 0, 16 or 24"
      return 1
    fi
    ;;
  "n2-standard-48")
    if [ ${num_ssd} -ne 0 -a ${num_ssd} -ne 8 -a ${num_ssd} -ne 16 -a ${num_ssd} -ne 24 ]; then
      echo "Unsupported num-ssd "${num_ssd}" with given machine-type "${machine_type}". It should be 0, 8, 16 or 24"
      return 1
    fi
    ;;
  "n2-standard-32")
    if [ ${num_ssd} -ne 0 -a ${num_ssd} -ne 4 -a ${num_ssd} -ne 8 -a ${num_ssd} -ne 16 -a ${num_ssd} -ne 24 ]; then
      echo "Unsupported num-ssd "${num_ssd}" with given machine-type "${machine_type}". It should be 0, 4, 8, 16 or 24"
      return 1
    fi
    ;;
  *) ;;
  esac

  return 0
}

function doesNodePoolExist() {
  local cluster_name=${1}
  local zone=${2}
  local node_pool=${3}
  gcloud container node-pools list --cluster=${cluster_name} --zone=${zone} | grep -owq ${node_pool}
}

function deleteExistingNodePool() {
  local cluster_name=${1}
  local zone=${2}
  local node_pool=${3}
  if doesNodePoolExist ${cluster_name} ${zone} ${node_pool}; then
    gcloud -q container node-pools delete ${node_pool} --cluster ${cluster_name} --zone ${zone}
  fi
}

function resizeExistingNodePool() {
  local cluster_name=${1}
  local zone=${2}
  local node_pool=${3}
  local num_nodes=${4}
  if doesNodePoolExist ${cluster_name} ${zone} ${node_pool}; then
    gcloud -q container clusters resize ${cluster_name} --node-pool=${node_pool} --num-nodes=${num_nodes} --zone ${zone}
  fi
}

function createNewNodePool() {
  local cluster_name=${1}
  local zone=${2}
  local node_pool=${3}
  local machine_type=${4}
  local num_nodes=${5}
  local num_ssd=${6}
  gcloud container node-pools create ${node_pool} --cluster ${cluster_name} --ephemeral-storage-local-ssd count=${num_ssd} --network-performance-configs=total-egress-bandwidth-tier=TIER_1 --machine-type ${machine_type} --zone ${zone} --num-nodes ${num_nodes} --workload-metadata=GKE_METADATA
}

function getMachineTypeInNodePool() {
  local cluster=${1}
  local node_pool=${2}
  gcloud container node-pools describe --cluster=${cluster_name} ${node_pool} | grep -w 'machineType' | tr -s '\t' ' ' | rev | cut -d' ' -f1 | rev
}

function getNumNodesInNodePool() {
  local cluster=${1}
  local node_pool=${2}
  gcloud container node-pools describe --cluster=${cluster_name} ${node_pool} | grep -w 'initialNodeCount' | tr -s '\t' ' ' | rev | cut -d' ' -f1 | rev
}

function getNumLocalSSDsPerNodeInNodePool() {
  local cluster=${1}
  local node_pool=${2}
  gcloud container node-pools describe --cluster=${cluster_name} ${node_pool} | grep -w 'localSsdCount' | tr -s '\t' ' ' | rev | cut -d' ' -f1 | rev
}

function doesClusterExist() {
  local cluster_name=${1}
  gcloud container clusters list | grep -woq ${cluster_name}
}

# Create and set up (or reconfigure) a GKE cluster.
function ensureGkeCluster() {
  echo "Creating/updating cluster ${cluster_name} ..."
  if doesClusterExist ${cluster_name}; then
    gcloud container clusters update ${cluster_name} --location=${zone} --workload-pool=${project_id}.svc.id.goog
  else
    gcloud container --project "${project_id}" clusters create ${cluster_name} --zone "${zone}" --workload-pool=${project_id}.svc.id.goog --machine-type "${machine_type}" --image-type "COS_CONTAINERD" --num-nodes ${num_nodes} --ephemeral-storage-local-ssd count=${num_ssd}
  fi
}

function ensureRequiredNodePoolConfiguration() {
  echo "Creating/updating node-pool ${node_pool} ..."
  function createNodePool() { createNewNodePool ${cluster_name} ${zone} ${node_pool} ${machine_type} ${num_nodes} ${num_ssd}; }
  function deleteNodePool() { deleteExistingNodePool ${cluster_name} ${zone} ${node_pool}; }
  function recreateNodePool() { deleteNodePool && createNodePool; }

  if doesNodePoolExist ${cluster_name} ${zone} ${node_pool}; then

    existing_machine_type=$(getMachineTypeInNodePool ${cluster_name} ${node_pool}})
    num_existing_localssd_per_node=$(getNumLocalSSDsPerNodeInNodePool ${cluster_name} ${node_pool})
    num_existing_nodes=$(getNumNodesInNodePool ${cluster_name} ${node_pool})

    if [ "${existing_machine_type}" != "${machine_type}" ]; then
      echo "cluster "${node_pool}" exists, but machine-type differs, so deleting and re-creating the node-pool."
      recreateNodePool
    elif [ ${num_existing_nodes} -ne ${num_nodes} ]; then
      echo "cluster "${node_pool}" exists, but number of nodes differs, so resizing the node-pool."
      resizeExistingNodePool ${cluster_name} ${zone} ${node_pool} ${num_nodes}
    elif [ ${num_existing_localssd_per_node} -ne ${num_ssd} ]; then
      echo "cluster "${node_pool}" exists, but number of SSDs differs, so deleting and re-creating the node-pool"
      recreateNodePool
    else
      echo "cluster "${node_pool}" already exists"
    fi
  else
    createNodePool
  fi
}

function enableManagedCsiDriverIfNeeded() {
  if ${use_custom_csi_driver}; then
    echo "Disabling csi add-on ..."
    gcloud -q container clusters update ${cluster_name} \
    --update-addons GcsFuseCsiDriver=DISABLED \
    --location=${zone}
  else
    echo "Enabling csi add-on ..."
    gcloud -q container clusters update ${cluster_name} \
      --update-addons GcsFuseCsiDriver=ENABLED \
      --location=${zone}
  fi
}

function activateCluster() {
  echo "Configuring cluster credentials ..."
  gcloud container clusters get-credentials ${cluster_name} --location=${zone}
  kubectl config current-context
}

function createKubernetesServiceAccountForCluster() {
  log="$(kubectl create namespace ${appnamespace} 2>&1)" || [[ "$log" == *"already exists"* ]]
  log="$(kubectl create serviceaccount ${ksa} --namespace ${appnamespace} 2>&1)" || [[ "$log" == *"already exists"* ]]
  kubectl config set-context --current --namespace=${appnamespace}
  # Validate it
  kubectl config view --minify | grep namespace:
}

function addGCSAccessPermissions() {
  test -f "${workload_config}"
  grep -wh '\"bucket\"' "${workload_config}" | cut -d: -f2 | cut -d, -f1 | cut -d \" -f2 | sort | uniq | grep -v ' ' | while read workload_bucket; do
    gcloud storage buckets add-iam-policy-binding gs://${workload_bucket} \
      --member "principal://iam.googleapis.com/projects/${project_number}/locations/global/workloadIdentityPools/${project_id}.svc.id.goog/subject/ns/${appnamespace}/sa/${ksa}" \
      --role "roles/storage.objectUser"
  done
}

function ensureGcsfuseCode() {
  echo "Ensuring we have gcsfuse code ..."
  # clone gcsfuse code if needed
  if ! test -d "${gcsfuse_src_dir}"; then
    cd $(dirname "${gcsfuse_src_dir}") && git clone ${gcsfuse_github_path} && cd "${gcsfuse_src_dir}" && git switch ${gcsfuse_branch} && cd - && cd -
  fi

  test -d "${gke_testing_dir}" || (echo "${gke_testing_dir} does not exist" && exit 1)
}

function ensureGcsFuseCsiDriverCode() {
  echo "Ensuring we have gcs-fuse-csi-driver code ..."
  # clone csi-driver code if needed
  if ! test -d "${csi_src_dir}"; then
    cd $(dirname "${csi_src_dir}") && git clone ${csi_driver_github_path} && cd "${csi_src_dir}" && git switch ${csi_driver_branch} && cd - && cd -
  fi
}

function createCustomCsiDriverIfNeeded() {
  if ${use_custom_csi_driver}; then
    echo "Disabling managed CSI driver ..."
    gcloud -q container clusters update ${cluster_name} --update-addons GcsFuseCsiDriver=DISABLED --location=${zone}

    echo "Building custom CSI driver ..."

    # Create a bucket for storing custom-csi driver.
    test -n "${package_bucket}" || export package_bucket=${USER/google/}-gcsfuse-binary-package
    (gcloud storage buckets list | grep -wqo ${package_bucket}) || (region=$(echo ${zone} | rev | cut -d- -f2- | rev) && gcloud storage buckets create gs://${package_bucket} --location=${region})

    # Build a new gcsfuse binary
    cd "${gcsfuse_src_dir}"
    rm -rfv ./bin ./sbin
    go mod vendor
    GOOS=linux GOARCH=amd64 go run tools/build_gcsfuse/main.go . . v3
    # Copy the binary to a GCS bucket for csi driver build.
    gcloud storage -q cp ./bin/gcsfuse gs://${package_bucket}/linux/amd64/
    gcloud storage -q cp gs://${package_bucket}/linux/amd64/gcsfuse gs://${package_bucket}/linux/arm64/ # needed as build on arm64 doesn't work on cloudtop.
    # clean-up
    rm -rfv "${gcsfuse_src_dir}"/bin "${gcsfuse_src_dir}"/sbin
    cd -

    echo "Installing custom CSI driver ..."
    # Build and install csi driver
    ensureGcsFuseCsiDriverCode
    cd "${csi_src_dir}"
    make uninstall || true
    make generate-spec-yaml
    printf "\nBuilding a new custom CSI driver using the above GCSFuse binary ...\n\n"
    make build-image-and-push-multi-arch REGISTRY=gcr.io/${project_id}/${USER} GCSFUSE_PATH=gs://${package_bucket}
    make install PROJECT=${project_id} REGISTRY=gcr.io/${project_id}/${USER}
    cd -
    # Wait some time after csi driver installation before deploying pods
    # to avoid failures caused by 'the webhook failed to inject the
    # sidecar container into the Pod spec' error.
    printf "\nSleeping 30 seconds after csi custom driver installation before deploying pods ...\n\n"
    sleep 30
  else
    echo ""
    echo "Enabling managed CSI driver ..."
    gcloud -q container clusters update ${cluster_name} --update-addons GcsFuseCsiDriver=ENABLED --location=${zone}
  fi
}

function deleteAllHelmCharts() {
  echo "Deleting all existing helm charts ..."
  helm ls --namespace=${appnamespace} | tr -s '\t' ' ' | cut -d' ' -f1 | tail -n +2 | while read helmchart; do helm uninstall ${helmchart} --namespace=${appnamespace}; done
}

function deleteAllPods() {
  deleteAllHelmCharts

  echo "Deleting all existing pods ..."
  kubectl get pods --namespace=${appnamespace}  | tail -n +2 | cut -d' ' -f1 | while read podname; do kubectl delete pods/${podname} --namespace=${appnamespace} --grace-period=0 --force || true; done
}

function deployAllFioHelmCharts() {
  echo "Deploying all fio helm charts ..."
  cd "${gke_testing_dir}"/examples/fio && python3 ./run_tests.py --workload-config "${workload_config}" --instance-id ${instance_id} --machine-type="${machine_type}" && cd -
}

function deployAllDlioHelmCharts() {
  echo "Deploying all dlio helm charts ..."
  cd "${gke_testing_dir}"/examples/dlio && python3 ./run_tests.py --workload-config "${workload_config}" --instance-id ${instance_id} --machine-type="${machine_type}" && cd -
}

function listAllHelmCharts() {
  echo "Listing all helm charts ..."
  # monitor and debug pods
  helm ls --namespace=${appnamespace}  | tr -s '\t' | cut -f1,5,6

  # Sample output.
  # NAME STATUS CHART \
  # fio-loading-test-100m-randread-gcsfuse-file-cache deployed fio-loading-test-0.1.0
  # gke-dlio-unet3d-100kb-500k-128-gcsfuse-file-cache deployed unet3d-loading-test-0.1.0
}

function waitTillAllPodsComplete() {
  start_epoch=$(date +%s)
  printf "\nScanning and waiting till all pods either complete/fail, or time out (start-time epoch = ${start_epoch} seconds, timeout duration = ${pod_timeout_in_seconds} seconds) ...\n\n"
  while true; do
    cur_epoch=$(date +%s)
    time_till_timeout=$((start_epoch+pod_timeout_in_seconds-cur_epoch))
    if [[ ${time_till_timeout} -lt 0 ]]; then
      printf "\nPod-run timed out!\n\n"
      printf "Clearing all pods created in this run...\n"
      deleteAllPods
      exitWithFailure
    fi
    printf "Checking pods status at ${cur_epoch} seconds:\n"
    printf " -----------------------------------------\n"
    podslist="$(kubectl get pods --namespace=${appnamespace} -o wide)"
    echo "${podslist}"
    num_completed_pods=$(echo "${podslist}" | tail -n +2 | grep -i 'completed\|succeeded' | wc -l)
    if [ ${num_completed_pods} -gt 0 ]; then
      printf ${num_completed_pods}" pod(s) completed.\n"
    fi
    num_noncompleted_pods=$(echo "${podslist}" | tail -n +2 | grep -i -v 'completed\|succeeded\|fail' | wc -l)
    num_failed_pods=$(echo "${podslist}" | tail -n +2 | grep -i 'failed' | wc -l)
    if [ ${num_failed_pods} -gt 0 ]; then
      printf ${num_failed_pods}" pod(s) failed.\n\n"
    fi
    if [ ${num_noncompleted_pods} -eq 0 ]; then
      printf "All pods completed.\n\n"
      break
    else
      printf "\n${num_noncompleted_pods} pod(s) is/are still pending or running (time till timeout=${time_till_timeout} seconds). Will check again in "${pod_wait_time_in_seconds}" seconds. Sleeping for now.\n\n"
      printf "\nYou can take a break too if you want. Just kill this run and connect back to it later, for fetching and parsing outputs, using the following command: \n"
      printf "   only_parse=true instance_id=${instance_id} project_id=${project_id} project_number=${project_number} zone=${zone} machine_type=${machine_type} use_custom_csi_driver=${use_custom_csi_driver} gcsfuse_src_dir=\"${gcsfuse_src_dir}\" csi_src_dir=\"${csi_src_dir}\" pod_wait_time_in_seconds=${pod_wait_time_in_seconds} workload_config=\"${workload_config}\" cluster_name=${cluster_name} output_dir=\"${output_dir}\" $0 \n"
      printf "\nbut remember that this will reset the start-timer for pod timeout.\n\n"
      printf "\nTo ssh to any specific pod, use the following command: \n"
      printf "  gcloud container clusters get-credentials ${cluster_name} --location=${zone}\n"
      printf "  kubectl config set-context --current --namespace=${appnamespace}\n"
      printf "  kubectl exec -it pods/<podname> [-c {gke-gcsfuse-sidecar|fio-tester|dlio-tester}] --namespace=${appnamespace} -- /bin/bash \n"
      printf "\nTo view cpu/memory usage of different pods/containers: \n"
      printf "  kubectl top pod [<podname>] --namespace=${appnamespace} [--containers] \n"
      printf "\nTo view the latest status of all the pods in this cluster/namespace: \n"
      printf "  kubectl get pods --namespace=${appnamespace} [-o wide] [--watch] \n"
      printf "\nTo output the configuration of all or one of the pods in this cluster/namespace (useful for debugging): \n"
      printf "  kubectl get [pods or pods/<podname>] --namespace=${appnamespace} -o yaml \n"

      printf "\n\n\n"
    fi
    sleep ${pod_wait_time_in_seconds}
    unset podslist # necessary to update the value of podslist every iteration
  done
}

function fetchAndParseFioOutputs() {
  echo "Fetching and parsing fio outputs ..."
  cd "${gke_testing_dir}"/examples/fio
  python3 parse_logs.py --project-number=${project_number} --workload-config "${workload_config}" --instance-id ${instance_id} --output-file "${output_dir}"/fio/output.csv --project-id=${project_id} --cluster-name=${cluster_name} --namespace-name=${appnamespace}
  cd -
}

function fetchAndParseDlioOutputs() {
  echo "Fetching and parsing dlio outputs ..."
  cd "${gke_testing_dir}"/examples/dlio
  python3 parse_logs.py --project-number=${project_number} --workload-config "${workload_config}" --instance-id ${instance_id} --output-file "${output_dir}"/dlio/output.csv --project-id=${project_id} --cluster-name=${cluster_name} --namespace-name=${appnamespace}
  cd -
}

# prep
printRunParameters
validateMachineConfig ${machine_type} ${num_nodes} ${num_ssd}
installDependencies

# GCP configuration
ensureGcpAuthsAndConfig
ensureGkeCluster
# ensureRequiredNodePoolConfiguration
enableManagedCsiDriverIfNeeded
activateCluster
createKubernetesServiceAccountForCluster

# GCSFuse driver source code
ensureGcsfuseCode

# GCP/GKE configuration dependent on GCSFuse/CSI driver source code
addGCSAccessPermissions
createCustomCsiDriverIfNeeded

# Run latest workload configuration
deleteAllPods
deployAllFioHelmCharts
deployAllDlioHelmCharts

# monitor pods
listAllHelmCharts
waitTillAllPodsComplete

# clean-up after run
deleteAllPods
fetchAndParseFioOutputs
fetchAndParseDlioOutputs
