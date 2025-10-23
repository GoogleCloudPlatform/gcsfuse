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

import argparse
import asyncio
import os
import shlex
import shutil
import subprocess
import sys
import tempfile
import re
import yaml
from datetime import datetime
from string import Template

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
STAGING_VERSION = "orbax-benchmark"

# Helper functions for running commands
async def run_command_async(command_list, check=True, cwd=None):
    """Runs a command asynchronously, preventing command injection."""
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
    """Checks for and installs required command-line tools."""
    print("Checking for required tools...")
    tools = {
        "gcloud": ["gcloud", "--version"],
        "git": ["git", "--version"],
        "make": ["make", "--version"],
        "kubectl": ["kubectl", "version", "--client=true"]
    }

    for tool, version_cmd in tools.items():
        try:
            await run_command_async(version_cmd)
        except (FileNotFoundError, subprocess.CalledProcessError):
            if tool == "kubectl":
                print("kubectl not found. Attempting to install via gcloud components...")
                try:
                    await run_command_async(["gcloud", "components", "install", "kubectl"])
                except (FileNotFoundError, subprocess.CalledProcessError) as e:
                    print(f"Error: Failed to install kubectl: {e}", file=sys.stderr)
                    sys.exit(1)
            else:
                print(f"Error: Required tool '{tool}' is not installed. Please install it before running.", file=sys.stderr)
                sys.exit(1)
    print("All required tools are installed.")

# GKE Cluster and Node Pool Management
async def get_cluster_async(project_id, zone, cluster_name):
    """Checks if a GKE cluster exists."""
    cmd = ["gcloud", "container", "clusters", "describe", cluster_name, f"--project={project_id}", f"--zone={zone}", "--format=value(name)"]
    _, _, returncode = await run_command_async(cmd, check=False)
    return returncode == 0

async def get_node_pool_async(project_id, zone, cluster_name, node_pool_name):
    """Checks if a node pool exists in a GKE cluster."""
    cmd = ["gcloud", "container", "node-pools", "describe", node_pool_name, f"--project={project_id}", f"--zone={zone}", f"--cluster={cluster_name}", "--format=value(name)"]
    _, _, returncode = await run_command_async(cmd, check=False)
    return returncode == 0

async def is_node_pool_healthy_async(project_id, zone, cluster_name, node_pool_name):
    """Checks if a node pool's status is RUNNING."""
    cmd = ["gcloud", "container", "node-pools", "describe", node_pool_name, f"--project={project_id}", f"--zone={zone}", f"--cluster={cluster_name}", "--format=value(status)"]
    status, _, returncode = await run_command_async(cmd, check=False)
    return returncode == 0 and status == "RUNNING"

async def create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type):
    """Creates a new node pool."""
    cmd = ["gcloud", "container", "node-pools", "create", node_pool_name, f"--project={project_id}", f"--cluster={cluster_name}", f"--zone={zone}", f"--machine-type={machine_type}", "--num-nodes=1"]
    await run_command_async(cmd)

async def delete_node_pool_async(project_id, zone, cluster_name, node_pool_name):
    """Deletes an existing node pool."""
    cmd = ["gcloud", "container", "node-pools", "delete", node_pool_name, f"--project={project_id}", f"--cluster={cluster_name}", f"--zone={zone}", "--quiet"]
    await run_command_async(cmd, check=False)

async def create_network(project_id, network_name, subnet_name, region, mtu):
    """Creates a new network and subnet if they don't exist."""
    await run_command_async(["gcloud", "compute", "networks", "create", network_name, f"--project={project_id}", "--subnet-mode=custom", f"--mtu={mtu}"], check=False)
    await run_command_async(["gcloud", "compute", "networks", "subnets", "create", subnet_name, f"--project={project_id}", f"--network={network_name}", "--range=10.0.0.0/24", f"--region={region}"], check=False)

async def setup_gke_cluster(project_id, zone, cluster_name, network_name, subnet_name, region, machine_type, node_pool_name):
    """Sets up the GKE cluster and required node pool."""
    if await get_cluster_async(project_id, zone, cluster_name):
        if await get_node_pool_async(project_id, zone, cluster_name, node_pool_name):
            if not await is_node_pool_healthy_async(project_id, zone, cluster_name, node_pool_name):
                await delete_node_pool_async(project_id, zone, cluster_name, node_pool_name)
                await create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type)
        else:
            await create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type)
    else:
        await create_network(project_id, network_name, subnet_name, region, 8896)
        cmd = ["gcloud", "container", "clusters", "create", cluster_name, f"--project={project_id}", f"--zone={zone}", f"--network={network_name}", f"--subnetwork={subnet_name}", f"--workload-pool={project_id}.svc.id.goog", "--addons=GcsFuseCsiDriver", "--num-nodes=1"]
        await run_command_async(cmd)
        await create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type)
    print("GKE cluster setup complete.")

