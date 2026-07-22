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

"""Common utilities for GKE tests."""

import asyncio
import json
import os
import shlex
import shutil
import subprocess
import sys
import time

# The Maximum Transmission Unit (MTU) for the network.
DEFAULT_MTU = 8896
# Default configuration for testing on TPU
DEFAULT_PROJECT_ID = "gcs-fuse-test-ml"
DEFAULT_ZONE = "europe-west4-a"
DEFAULT_RESERVATION_NAME = "cloudtpu-20260521143000-1388945208"

_gcloud_apt_repo_setup_done = False

async def _setup_gcloud_apt_repo():
  global _gcloud_apt_repo_setup_done
  if _gcloud_apt_repo_setup_done:
    return
  print("Setting up Google Cloud apt repository...")
  await run_command_async([
      "sudo", "apt", "install", "-y", "apt-transport-https", "ca-certificates", "gnupg", "curl"
  ])
  await run_command_async([
      "bash", "-c",
      "curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --yes --dearmor -o /usr/share/keyrings/cloud.google.gpg"
  ])
  await run_command_async([
      "bash", "-c",
      "echo 'deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main' | sudo tee /etc/apt/sources.list.d/google-cloud-sdk.list"
  ])
  await run_command_async(["sudo", "apt", "update", "-y"])
  _gcloud_apt_repo_setup_done = True

async def _install_gcloud():
  await _setup_gcloud_apt_repo()
  await run_command_async(["sudo", "apt", "install", "-y", "google-cloud-sdk"])

async def _install_git():
  await run_command_async(["sudo", "apt", "install", "-y", "git"])

async def _install_make():
  await run_command_async(["sudo", "apt", "install", "-y", "make"])

async def _install_kubectl():
  await run_command_async(["sudo", "snap", "install", "kubectl", "--classic"])

async def _install_gke_auth_plugin():
  await _setup_gcloud_apt_repo()
  await run_command_async([
      "sudo", "apt", "install", "-y", "google-cloud-sdk-gke-gcloud-auth-plugin"
  ])

# Global dictionary of supported tools, their version commands, and installation functions.
SUPPORTED_TOOLS = {
    "gcloud": {
        "version_cmd": ["gcloud", "--version"],
        "install_func": _install_gcloud,
    },
    "git": {
        "version_cmd": ["git", "--version"],
        "install_func": _install_git,
    },
    "make": {
        "version_cmd": ["make", "--version"],
        "install_func": _install_make,
    },
    "kubectl": {
        "version_cmd": ["kubectl", "version", "--client=true"],
        "install_func": _install_kubectl,
    },
    "gke-gcloud-auth-plugin": {
        "version_cmd": ["gke-gcloud-auth-plugin", "--version"],
        "install_func": _install_gke_auth_plugin,
    },
}


# Global counter to track and number the major execution steps in the test scripts.
_STEP_COUNTER = 1

def log_step(msg):
  """Prints a highly visible banner for a major execution step and increments the step counter."""
  global _STEP_COUNTER
  print(f"\n\n{'=' * 80}\n[STEP {_STEP_COUNTER}] {msg}\n{'=' * 80}\n\n", flush=True)
  _STEP_COUNTER += 1

def log_info(msg):
  """Prints a standard informational message."""
  print(f"[INFO] {msg}", flush=True)

def log_success(msg):
  """Prints a success message, typically used at the end of a successful operation."""
  print(f"[SUCCESS] {msg}", flush=True)

def log_error(msg):
  """Prints an error message to standard error (stderr)."""
  import sys
  print(f"[ERROR] {msg}", file=sys.stderr, flush=True)


