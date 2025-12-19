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
import shlex
import subprocess
import sys


async def run_command_async(command_list, check=True, cwd=None):
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
  print(f"Executing command: {command_str}")
  process = await asyncio.create_subprocess_exec(
      *command_list,
      stdout=asyncio.subprocess.PIPE,
      stderr=asyncio.subprocess.PIPE,
      cwd=cwd,
  )
  stdout, stderr = await process.communicate()
  stdout_decoded = stdout.decode().strip()
  stderr_decoded = stderr.decode().strip()

  if check and process.returncode != 0:
    raise subprocess.CalledProcessError(
        process.returncode, command_str, stdout_decoded, stderr_decoded
    )

  print(stdout_decoded)
  print(stderr_decoded, file=sys.stderr)
  sys.stdout.flush()
  return stdout_decoded, stderr_decoded, process.returncode


async def get_cluster_async(project_id, zone, cluster_name):
  """Checks if a GKE cluster exists.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone where the cluster is located.
      cluster_name: The name of the GKE cluster.

  Returns:
      True if the cluster exists, False otherwise.
  """
  cmd = [
      "gcloud",
      "container",
      "clusters",
      "describe",
      cluster_name,
      f"--project={project_id}",
      f"--zone={zone}",
      "--format=value(name)",
  ]
  _, _, returncode = await run_command_async(cmd, check=False)
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
  cmd = [
      "gcloud",
      "container",
      "node-pools",
      "describe",
      node_pool_name,
      f"--project={project_id}",
      f"--zone={zone}",
      f"--cluster={cluster_name}",
      "--format=value(name)",
  ]
  _, _, returncode = await run_command_async(cmd, check=False)
  return returncode == 0


async def is_node_pool_healthy_async(
    project_id, zone, cluster_name, node_pool_name
):
  """Checks if a node pool's status is RUNNING.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone where the cluster is located.
      cluster_name: The name of the GKE cluster.
      node_pool_name: The name of the node pool.

  Returns:
      True if the node pool status is 'RUNNING', False otherwise.
  """
  cmd = [
      "gcloud",
      "container",
      "node-pools",
      "describe",
      node_pool_name,
      f"--project={project_id}",
      f"--zone={zone}",
      f"--cluster={cluster_name}",
      "--format=value(status)",
  ]
  status, _, returncode = await run_command_async(cmd, check=False)
  return returncode == 0 and status == "RUNNING"


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
  cmd = [
      "gcloud",
      "container",
      "node-pools",
      "create",
      node_pool_name,
      f"--project={project_id}",
      f"--cluster={cluster_name}",
      f"--zone={zone}",
      f"--machine-type={machine_type}",
      "--num-nodes=1",
      "--scopes=https://www.googleapis.com/auth/cloud-platform",
  ]
  if reservation_name:
    cmd.extend([
        f"--reservation-affinity=specific",
        f"--reservation={reservation_name}",
    ])
  await run_command_async(cmd)


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
  cmd = [
      "gcloud",
      "container",
      "node-pools",
      "delete",
      node_pool_name,
      f"--project={project_id}",
      f"--cluster={cluster_name}",
      f"--zone={zone}",
      "--quiet",
  ]
  await run_command_async(cmd, check=False)


async def create_network(project_id, network_name, subnet_name, region, mtu):
  """Creates a new network and subnet if they don't exist.

  Args:
      project_id: The Google Cloud project ID.
      network_name: The name for the new VPC network.
      subnet_name: The name for the new subnet.
      region: The GCP region for the subnet.
      mtu: The Maximum Transmission Unit (MTU) for the network.
  """
  await run_command_async(
      [
          "gcloud",
          "compute",
          "networks",
          "create",
          network_name,
          f"--project={project_id}",
          "--subnet-mode=custom",
          f"--mtu={mtu}",
      ],
      check=False,
  )
  await run_command_async(
      [
          "gcloud",
          "compute",
          "networks",
          "subnets",
          "create",
          subnet_name,
          f"--project={project_id}",
          f"--network={network_name}",
          "--range=10.0.0.0/24",
          f"--region={region}",
      ],
      check=False,
  )


async def cleanup(project_id, zone, cluster_name, network_name, subnet_name):
  """Cleans up the created GKE, network, and firewall resources.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone where the resources are located.
      cluster_name: The name of the GKE cluster to delete.
      network_name: The name of the VPC network to delete.
      subnet_name: The name of the subnet to delete.
  """
  print("Cleaning up GKE and network resources...")
  # First, delete the cluster, which is the primary user of the firewall rules.
  await run_command_async(
      [
          "gcloud",
          "container",
          "clusters",
          "delete",
          cluster_name,
          f"--project={project_id}",
          f"--zone={zone}",
          "--quiet",
      ],
      check=False,
  )

  # Find and delete firewall rules associated with the network.
  print(f"Finding and deleting firewall rules for network '{network_name}'...")
  list_fw_cmd = [
      "gcloud",
      "compute",
      "firewall-rules",
      "list",
      f"--project={project_id}",
      f"--filter=network~/{network_name}$",
      "--format=value(name)",
  ]
  fw_rules_str, _, returncode = await run_command_async(
      list_fw_cmd, check=False
  )
  if returncode == 0 and fw_rules_str:
    fw_rules = fw_rules_str.splitlines()
    delete_tasks = []
    for rule in fw_rules:
      print(f"Deleting firewall rule: {rule}")
      delete_fw_cmd = [
          "gcloud",
          "compute",
          "firewall-rules",
          "delete",
          rule,
          f"--project={project_id}",
          "--quiet",
      ]
      delete_tasks.append(run_command_async(delete_fw_cmd, check=False))
    if delete_tasks:
      await asyncio.gather(*delete_tasks)

  # Now, delete the subnetwork and network.
  print(f"Deleting subnetwork '{subnet_name}'...")
  await run_command_async(
      [
          "gcloud",
          "compute",
          "networks",
          "subnets",
          "delete",
          subnet_name,
          f"--project={project_id}",
          f"--region={zone.rsplit('-', 1)[0]}",
          "--quiet",
      ],
      check=False,
  )

  print(f"Deleting network '{network_name}'...")
  await run_command_async(
      [
          "gcloud",
          "compute",
          "networks",
          "delete",
          network_name,
          f"--project={project_id}",
          "--quiet",
      ],
      check=False,
  )

  print("Cleanup complete.")
