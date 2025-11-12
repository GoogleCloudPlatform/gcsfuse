#!/usr/bin/env python3
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

"""Run GKE Orbax benchmark.

This script automates the process of running the Orbax benchmark on a GKE cluster.
It performs the following steps:
1.  Checks for prerequisite tools (gcloud, git, make, kubectl).
2.  Sets up a GKE cluster with a specific node pool if it doesn't exist.
3.  Builds a GCSFuse CSI driver image from a specified git branch.
4.  Deploys a Kubernetes pod that runs the benchmark workload.
5.  Parses the benchmark results (throughput) from the pod logs.
6.  Determines if the benchmark passed based on a performance threshold.
7.  Cleans up all created cloud resources (GKE cluster, network, etc.).
"""

import argparse
import asyncio
import os
import shlex
import subprocess
import sys
import tempfile
import re
from datetime import datetime
from string import Template

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
STAGING_VERSION = "orbax-benchmark"
DEFAULT_MTU = 8896

# Helper functions for running commands
async def run_command_async(command_list, check=True, cwd=None):
    """Runs a command asynchronously, preventing command injection.

    Args:
        command_list: A list of strings representing the command and its arguments.
        check: If True, raises CalledProcessError if the command returns a non-zero
          exit code.
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
        cwd=cwd)
    stdout, stderr = await process.communicate()
    stdout_decoded = stdout.decode().strip()
    stderr_decoded = stderr.decode().strip()

    if check and process.returncode != 0:
        raise subprocess.CalledProcessError(
            process.returncode, command_str, stdout_decoded, stderr_decoded)

    print(stdout_decoded)
    print(stderr_decoded, file=sys.stderr)
    return stdout_decoded, stderr_decoded, process.returncode

# Prerequisite Checks
async def check_prerequisites():
    """Checks for required command-line tools.

    Verifies that gcloud, git, make, and kubectl are installed. If kubectl is
    missing, it attempts to install it using 'gcloud components install'.
    Exits the script if any other required tool is not found.
    """
    await run_command_async(["sudo", "apt", "install", "-y", "apt-transport-https", "ca-certificates", "gnupg", "curl"])

    # Pipe curl output to gpg
    curl_process = await asyncio.create_subprocess_exec(
        "curl", "https://packages.cloud.google.com/apt/doc/apt-key.gpg",
        stdout=asyncio.subprocess.PIPE
    )
    gpg_process = await asyncio.create_subprocess_exec(
        "sudo", "gpg", "--dearmor", "-o", "/usr/share/keyrings/cloud.google.gpg",
        stdin=asyncio.subprocess.PIPE
    )
    await gpg_process.communicate(input=await curl_process.stdout.read())

    # Pipe echo output to tee
    echo_process = await asyncio.create_subprocess_exec(
        "echo", "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main",
        stdout=asyncio.subprocess.PIPE
    )
    tee_process = await asyncio.create_subprocess_exec(
        "sudo", "tee", "/etc/apt/sources.list.d/google-cloud-sdk.list",
        stdin=asyncio.subprocess.PIPE
    )
    await tee_process.communicate(input=await echo_process.stdout.read())

    await run_command_async(["sudo", "apt", "update", "-y"])
    print("Checking for required tools...")
    tools = {
        "gcloud": ["gcloud", "--version"],
        "git": ["git", "--version"],
        "make": ["make", "--version"],
        "kubectl": ["kubectl", "version", "--client=true"],
        "gke-gcloud-auth-plugin": ["gke-gcloud-auth-plugin", "--version"]
    }

    for tool, version_cmd in tools.items():
        try:
            await run_command_async(version_cmd)
        except (FileNotFoundError, subprocess.CalledProcessError):
            if tool == "gcloud":
                print("gcloud not found. Attempting to install...")
                try:
                    await run_command_async(["sudo", "apt", "install", "-y", "google-cloud-sdk"])
                    # Re-check after installation
                    await run_command_async(version_cmd)
                except (FileNotFoundError, subprocess.CalledProcessError) as install_e:
                    print(f"Error: Failed to install gcloud: {install_e}", file=sys.stderr)
                    sys.exit(1)
            if tool == "make":
                print("make not found. Attempting to install...")
                try:
                    await run_command_async(["sudo", "apt", "install", "-y", "make"])
                    await run_command_async(version_cmd)
                except (FileNotFoundError, subprocess.CalledProcessError) as install_e:
                    print(f"Error: Failed to install make: {install_e}", file=sys.stderr)
                    sys.exit(1)
            if tool == "kubectl":
                print("kubectl not found. Attempting to install...")
                try:
                    await run_command_async(["sudo", "snap", "install", "kubectl", "--classic"])                    
                except (FileNotFoundError, subprocess.CalledProcessError) as e:
                    print(f"Error: Failed to install kubectl: {e}", file=sys.stderr)
                    sys.exit(1)
            elif tool == "gke-gcloud-auth-plugin":
                print("gke-gcloud-auth-plugin not found. Attempting to install...")
                try:
                    await run_command_async(["sudo", "apt", "install", "-y", "google-cloud-sdk-gke-gcloud-auth-plugin"])
                except (FileNotFoundError, subprocess.CalledProcessError) as e:
                    print(f"Error: Failed to install gke-gcloud-auth-plugin: {e}", file=sys.stderr)
                    sys.exit(1)
            else:
                print(f"Error: Required tool '{tool}' is not installed. Please install it before running.", file=sys.stderr)
                sys.exit(1)
    print("All required tools are installed.")

# GKE Cluster and Node Pool Management
async def get_cluster_async(project_id, zone, cluster_name):
    """Checks if a GKE cluster exists.

    Args:
        project_id: The Google Cloud project ID.
        zone: The GCP zone where the cluster is located.
        cluster_name: The name of the GKE cluster.

    Returns:
        True if the cluster exists, False otherwise.
    """
    cmd = ["gcloud", "container", "clusters", "describe", cluster_name, f"--project={project_id}", f"--zone={zone}", "--format=value(name)"]
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
    cmd = ["gcloud", "container", "node-pools", "describe", node_pool_name, f"--project={project_id}", f"--zone={zone}", f"--cluster={cluster_name}", "--format=value(name)"]
    _, _, returncode = await run_command_async(cmd, check=False)
    return returncode == 0

async def is_node_pool_healthy_async(project_id, zone, cluster_name, node_pool_name):
    """Checks if a node pool's status is RUNNING.

    Args:
        project_id: The Google Cloud project ID.
        zone: The GCP zone where the cluster is located.
        cluster_name: The name of the GKE cluster.
        node_pool_name: The name of the node pool.

    Returns:
        True if the node pool status is 'RUNNING', False otherwise.
    """
    cmd = ["gcloud", "container", "node-pools", "describe", node_pool_name, f"--project={project_id}", f"--zone={zone}", f"--cluster={cluster_name}", "--format=value(status)"]
    status, _, returncode = await run_command_async(cmd, check=False)
    return returncode == 0 and status == "RUNNING"

async def create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type, reservation_name=None):
    """Creates a new node pool.

    Args:
        project_id: The Google Cloud project ID.
        zone: The GCP zone for the node pool.
        cluster_name: The name of the GKE cluster.
        node_pool_name: The name for the new node pool.
        machine_type: The machine type for the nodes in the pool.
        reservation_name: The specific reservation to use for the nodes.
    """
    cmd = ["gcloud", "container", "node-pools", "create", node_pool_name, f"--project={project_id}", f"--cluster={cluster_name}", f"--zone={zone}", f"--machine-type={machine_type}", "--num-nodes=1"]
    if reservation_name:
        cmd.extend([f"--reservation-affinity=specific", f"--reservation={reservation_name}"])
    await run_command_async(cmd)

async def delete_node_pool_async(project_id, zone, cluster_name, node_pool_name):
    """Deletes an existing node pool.

    Args:
        project_id: The Google Cloud project ID.
        zone: The GCP zone where the node pool is located.
        cluster_name: The name of the GKE cluster.
        node_pool_name: The name of the node pool to delete.
    """
    cmd = ["gcloud", "container", "node-pools", "delete", node_pool_name, f"--project={project_id}", f"--cluster={cluster_name}", f"--zone={zone}", "--quiet"]
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
    await run_command_async(["gcloud", "compute", "networks", "create", network_name, f"--project={project_id}", "--subnet-mode=custom", f"--mtu={mtu}"], check=False)
    await run_command_async(["gcloud", "compute", "networks", "subnets", "create", subnet_name, f"--project={project_id}", f"--network={network_name}", "--range=10.0.0.0/24", f"--region={region}"], check=False)

async def setup_gke_cluster(project_id, zone, cluster_name, network_name, subnet_name, region, machine_type, node_pool_name, reservation_name=None):
    """Sets up the GKE cluster and required node pool.

    This function ensures a GKE cluster and a specific node pool are ready for
    the benchmark. It will create the cluster, network, and node pool if they
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
    if await get_cluster_async(project_id, zone, cluster_name):
        if await get_node_pool_async(project_id, zone, cluster_name, node_pool_name):
            if not await is_node_pool_healthy_async(project_id, zone, cluster_name, node_pool_name):
                await delete_node_pool_async(project_id, zone, cluster_name, node_pool_name)
                await create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type, reservation_name)
        else:
            await create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type, reservation_name)
    else:
        await create_network(project_id, network_name, subnet_name, region, DEFAULT_MTU)
        cmd = ["gcloud", "container", "clusters", "create", cluster_name, f"--project={project_id}", f"--zone={zone}", f"--network={network_name}", f"--subnetwork={subnet_name}", f"--workload-pool={project_id}.svc.id.goog", "--addons=GcsFuseCsiDriver", "--num-nodes=1"]
        await run_command_async(cmd)
        await create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type, reservation_name)

    # Get credentials for the cluster to allow kubectl to connect.
    await run_command_async(["gcloud", "container", "clusters", "get-credentials", cluster_name, f"--project={project_id}", f"--zone={zone}"])
    print("GKE cluster setup complete.")