async def run_command_async(command_list, check=True, cwd=None, silent=False):
  """Runs a command asynchronously, preventing command injection.

  Args:
      command_list: A list of strings representing the command and its
        arguments.
      check: If True, raises CalledProcessError if the command returns a
        non-zero exit code.
      cwd: The working directory to run the command in.

  Returns:
      A tuple containing (stdout, stderr, returncode).

  Raises:
      subprocess.CalledProcessError: If the command fails and check is True.
  """
  command_str = " ".join(map(shlex.quote, command_list))
  if not silent:
    print(f"Executing: {command_str}")
    sys.stdout.flush()
  start_time = time.time()
  
  process = await asyncio.create_subprocess_exec(
      *command_list,
      stdout=asyncio.subprocess.PIPE,
      stderr=asyncio.subprocess.PIPE,
      cwd=cwd,
  )

  async def _progress_ticker():
    start = time.time()
    while True:
      await asyncio.sleep(60)
      if process.returncode is not None:
        break
      elapsed = int((time.time() - start) / 60)
      cmd_short = command_str if len(command_str) < 80 else command_str[:77] + "..."
      print(f"[INFO] Still executing ({elapsed}m elapsed): {cmd_short}", flush=True)

  ticker_task = asyncio.create_task(_progress_ticker())

  try:
    stdout, stderr = await process.communicate()
  finally:
    ticker_task.cancel()
  stdout_decoded = stdout.decode().strip()
  stderr_decoded = stderr.decode().strip()

  duration = time.time() - start_time
  mins = int(duration // 60)
  secs = int(duration % 60)
  duration_str = f"{mins}m {secs}s" if mins > 0 else f"{secs}s"

  if process.returncode != 0:
    if not silent and check:
      print(f"Failed: {command_str} (took {duration_str})", file=sys.stderr)
      print(f"--- STDOUT ---\n{stdout_decoded}\n--- STDERR ---\n{stderr_decoded}\n", file=sys.stderr)
      sys.stderr.flush()
    if check:
      raise subprocess.CalledProcessError(
          process.returncode, command_str, stdout_decoded, stderr_decoded
      )
  else:
    if not silent:
      print(f"Completed: {command_str} (took {duration_str})")
      sys.stdout.flush()

  return stdout_decoded, stderr_decoded, process.returncode


async def check_prerequisites(tools_to_check=None):
  """Checks for required command-line tools.

  Args:
      tools_to_check: A list of tool names to check. If None, checks all keys in SUPPORTED_TOOLS.

  Verifies that tools are installed. If missing, it attempts to install them
  using the command defined in SUPPORTED_TOOLS. Exits the script if any other
  required tool is not found.
  """
  log_step("Checking prerequisites")
  if tools_to_check is None:
    tools_to_check = list(SUPPORTED_TOOLS.keys())

  log_info(f"Checking for required tools: {', '.join(tools_to_check)}...")

  for tool in tools_to_check:
    path = shutil.which(tool)
    if path is None:
      if tool not in SUPPORTED_TOOLS or "install_func" not in SUPPORTED_TOOLS[tool]:
        log_error(f"Required tool '{tool}' is not installed and no install command is known.")
        sys.exit(1)
        
      log_info(f"{tool} not found. Attempting to install...")
      try:
        await SUPPORTED_TOOLS[tool]["install_func"]()
        path = shutil.which(tool)
        if path is None:
          raise FileNotFoundError(f"{tool} still not found after installation.")
      except (FileNotFoundError, subprocess.CalledProcessError) as e:
        log_error(f"Failed to install {tool}: {e}")
        sys.exit(1)
        
    log_info(f"Found {tool} at {path}")
    if tool in SUPPORTED_TOOLS and "version_cmd" in SUPPORTED_TOOLS[tool]:
      try:
        stdout, _, _ = await run_command_async(SUPPORTED_TOOLS[tool]["version_cmd"], silent=True)
        # Print only the first line of version output to keep logs clean
        version_info = stdout.splitlines()[0] if stdout else "Unknown version"
        log_info(f"Version: {version_info}")
      except subprocess.CalledProcessError:
        log_error(f"Version: Could not determine version")
  log_success("All required tools are installed.")





async def setup_gke_cluster(
    project_id,
    zone,
    cluster_name,
    network_name,
    subnet_name,
    region,
    machine_type,
    node_pool_name,
    reservation_name=None,
):
  """Sets up the GKE cluster and required node pool.

  This function ensures a GKE cluster and a specific node pool are ready for
  the test. It will create the cluster, network, and node pool if they
  don't exist. If the node pool exists but is unhealthy, it will be recreated.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone for the cluster and node pool.
      cluster_name: The name of the GKE cluster.
      network_name: The name of the VPC network.
      subnet_name: The name of the VPC subnet.
      region: The GCP region for the network.
      machine_type: The machine type for the node pool.
      node_pool_name: The name of the node pool.
      reservation_name: The specific reservation to use for the nodes.
  """
  # The user confirmed their existing logic handles node pool creation 
  # without needing manual capacity waiting.

  log_step(f"Setting up GKE cluster '{cluster_name}' in zone '{zone}'...")
  if await get_cluster_async(project_id, zone, cluster_name):
    log_info(f"Cluster '{cluster_name}' already exists.")
    if await get_node_pool_async(
        project_id, zone, cluster_name, node_pool_name
    ):
      log_info(f"Node pool '{node_pool_name}' exists.")
      status = await get_node_pool_status_async(
          project_id, zone, cluster_name, node_pool_name
      )
      
      wait_time = 0
      while status in ("PROVISIONING", "RECONCILING", "STOPPING") and wait_time < 600:
        log_info(f"Node pool is {status}. Waiting 30s for it to become RUNNING...")
        await asyncio.sleep(30)
        wait_time += 30
        status = await get_node_pool_status_async(
            project_id, zone, cluster_name, node_pool_name
        )

      if status != "RUNNING":
        log_info(f"Node pool '{node_pool_name}' is unhealthy (status: {status}). Recreating...")
        await delete_node_pool_async(
            project_id, zone, cluster_name, node_pool_name
        )
        await create_node_pool_async(
            project_id,
            zone,
            cluster_name,
            node_pool_name,
            machine_type,
            reservation_name,
        )
    else:
      log_info(f"Creating node pool '{node_pool_name}'...")
      await create_node_pool_async(
          project_id,
          zone,
          cluster_name,
          node_pool_name,
          machine_type,
          reservation_name,
      )
  else:
    log_info(f"Creating network '{network_name}' and subnet '{subnet_name}'...")
    await create_network(project_id, network_name, subnet_name, region)
    log_info(f"Creating cluster '{cluster_name}'...")
    await create_cluster_async(project_id, zone, cluster_name, network_name, subnet_name)
    print(f"Creating node pool '{node_pool_name}'...")
    await create_node_pool_async(
        project_id,
        zone,
        cluster_name,
        node_pool_name,
        machine_type,
        reservation_name,
    )

  # Get credentials for the cluster to allow kubectl to connect.
  print("Fetching cluster endpoint and auth data.")
  await get_cluster_credentials_async(project_id, zone, cluster_name)
  print("GKE cluster setup complete.")


async def create_cluster_async(project_id, zone, cluster_name, network_name, subnet_name):
  """Creates a new GKE cluster."""
  await run_command_async([
      "gcloud", "container", "clusters", "create", cluster_name,
      f"--project={project_id}", f"--zone={zone}", f"--network={network_name}",
      f"--subnetwork={subnet_name}", f"--workload-pool={project_id}.svc.id.goog",
      "--addons=GcsFuseCsiDriver", "--num-nodes=1"
  ])

async def get_cluster_credentials_async(project_id, zone, cluster_name):
  """Fetches cluster endpoint and auth data."""
  await run_command_async([
      "gcloud", "container", "clusters", "get-credentials", cluster_name,
      f"--project={project_id}", f"--zone={zone}"
  ])

async def get_cluster_async(project_id, zone, cluster_name):
  """Checks if a GKE cluster exists.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone where the cluster is located.
      cluster_name: The name of the GKE cluster.

  Returns:
      True if the cluster exists, False otherwise.
  """
  _, _, returncode = await run_command_async([
      "gcloud", "container", "clusters", "describe", cluster_name,
      f"--project={project_id}", f"--zone={zone}", "--format=value(name)"
  ], check=False)
  return returncode == 0


async def get_node_pool_async(project_id, zone, cluster_name, node_pool_name):
  """Checks if a node pool exists in a GKE cluster.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone where the cluster is located.
      cluster_name: The name of the GKE cluster.
      node_pool_name: The name of the node pool.

  Returns:
      True if the node pool exists, False otherwise.
  """
  _, _, returncode = await run_command_async([
      "gcloud", "container", "node-pools", "describe", node_pool_name,
      f"--project={project_id}", f"--zone={zone}", f"--cluster={cluster_name}",
      "--format=value(name)"
  ], check=False)
  return returncode == 0


async def get_node_pool_status_async(
    project_id, zone, cluster_name, node_pool_name
):
  """Gets a node pool's status.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone where the cluster is located.
      cluster_name: The name of the GKE cluster.
      node_pool_name: The name of the node pool.

  Returns:
      The node pool status string, or None if it fails.
  """
  status, _, returncode = await run_command_async([
      "gcloud", "container", "node-pools", "describe", node_pool_name,
      f"--project={project_id}", f"--zone={zone}", f"--cluster={cluster_name}",
      "--format=value(status)"
  ], check=False)
  if returncode == 0:
    return status
  return None


async def create_node_pool_async(
    project_id,
    zone,
    cluster_name,
    node_pool_name,
    machine_type,
    reservation_name=None,
):
  """Creates a new node pool.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone for the node pool.
      cluster_name: The name of the GKE cluster.
      node_pool_name: The name for the new node pool.
      machine_type: The machine type for the nodes in the pool.
      reservation_name: The specific reservation to use for the nodes.
  """
  args = [
      "gcloud", "container", "node-pools", "create", node_pool_name,
      f"--project={project_id}", f"--cluster={cluster_name}", f"--zone={zone}",
      f"--machine-type={machine_type}", "--num-nodes=1",
      "--scopes=https://www.googleapis.com/auth/cloud-platform",
  ]
  if reservation_name:
    args.extend([
        f"--reservation-affinity=specific",
        f"--reservation={reservation_name}",
    ])
  await run_command_async(args)


async def delete_node_pool_async(
    project_id, zone, cluster_name, node_pool_name
):
  """Deletes an existing node pool.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone where the node pool is located.
      cluster_name: The name of the GKE cluster.
      node_pool_name: The name of the node pool to delete.
  """
  args = [
      "gcloud", "container", "node-pools", "delete", node_pool_name,
      f"--project={project_id}", f"--cluster={cluster_name}", f"--zone={zone}",
      "--quiet",
  ]
  for attempt in range(3):
    _, _, returncode = await run_command_async(args, check=False)
    if returncode == 0:
      return
    print(f"Deletion failed on attempt {attempt+1}. Retrying in 30s...")
    await asyncio.sleep(30)
  
  # Final attempt with check=True to raise error if it still fails
  await run_command_async(args, check=True)


async def get_network_mtu_async(project_id, network_name):
  """Gets the MTU of a VPC network. Returns the MTU string or None if not found."""
  stdout, _, net_rc = await run_command_async([
      "gcloud", "compute", "networks", "describe", network_name,
      f"--project={project_id}", "--format=value(mtu)"
  ], check=False, silent=True)
  if net_rc == 0:
    return stdout.strip()
  return None


async def update_network_mtu_async(project_id, network_name, mtu):
  """Updates the MTU of a VPC network."""
  await run_command_async([
      "gcloud", "compute", "networks", "update", network_name,
      f"--project={project_id}", f"--mtu={mtu}"
  ])


async def create_network_vpc_async(project_id, network_name, mtu):
  """Creates a custom VPC network."""
  await run_command_async([
      "gcloud", "compute", "networks", "create", network_name,
      f"--project={project_id}", "--subnet-mode=custom", f"--mtu={mtu}"
  ])


async def get_subnet_async(project_id, region, subnet_name):
  """Checks if a subnet exists."""
  _, _, sub_rc = await run_command_async([
      "gcloud", "compute", "networks", "subnets", "describe", subnet_name,
      f"--project={project_id}", f"--region={region}", "--format=value(name)"
  ], check=False, silent=True)
  return sub_rc == 0


async def create_subnet_async(project_id, region, network_name, subnet_name):
  """Creates a subnet in a VPC network."""
  await run_command_async([
      "gcloud", "compute", "networks", "subnets", "create", subnet_name,
      f"--project={project_id}", f"--network={network_name}",
      "--range=10.0.0.0/24", f"--region={region}"
  ])


async def create_network(project_id, network_name, subnet_name, region):
  """Creates a new network and subnet if they do not exist, or updates MTU if incorrect."""
  current_mtu = await get_network_mtu_async(project_id, network_name)
  
  if current_mtu is not None:
    if current_mtu == str(DEFAULT_MTU):
      log_info(f"Network '{network_name}' already exists with correct MTU ({DEFAULT_MTU}). Skipping creation.")
    else:
      log_info(f"Network '{network_name}' exists but has incorrect MTU ({current_mtu}). Updating to {DEFAULT_MTU}...")
      await update_network_mtu_async(project_id, network_name, DEFAULT_MTU)
  else:
    log_info(f"Creating network '{network_name}'...")
    await create_network_vpc_async(project_id, network_name, DEFAULT_MTU)

  if await get_subnet_async(project_id, region, subnet_name):
    log_info(f"Subnet '{subnet_name}' already exists. Skipping creation.")
  else:
    log_info(f"Creating subnet '{subnet_name}'...")
    await create_subnet_async(project_id, region, network_name, subnet_name)


async def cleanup(project_id, zone, cluster_name, network_name, subnet_name):
  """No-op cleanup to preserve the long-running GKE cluster and network."""
  log_info("Skipping GKE and network cleanup to maintain long-running infrastructure.")


# GCSFuse Build and Deploy
async def clone_and_log_branch_info(branch, temp_dir):
  """Clones GCSFuse, checks out the correct commit, and logs the branch info."""
  log_step("Fetching GCSFuse source and commit info")
  gcsfuse_dir = os.path.join(temp_dir, "gcsfuse")
  await run_command_async([
      "git", "clone", "-b", branch, "https://github.com/GoogleCloudPlatform/gcsfuse.git", gcsfuse_dir
  ])
  
  initiator = os.environ.get("KOKORO_BUILD_INITIATOR", "")
  if initiator == "kokoro":
    # If run in Kokoro, pick the latest commit before yesterday to ensure stability
    stdout, _, _ = await run_command_async([
        "git", "log", "--before=yesterday 23:59:59", "--max-count=1", "--pretty=%H"
    ], cwd=gcsfuse_dir)
  else:
    stdout, _, _ = await run_command_async([
        "git", "log", "-n", "1", "--pretty=%H"
    ], cwd=gcsfuse_dir)
  
  commit_id = stdout.strip()
  log_info(f"GCSFuse branch: '{branch}', commit ID: {commit_id} (initiator: {initiator})")
  await run_command_async(["git", "checkout", commit_id], cwd=gcsfuse_dir)

# GCSFuse Build and Deploy
async def build_gcsfuse_image(project_id, temp_dir, staging_version):
  """Builds the CSI driver image from the already cloned repository.

  Args:
      project_id: The Google Cloud project ID.
      temp_dir: A temporary directory where the repository is cloned.
      staging_version: The staging version for the image.
  """
  gcsfuse_dir = os.path.join(temp_dir, "gcsfuse")
  await run_command_async([
      "make", "build-csi", f"PROJECT={project_id}", f"STAGINGVERSION={staging_version}"
  ], cwd=gcsfuse_dir)

