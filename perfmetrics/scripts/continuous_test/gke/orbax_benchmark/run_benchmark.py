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
sys.path.append(os.path.dirname(SCRIPT_DIR))
from common import utils

# The prefix prow-gob-internal-boskos- is needed to allow passing machine-type from gke csi driver to gcsfuse,
# bypassing the check at
# https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver/blob/15afd00dcc2cfe0f9753ddc53c81631ff037c3f2/pkg/csi_driver/utils.go#L532.
STAGING_VERSION = "prow-gob-internal-boskos-orbax-benchmark"


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


# Main function
async def main():
    """Parses arguments, orchestrates the benchmark execution, and handles cleanup.

    This is the main entry point of the script.
    """
    parser = argparse.ArgumentParser(description="Run GKE Orbax benchmark.")
    parser.add_argument("--project_id", default=os.environ.get("PROJECT_ID", utils.DEFAULT_PROJECT_ID), help="Google Cloud project ID. Can also be set with PROJECT_ID env var.")
    parser.add_argument("--bucket_name", required=os.environ.get("BUCKET_NAME") is None, default=os.environ.get("BUCKET_NAME"), help="GCS bucket name for the workload. Can also be set with BUCKET_NAME env var.")
    parser.add_argument("--zone", default=os.environ.get("ZONE", utils.DEFAULT_ZONE), help="GCP zone. Can also be set with ZONE env var.")
    parser.add_argument("--cluster_name", default=os.environ.get("CLUSTER_NAME", "gke-orbax-benchmark-cluster"), help="GKE cluster name. Can also be set with CLUSTER_NAME env var.")
    parser.add_argument("--network_name", default=os.environ.get("NETWORK_NAME", "gke-orbax-benchmark-network"), help="VPC network name. Can also be set with NETWORK_NAME env var.")
    parser.add_argument("--subnet_name", default=os.environ.get("SUBNET_NAME", "gke-orbax-benchmark-subnet"), help="VPC subnet name. Can also be set with SUBNET_NAME env var.")
    parser.add_argument("--machine_type", default=os.environ.get("MACHINE_TYPE", "ct6e-standard-4t"), help="Machine type. Can also be set with MACHINE_TYPE env var.")
    parser.add_argument("--node_pool_name", default=os.environ.get("NODE_POOL_NAME", "ct6e-pool"), help="Node pool name. Can also be set with NODE_POOL_NAME env var.")
    parser.add_argument("--gcsfuse_branch", default=os.environ.get("GCSFUSE_BRANCH", "master"), help="GCSFuse branch or tag to build. Can also be set with GCSFUSE_BRANCH env var.")
    parser.add_argument("--reservation_name", default=os.environ.get("RESERVATION_NAME", utils.DEFAULT_RESERVATION_NAME), help="The specific reservation to use for the nodes. Can also be set with RESERVATION_NAME env var.")
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

    await utils.check_prerequisites()

    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    with tempfile.TemporaryDirectory() as temp_dir:
        try:
            if args.skip_csi_driver_build:
                await utils.setup_gke_cluster(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name, args.zone.rsplit('-', 1)[0], args.machine_type, args.node_pool_name, args.reservation_name)
            else:
                setup_task = asyncio.create_task(utils.setup_gke_cluster(args.project_id, args.zone, args.cluster_name, args.network_name, args.subnet_name, args.zone.rsplit('-', 1)[0], args.machine_type, args.node_pool_name, args.reservation_name))
                build_task = asyncio.create_task(utils.build_gcsfuse_image(args.project_id,args.gcsfuse_branch, temp_dir, STAGING_VERSION))
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