# GCSFuse Build and Deploy
async def build_gcsfuse_image(project_id, branch, temp_dir):
    """Clones GCSFuse and builds the CSI driver image.

    Args:
        branch: The git branch or tag of the GCSFuse repository to use.
        temp_dir: A temporary directory to clone the repository into.
    """
    gcsfuse_dir = os.path.join(temp_dir, "gcsfuse")
    await run_command_async(["git", "clone", "--depth=1", "-b", branch, "https://github.com/GoogleCloudPlatform/gcsfuse.git", gcsfuse_dir])
    build_cmd = ["make", "build-csi", f"PROJECT={project_id}", f"STAGINGVERSION={STAGING_VERSION}"]
    await run_command_async(build_cmd, cwd=gcsfuse_dir)

def parse_all_gbytes_per_sec(logs):
    """Parses logs to find and extract all gbytes_per_sec values.

    Args:
        logs: A string containing the log output from the benchmark pod.

    Returns:
        A list of float values representing the 'gbytes_per_sec' found in the logs.
    """
    values = []
    for line in logs.splitlines():
        match = re.search(r"gbytes_per_sec: ([\d.]+) Bytes/s", line)
        if match:
            gbytes_per_sec = float(match.group(1))
            print(f"Extracted gbytes_per_sec: {gbytes_per_sec}")
            values.append(gbytes_per_sec)
    if not values:
        print("gbytes_per_sec not found in logs.", file=sys.stderr)
    return values

