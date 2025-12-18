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
import re
import subprocess
import sys
import tempfile
from datetime import datetime
from string import Template

# Add the parent directory to sys.path to allow imports from common
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
sys.path.append(os.path.abspath(os.path.join(SCRIPT_DIR, '..')))
from common import utils

# The prefix prow-gob-internal-boskos- is needed to allow passing machine-type from gke csi driver to gcsfuse,
# bypassing the check at
# https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver/blob/15afd00dcc2cfe0f9753ddc53c81631ff037c3f2/pkg/csi_driver/utils.go#L532.
STAGING_VERSION = "prow-gob-internal-boskos-orbax-benchmark"
DEFAULT_MTU = 8896

# Prerequisite Checks
async def check_prerequisites():
    """Checks for required command-line tools.

    Verifies that gcloud, git, make, and kubectl are installed. If kubectl is
    missing, it attempts to install it using 'gcloud components install'.
    Exits the script if any other required tool is not found.
    """
    await utils.run_command_async(["sudo", "apt", "install", "-y", "apt-transport-https", "ca-certificates", "gnupg", "curl"])

    # Pipe curl output to gpg
    curl_process = await asyncio.create_subprocess_exec(
        "curl", "https://packages.cloud.google.com/apt/doc/apt-key.gpg",
        stdout=asyncio.subprocess.PIPE
    )
    gpg_process = await asyncio.create_subprocess_exec(
        "sudo", "gpg", "--yes", "--dearmor", "-o", "/usr/share/keyrings/cloud.google.gpg",
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

    await utils.run_command_async(["sudo", "apt", "update", "-y"])
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
            await utils.run_command_async(version_cmd)
        except (FileNotFoundError, subprocess.CalledProcessError):
            if tool == "gcloud":
                print("gcloud not found. Attempting to install...")
                try:
                    await utils.run_command_async(["sudo", "apt", "install", "-y", "google-cloud-sdk"])
                    # Re-check after installation
                    await utils.run_command_async(version_cmd)
                except (FileNotFoundError, subprocess.CalledProcessError) as install_e:
                    print(f"Error: Failed to install gcloud: {install_e}", file=sys.stderr)
                    sys.exit(1)
            if tool == "make":
                print("make not found. Attempting to install...")
                try:
                    await utils.run_command_async(["sudo", "apt", "install", "-y", "make"])
                    await utils.run_command_async(version_cmd)
                except (FileNotFoundError, subprocess.CalledProcessError) as install_e:
                    print(f"Error: Failed to install make: {install_e}", file=sys.stderr)
                    sys.exit(1)
            if tool == "kubectl":
                print("kubectl not found. Attempting to install...")
                try:
                    await utils.run_command_async(["sudo", "snap", "install", "kubectl", "--classic"])
                except (FileNotFoundError, subprocess.CalledProcessError) as e:
                    print(f"Error: Failed to install kubectl: {e}", file=sys.stderr)
                    sys.exit(1)
            elif tool == "gke-gcloud-auth-plugin":
                print("gke-gcloud-auth-plugin not found. Attempting to install...")
                try:
                    await utils.run_command_async(["sudo", "apt", "install", "-y", "google-cloud-sdk-gke-gcloud-auth-plugin"])
                except (FileNotFoundError, subprocess.CalledProcessError) as e:
                    print(f"Error: Failed to install gke-gcloud-auth-plugin: {e}", file=sys.stderr)
                    sys.exit(1)
            else:
                print(f"Error: Required tool '{tool}' is not installed. Please install it before running.", file=sys.stderr)
                sys.exit(1)
    print("All required tools are installed.")

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
    if await utils.get_cluster_async(project_id, zone, cluster_name):
        if await utils.get_node_pool_async(project_id, zone, cluster_name, node_pool_name):
            if not await utils.is_node_pool_healthy_async(project_id, zone, cluster_name, node_pool_name):
                await utils.delete_node_pool_async(project_id, zone, cluster_name, node_pool_name)
                await utils.create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type, reservation_name)
        else:
            await utils.create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type, reservation_name)
    else:
        await utils.create_network(project_id, network_name, subnet_name, region, DEFAULT_MTU)
        cmd = ["gcloud", "container", "clusters", "create", cluster_name, "--release-channel=rapid", f"--project={project_id}", f"--zone={zone}", f"--network={network_name}", f"--subnetwork={subnet_name}", f"--workload-pool={project_id}.svc.id.goog", "--addons=GcsFuseCsiDriver", "--num-nodes=1"]
        await utils.run_command_async(cmd)
        await utils.create_node_pool_async(project_id, zone, cluster_name, node_pool_name, machine_type, reservation_name)

    # Get credentials for the cluster to allow kubectl to connect.
    await utils.run_command_async(["gcloud", "container", "clusters", "get-credentials", cluster_name, f"--project={project_id}", f"--zone={zone}"])
    print("GKE cluster setup complete.")