# GCSFuse Build and Deploy
async def build_gcsfuse_image(project_id, branch, temp_dir):
    """Clones GCSFuse and builds the CSI driver image."""
    gcsfuse_dir = os.path.join(temp_dir, "gcsfuse")
    await run_command_async(["git", "clone", "--depth=1", "-b", branch, "https://github.com/GoogleCloudPlatform/gcsfuse.git", gcsfuse_dir])
    build_cmd = ["make", "build-csi", f"STAGINGVERSION={STAGING_VERSION}"]
    await run_command_async(build_cmd, cwd=gcsfuse_dir)
    shutil.rmtree(gcsfuse_dir)

def parse_all_gbytes_per_sec(logs):
    """Parses logs to find and extract all gbytes_per_sec values."""
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
async def execute_workload_and_gather_results(project_id, zone, cluster_name, bucket_name, timestamp, iterations, staging_version):
    """Executes the workload pod, gathers results, and cleans up workload resources."""
    await run_command_async(["kubectl", "create", "configmap", "orbax-benchmark", "--from-file=test_load.py"])

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
        while (datetime.now() - start_time).total_seconds() < 30 * 60:
            status, stderr, _ = await run_command_async(["kubectl", "get", "pod", pod_name, "-o", "jsonpath='{.status.phase}'"], check=False)
            if "Succeeded" in status or "Failed" in status:
                pod_finished = True
                break
            await asyncio.sleep(10)

        if not pod_finished:
            raise TimeoutError("Pod did not complete within 30 minutes.")

        logs, _, _ = await run_command_async(["kubectl", "logs", pod_name], check=False)
        if logs:
            return parse_all_gbytes_per_sec(logs)
        return []
    finally:
        await run_command_async(["kubectl", "delete", "configmap", "orbax-benchmark"])
        await run_command_async(["kubectl", "delete", "-f", manifest_filename], check=False)
        if os.path.exists(manifest_filename):
            os.remove(manifest_filename)
    return []

# Cleanup
async def cleanup(project_id, zone, cluster_name, network_name, subnet_name):
    """Cleans up the created GKE, network, and firewall resources."""
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
    """Main function."""
    parser = argparse.ArgumentParser(description="Run GKE Orbax benchmark.")
    parser.add_argument("--project_id", required=True, help="Google Cloud project ID.")
    parser.add_argument("--bucket_name", required=True, help="GCS bucket name for the workload.")
    parser.add_argument("--zone", default="us-east5-b", help="GCP zone.")
    parser.add_argument("--cluster_name", default="gke-orbax-benchmark-cluster", help="GKE cluster name.")
    parser.add_argument("--network_name", default="gke-orbax-benchmark-network", help="VPC network name.")
    parser.add_argument("--subnet_name", default="gke-orbax-benchmark-subnet", help="VPC subnet name.")
    parser.add_argument("--machine_type", default="ct6e-standard-4t", help="Machine type.")
    parser.add_argument("--node_pool_name", default="ct6e-pool", help="Node pool name.")
    parser.add_argument("--gcsfuse_branch", default="master", help="GCSFuse branch or tag to build.")
    parser.add_argument("--no_cleanup", action="store_true", help="Don't clean up resources after.")
    parser.add_argument("--iterations", type=int, default=10, help="Number of iterations for the benchmark.")
    parser.add_argument("--performance_threshold_gbps", type=float, default=13.0, help="Minimum throughput in GB/s for a successful iteration.")
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
            setup_task = asyncio.create_task(setup_gke_cluster(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name, args.zone.rsplit('-', 1)[0], args.machine_type, args.node_pool_name))
            build_task = asyncio.create_task(build_gcsfuse_image(args.project_id, args.gcsfuse_branch, temp_dir))
            await asyncio.gather(setup_task, build_task)

            throughputs = await execute_workload_and_gather_results(args.project_id, args.zone, args.cluster_name, args.bucket_name, timestamp, args.iterations, STAGING_VERSION)

            if not throughputs:
                print("No throughput data was collected.", file=sys.stderr)
                if not args.no_cleanup:
                    await cleanup(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name)
                sys.exit(-1)

            successful_iterations = sum(1 for t in throughputs if t >= args.performance_threshold_gbps)
            if successful_iterations <= len(throughputs) / 2: # At least half iterations must meet the threshold.
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