# Workload Execution and Result Gathering
async def execute_workload_and_gather_results(project_id, zone, cluster_name, bucket_name, timestamp, iterations, staging_version, pod_timeout_seconds):
    """Executes the workload pod, gathers results, and cleans up workload resources.

    This function creates a Kubernetes ConfigMap and a Pod to run the benchmark.
    It waits for the pod to complete, collects its logs, parses the throughput
    results, and then deletes the created Kubernetes resources.

    Args:
        project_id: The Google Cloud project ID.
        zone: The GCP zone of the cluster.
        cluster_name: The name of the GKE cluster.
        bucket_name: The GCS bucket to use for the benchmark.
        timestamp: A unique timestamp string for manifest naming.
        iterations: The number of benchmark iterations to run inside the pod.
        staging_version: The version tag for the GCSFuse CSI driver image.
        pod_timeout_seconds: The timeout in seconds for the pod to complete.

    Returns:
        A list of throughput values (float) parsed from the pod logs.
    """
    await run_command_async(["kubectl", "create", "configmap", "orbax-benchmark", f"--from-file={os.path.join(SCRIPT_DIR, 'test_load.py')}"])

    template_path = os.path.join(SCRIPT_DIR, "pod.yaml.template")
    with open(template_path, "r") as f:
        pod_template = Template(f.read())

    manifest = pod_template.safe_substitute(project_id=project_id, bucket_name=bucket_name, iterations=iterations, staging_version=staging_version)
    manifest_filename = f"manifest-{timestamp}.yaml"
    pod_name = f"gcsfuse-test"

    try:
        with open(manifest_filename, "w") as f:
            f.write(manifest)
        await run_command_async(["kubectl", "apply", "-f", manifest_filename])

        start_time = datetime.now()
        pod_finished = False
        while (datetime.now() - start_time).total_seconds() < pod_timeout_seconds:
            status, stderr, _ = await run_command_async(["kubectl", "get", "pod", pod_name, "-o", "jsonpath='{.status.phase}'"], check=False)
            if "Succeeded" in status or "Failed" in status:
                pod_finished = True
                break
            await asyncio.sleep(10)

        if not pod_finished:
            raise TimeoutError(f"Pod did not complete within {pod_timeout_seconds / 60} minutes.")

        logs, _, _ = await run_command_async(["kubectl", "logs", pod_name], check=False)
        if logs:
            return parse_all_gbytes_per_sec(logs)
        return []
    finally:
        await run_command_async(["kubectl", "delete", "configmap", "orbax-benchmark"])
        await run_command_async(["kubectl", "delete", "-f", manifest_filename], check=False)
        if os.path.exists(manifest_filename):
            os.remove(manifest_filename)