# GCSFuse Build and Deploy
async def build_gcsfuse_image(project_id, branch, temp_dir):
    """Clones GCSFuse and builds the CSI driver image.

    Args:
        branch: The git branch or tag of the GCSFuse repository to use.
        temp_dir: A temporary directory to clone the repository into.
    """
    gcsfuse_dir = os.path.join(temp_dir, "gcsfuse")
    await utils.run_command_async(["git", "clone", "--depth=1", "-b", branch, "https://github.com/GoogleCloudPlatform/gcsfuse.git", gcsfuse_dir])
    build_cmd = ["make", "build-csi", f"PROJECT={project_id}", f"STAGINGVERSION={STAGING_VERSION}"]
    await utils.run_command_async(build_cmd, cwd=gcsfuse_dir)

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
    await utils.run_command_async(["kubectl", "create", "configmap", "orbax-benchmark", f"--from-file={os.path.join(SCRIPT_DIR, 'test_load.py')}"])

    template_path = os.path.join(SCRIPT_DIR, "pod.yaml.template")
    with open(template_path, "r") as f:
        pod_template = Template(f.read())

    manifest = pod_template.safe_substitute(project_id=project_id, bucket_name=bucket_name, iterations=iterations, staging_version=staging_version)
    manifest_filename = f"manifest-{timestamp}.yaml"
    pod_name = f"gcsfuse-test"

    try:
        with open(manifest_filename, "w") as f:
            f.write(manifest)
        await utils.run_command_async(["kubectl", "apply", "-f", manifest_filename])

        start_time = datetime.now()
        pod_finished = False
        while (datetime.now() - start_time).total_seconds() < pod_timeout_seconds:
            status, stderr, _ = await utils.run_command_async(["kubectl", "get", "pod", pod_name, "-o", "jsonpath='{.status.phase}'"], check=False)
            if "Succeeded" in status or "Failed" in status:
                pod_finished = True
                break
            await asyncio.sleep(10)

        if not pod_finished:
            raise TimeoutError(f"Pod did not complete within {pod_timeout_seconds / 60} minutes.")

        logs, _, _ = await utils.run_command_async(["kubectl", "logs", pod_name], check=False)
        if logs:
            return parse_all_gbytes_per_sec(logs)
        return []
    finally:
        await utils.run_command_async(["kubectl", "delete", "configmap", "orbax-benchmark"])
        await utils.run_command_async(["kubectl", "delete", "-f", manifest_filename], check=False)
        if os.path.exists(manifest_filename):
            os.remove(manifest_filename)

# Cleanup logic is shared in utils.cleanup, but Orbax calls it with specific args in main

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
                    await utils.cleanup(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name)
                sys.exit(-1)

            successful_iterations = sum(1 for t in throughputs if t >= args.performance_threshold_gbps)
            if successful_iterations < (len(throughputs) * 5)/8: # At least 5/8th of the iterations must meet the threshold.
                print(f"Benchmark failed: Only {successful_iterations}/{len(throughputs)} iterations were >= {args.performance_threshold_gbps} gbytes/sec.", file=sys.stderr)
                if not args.no_cleanup:
                    await utils.cleanup(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name)
                sys.exit(-1)

            print(f"Benchmark successful: {successful_iterations}/{len(throughputs)} iterations met the performance threshold ({args.performance_threshold_gbps} GB/s).")
        finally:
            if not args.no_cleanup:
                await utils.cleanup(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name)

if __name__ == "__main__":
    asyncio.run(main())