# Cleanup
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
    await run_command_async(["gcloud", "container", "clusters", "delete", cluster_name, f"--project={project_id}", f"--zone={zone}", "--quiet"], check=False)

    # Find and delete firewall rules associated with the network.
    print(f"Finding and deleting firewall rules for network '{network_name}'...")
    list_fw_cmd = ["gcloud", "compute", "firewall-rules", "list", f"--project={project_id}", f"--filter=network~/{network_name}$", "--format=value(name)"]
    fw_rules_str, _, returncode = await run_command_async(list_fw_cmd, check=False)
    if returncode == 0 and fw_rules_str:
        fw_rules = fw_rules_str.splitlines()
        delete_tasks = []
        for rule in fw_rules:
            print(f"Deleting firewall rule: {rule}")
            delete_fw_cmd = ["gcloud", "compute", "firewall-rules", "delete", rule, f"--project={project_id}", "--quiet"]
            delete_tasks.append(run_command_async(delete_fw_cmd, check=False))
        if delete_tasks:
            await asyncio.gather(*delete_tasks)

    # Now, delete the subnetwork and network.
    print(f"Deleting subnetwork '{subnet_name}'...")
    await run_command_async(["gcloud", "compute", "networks", "subnets", "delete", subnet_name, f"--project={project_id}", f"--region={zone.rsplit('-', 1)[0]}", "--quiet"], check=False)

    print(f"Deleting network '{network_name}'...")
    await run_command_async(["gcloud", "compute", "networks", "delete", network_name, f"--project={project_id}", "--quiet"], check=False)

    print("Cleanup complete.")

# Main function
async def main():
    """Parses arguments, orchestrates the benchmark execution, and handles cleanup.

    This is the main entry point of the script.
    """
    parser = argparse.ArgumentParser(description="Run GKE Orbax benchmark.")
    parser.add_argument("--project_id", required=os.environ.get("PROJECT_ID") is None, default=os.environ.get("PROJECT_ID"), help="Google Cloud project ID. Can also be set with PROJECT_ID env var.")
    parser.add_argument("--bucket_name", required=os.environ.get("BUCKET_NAME") is None, default=os.environ.get("BUCKET_NAME"), help="GCS bucket name for the workload. Can also be set with BUCKET_NAME env var.")
    parser.add_argument("--zone", required=os.environ.get("ZONE") is None, default=os.environ.get("ZONE"), help="GCP zone. Can also be set with ZONE env var.")
    parser.add_argument("--cluster_name", default=os.environ.get("CLUSTER_NAME", "gke-orbax-benchmark-cluster"), help="GKE cluster name. Can also be set with CLUSTER_NAME env var.")
    parser.add_argument("--network_name", default=os.environ.get("NETWORK_NAME", "gke-orbax-benchmark-network"), help="VPC network name. Can also be set with NETWORK_NAME env var.")
    parser.add_argument("--subnet_name", default=os.environ.get("SUBNET_NAME", "gke-orbax-benchmark-subnet"), help="VPC subnet name. Can also be set with SUBNET_NAME env var.")
    parser.add_argument("--machine_type", default=os.environ.get("MACHINE_TYPE", "ct6e-standard-4t"), help="Machine type. Can also be set with MACHINE_TYPE env var.")
    parser.add_argument("--node_pool_name", default=os.environ.get("NODE_POOL_NAME", "ct6e-pool"), help="Node pool name. Can also be set with NODE_POOL_NAME env var.")
    parser.add_argument("--gcsfuse_branch", default=os.environ.get("GCSFUSE_BRANCH", "master"), help="GCSFuse branch or tag to build. Can also be set with GCSFUSE_BRANCH env var.")
    parser.add_argument("--reservation_name", default=os.environ.get("RESERVATION_NAME"), help="The specific reservation to use for the nodes. Can also be set with RESERVATION_NAME env var.")
    parser.add_argument("--no_cleanup", action="store_true", default=os.environ.get("NO_CLEANUP", "False").lower() in ("true", "1"), help="Don't clean up resources after. Can also be set with NO_CLEANUP=true env var.")
    parser.add_argument("--iterations", type=int, default=int(os.environ.get("ITERATIONS", 20)), help="Number of iterations for the benchmark. Can also be set with ITERATIONS env var.")
    parser.add_argument("--performance_threshold_gbps", type=float, default=float(os.environ.get("PERFORMANCE_THRESHOLD_GBPS", 13.0)), help="Minimum throughput in GB/s for a successful iteration. Can also be set with PERFORMANCE_THRESHOLD_GBPS env var.")
    parser.add_argument("--pod_timeout_seconds", type=int, default=int(os.environ.get("POD_TIMEOUT_SECONDS", 1800)), help="Timeout in seconds for the benchmark pod to complete. Can also be set with POD_TIMEOUT_SECONDS env var.")
    parser.add_argument("--skip_csi_driver_build", action="store_true", default=os.environ.get("SKIP_CSI_DRIVER_BUILD", "False").lower() in ("true", "1"), help="Skip building the CSI driver. Can also be set with SKIP_CSI_DRIVER_BUILD=true env var.")
    args = parser.parse_args()

    # Append zone to default network and subnet names to avoid collisions
    if args.network_name == "gke-orbax-benchmark-network":
        args.network_name = f"{args.network_name}-{args.zone}"
    if args.subnet_name == "gke-orbax-benchmark-subnet":
        args.subnet_name = f"{args.subnet_name}-{args.zone}"

    await check_prerequisites()

    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    with tempfile.TemporaryDirectory() as temp_dir:
        try:
            if args.skip_csi_driver_build:
                await setup_gke_cluster(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name, args.zone.rsplit('-', 1)[0], args.machine_type, args.node_pool_name, args.reservation_name)
            else:
                setup_task = asyncio.create_task(setup_gke_cluster(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name, args.zone.rsplit('-', 1)[0], args.machine_type, args.node_pool_name, args.reservation_name))
                build_task = asyncio.create_task(build_gcsfuse_image(args.project_id,args.gcsfuse_branch, temp_dir))
                await asyncio.gather(setup_task, build_task)

            throughputs = await execute_workload_and_gather_results(args.project_id, args.zone, args.cluster_name, args.bucket_name, timestamp, args.iterations, STAGING_VERSION, args.pod_timeout_seconds)

            if not throughputs:
                print("No throughput data was collected.", file=sys.stderr)
                if not args.no_cleanup:
                    await cleanup(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name)
                sys.exit(-1)

            successful_iterations = sum(1 for t in throughputs if t >= args.performance_threshold_gbps)
            if successful_iterations < (len(throughputs) * 5)/8: # At least 5/8th of the iterations must meet the threshold.
                print(f"Benchmark failed: Only {successful_iterations}/{len(throughputs)} iterations were >= {args.performance_threshold_gbps} gbytes/sec.", file=sys.stderr)
                if not args.no_cleanup:
                    await cleanup(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name)
                sys.exit(-1)

            print(f"Benchmark successful: {successful_iterations}/{len(throughputs)} iterations met the performance threshold ({args.performance_threshold_gbps} GB/s).")
        finally:
            if not args.no_cleanup:
                await cleanup(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name)

if __name__ == "__main__":
    asyncio.run(main())
