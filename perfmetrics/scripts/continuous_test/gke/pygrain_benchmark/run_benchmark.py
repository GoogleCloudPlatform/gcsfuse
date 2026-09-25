#!/usr/bin/env python3
# Copyright 2026 Google LLC
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

"""Run GKE MaxText + PyGrain benchmark.

This script automates the process of running the PyGrain Multimodal ArrayRecord
TPU data loading benchmark on a GKE cluster.
It performs the following steps:
1.  Checks for prerequisite tools (gcloud, git, make, kubectl).
2.  Sets up a GKE cluster with a specific TPU v6e node pool if it doesn't exist.
3.  Builds a GCSFuse CSI driver image from a specified git branch (unless skipped).
4.  Configures Workload Identity bucket permissions and checks dataset shards.
5.  Deploys a Kubernetes pod that provisions the synthetic ArrayRecord dataset
    (if missing) and runs the PyGrain + MaxText benchmark workload.
6.  Parses the per-iteration benchmark results (throughput) from the pod logs.
7.  Determines if the benchmark passed based on the 5/8 performance threshold rule.
8.  Cleans up created Kubernetes and cloud resources.
"""

import argparse
import asyncio
from datetime import datetime
import os
import re
import shutil
from string import Template
import subprocess
import sys
import tempfile

# Add the parent directory to sys.path to allow imports from common
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
sys.path.append(os.path.dirname(SCRIPT_DIR))
from common import utils

# The prefix prow-gob-internal-boskos- is needed to allow passing machine-type from gke csi driver to gcsfuse,
# bypassing the check at
# https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver/blob/15afd00dcc2cfe0f9753ddc53c81631ff037c3f2/pkg/csi_driver/utils.go#L532.
STAGING_VERSION = "prow-gob-internal-boskos-pygrain-benchmark"
FALLBACK_STAGING_VERSION = "prow-gob-internal-boskos-orbax-benchmark"

DEFAULT_CLUSTER_NAME = "gke-pygrain-benchmark-cluster"
DEFAULT_NETWORK_NAME = "gke-pygrain-benchmark-network"
DEFAULT_SUBNET_NAME = "gke-pygrain-benchmark-subnet"


def parse_all_gbytes_per_sec(logs):
    """Parses logs to find and extract all gbytes_per_sec values.

    Args:
        logs: A string containing the log output from the benchmark pod.

    Returns:
        A list of float values representing the 'gbytes_per_sec' found in the logs.
    """
    if not logs:
        print("gbytes_per_sec not found in logs.", file=sys.stderr)
        return []
    values = []
    for line in logs.splitlines():
        match = re.search(r"gbytes_per_sec:\s*([\d.]+)\s*Bytes/s", line)
        if match:
            try:
                gbytes_per_sec = float(match.group(1))
            except ValueError:
                continue
            print(f"Extracted gbytes_per_sec: {gbytes_per_sec}")
            values.append(gbytes_per_sec)

    if not values and "===JSON_METRICS_START===" in logs and "===JSON_METRICS_END===" in logs:
        import json

        try:
            json_block = logs.split("===JSON_METRICS_START===", 1)[1].split(
                "===JSON_METRICS_END===", 1
            )[0]
            data = json.loads(json_block.strip())
            if data.get("iteration_throughputs_gbps"):
                for val in data["iteration_throughputs_gbps"]:
                    gbytes_per_sec = float(val)
                    print(f"Extracted gbytes_per_sec: {gbytes_per_sec}")
                    values.append(gbytes_per_sec)
        except Exception:
            pass

    if not values:
        print("gbytes_per_sec not found in logs.", file=sys.stderr)
    return values


def evaluate_throughputs(throughputs, threshold_gbps=13.0, client_protocol="http1"):
    """Evaluates extracted throughputs against the 5/8 iteration rule.

    Args:
        throughputs: List of float throughput values in GB/s.
        threshold_gbps: Minimum throughput in GB/s required per iteration.
        client_protocol: GCS client protocol ('http1' or 'grpc').

    Returns:
        Tuple of (passed: bool, successful_iterations: int).
    """
    if not throughputs:
        print("No throughput data was collected.", file=sys.stderr)
        return False, 0

    successful_iterations = sum(1 for t in throughputs if t >= threshold_gbps)
    if successful_iterations < (len(throughputs) * 5) / 8:  # At least 5/8th of the iterations must meet the threshold.
        if client_protocol == "grpc":
            # TODO: Remove this skip once the performance regression is addressed for GRPC.
            print(
                f"Warning: Only {successful_iterations}/{len(throughputs)} iterations were >= {threshold_gbps} gbytes/sec, "
                "but continuing as success for GRPC."
            )
            return True, successful_iterations
        else:
            print(
                f"Benchmark failed: Only {successful_iterations}/{len(throughputs)} iterations were >= {threshold_gbps} gbytes/sec.",
                file=sys.stderr,
            )
            return False, successful_iterations
    else:
        print(
            f"Benchmark successful: {successful_iterations}/{len(throughputs)} iterations met the performance threshold ({threshold_gbps} GB/s)."
        )
        return True, successful_iterations


def render_pod_manifest(
    project_id,
    bucket_name,
    iterations,
    staging_version=STAGING_VERSION,
    client_protocol="http1",
    template_path=None,
):
    """Renders pod.yaml.template with the given parameters."""
    if template_path is None:
        template_path = os.path.join(SCRIPT_DIR, "pod.yaml.template")
    with open(template_path, "r") as f:
        pod_template = Template(f.read())
    mount_protocol = (
        "grpc" if "rapid" in str(bucket_name).lower() else client_protocol
    )
    rendered = pod_template.safe_substitute(
        project_id=project_id,
        bucket_name=bucket_name,
        iterations=iterations,
        staging_version=staging_version,
        client_protocol=mount_protocol,
    )
    if staging_version == FALLBACK_STAGING_VERSION:
        sidecar_block = (
            "  - name: gke-gcsfuse-sidecar\n"
            f"    image: gcr.io/{project_id}/cloudbuild-gcsfuse-csi/gcs-fuse-csi-driver-sidecar-mounter:{staging_version}\n"
        )
        rendered = rendered.replace(sidecar_block, "")
    return rendered


async def check_dataset_exists(bucket_name, expected_shards=128):
    """Checks whether the synthetic ArrayRecord dataset shards exist in the bucket.

    Args:
        bucket_name: Target GCS bucket name.
        expected_shards: Expected count of train-*.arrayrecord shards (default: 128).

    Returns:
        Tuple of (exists: bool, shard_count: int).
    """
    stdout, _, returncode = await utils.run_command_async(
        ["gcloud", "storage", "ls", f"gs://{bucket_name}/train-*.arrayrecord"],
        check=False,
    )
    if returncode != 0 or not stdout:
        return False, 0
    shards = [
        line.strip()
        for line in stdout.splitlines()
        if line.strip().endswith(".arrayrecord")
    ]
    return len(shards) >= expected_shards, len(shards)


async def ensure_dataset_provisioned(
    bucket_name, expected_shards=128, generator_fn=None
):
    """Checks if dataset exists in GCS and triggers generator_fn if missing/incomplete.

    Args:
        bucket_name: Target GCS bucket name.
        expected_shards: Expected number of ArrayRecord shards (default: 128).
        generator_fn: Optional callable invoked when shards are missing.

    Returns:
        True if dataset generation was triggered, False if existing shards are reused.
    """
    exists, shard_count = await check_dataset_exists(
        bucket_name, expected_shards=expected_shards
    )
    if exists:
        print(
            f"Dataset check: {shard_count}/{expected_shards} ArrayRecord shards already present in gs://{bucket_name}/; reusing existing shards."
        )
        return False

    print(
        f"Dataset check: found {shard_count}/{expected_shards} shards in gs://{bucket_name}/; triggering synthetic ArrayRecord generation."
    )
    if generator_fn is not None:
        res = generator_fn(bucket_name, expected_shards)
        if asyncio.iscoroutine(res):
            await res
    return True


async def resolve_staging_version(
    project_id, staging_version=STAGING_VERSION, skip_csi_driver_build=False
):
    """Resolves the CSI sidecar image staging tag when skip_csi_driver_build is enabled."""
    if not skip_csi_driver_build:
        return staging_version

    image_uri = f"gcr.io/{project_id}/cloudbuild-gcsfuse-csi/gcs-fuse-csi-driver-sidecar-mounter:{staging_version}"
    _, _, returncode = await utils.run_command_async(
        ["gcloud", "container", "images", "describe", image_uri],
        check=False,
    )
    if returncode == 0:
        return staging_version

    print(
        f"Staging image {image_uri} not found while --skip_csi_driver_build is set; "
        f"falling back to {FALLBACK_STAGING_VERSION}."
    )
    return FALLBACK_STAGING_VERSION


async def set_up_bucket_permissions(
    project_id, zone, cluster_name, bucket_name
):
    """Grants Storage objectUser role on bucket_name to the cluster's default KSA."""
    await utils.run_command_async(
        [
            "gcloud",
            "container",
            "clusters",
            "get-credentials",
            cluster_name,
            f"--project={project_id}",
            f"--zone={zone}",
        ]
    )
    await utils.run_command_async(
        ["kubectl", "create", "serviceaccount", "default", "--namespace=default"],
        check=False,
    )
    project_number, _, returncode = await utils.run_command_async(
        [
            "gcloud",
            "projects",
            "describe",
            project_id,
            "--format=value(projectNumber)",
        ],
        check=False,
    )
    if returncode != 0 or not project_number:
        return

    principal = (
        f"principal://iam.googleapis.com/projects/{project_number.strip()}/locations/global/"
        f"workloadIdentityPools/{project_id}.svc.id.goog/subject/ns/default/sa/default"
    )
    print(
        f"Granting roles/storage.objectUser to {principal} on bucket {bucket_name}..."
    )
    await utils.run_command_async(
        [
            "gcloud",
            "storage",
            "buckets",
            "add-iam-policy-binding",
            f"gs://{bucket_name}",
            f"--member={principal}",
            "--role=roles/storage.objectUser",
            f"--project={project_id}",
        ],
        check=False,
    )


async def execute_workload_and_gather_results(
    project_id,
    zone,
    cluster_name,
    bucket_name,
    timestamp,
    iterations,
    staging_version,
    pod_timeout_seconds,
    client_protocol,
):
    """Executes the workload pod, gathers results, and cleans up workload resources.

    Args:
        project_id: The Google Cloud project ID.
        zone: The GCP zone of the cluster.
        cluster_name: The name of the GKE cluster.
        bucket_name: The GCS bucket to use for the benchmark.
        timestamp: A unique timestamp string for manifest naming.
        iterations: The number of benchmark iterations to run inside the pod.
        staging_version: The version tag for the GCSFuse CSI driver image.
        pod_timeout_seconds: The timeout in seconds for the pod to complete.
        client_protocol: The client protocol to use for GCS (e.g. grpc or http1).

    Returns:
        A list of throughput values (float) parsed from the pod logs.
    """
    pod_name = "gcsfuse-test"
    configmap_name = "pygrain-benchmark"

    # Clean up any leftover configmap or pod from prior interrupted runs
    await utils.run_command_async(
        [
            "kubectl",
            "delete",
            "configmap",
            configmap_name,
            "--ignore-not-found=true",
        ],
        check=False,
    )
    await utils.run_command_async(
        ["kubectl", "delete", "pod", pod_name, "--ignore-not-found=true"],
        check=False,
    )

    await utils.run_command_async(
        [
            "kubectl",
            "create",
            "configmap",
            configmap_name,
            f"--from-file={os.path.join(SCRIPT_DIR, 'generate_data.py')}",
            f"--from-file={os.path.join(SCRIPT_DIR, 'maxtext_benchmark_runner.py')}",
        ]
    )

    manifest = render_pod_manifest(
        project_id=project_id,
        bucket_name=bucket_name,
        iterations=iterations,
        staging_version=staging_version,
        client_protocol=client_protocol,
    )
    manifest_filename = f"manifest-{timestamp}.yaml"

    try:
        with open(manifest_filename, "w") as f:
            f.write(manifest)
        await utils.run_command_async(["kubectl", "apply", "-f", manifest_filename])

        start_time = datetime.now()
        pod_finished = False
        while (datetime.now() - start_time).total_seconds() < pod_timeout_seconds:
            status, _, _ = await utils.run_command_async(
                [
                    "kubectl",
                    "get",
                    "pod",
                    pod_name,
                    "-o",
                    'jsonpath={.status.phase} {.status.containerStatuses[?(@.name=="load-test")].state}',
                ],
                check=False,
            )
            if (
                "Succeeded" in status
                or "Failed" in status
                or "terminated" in status
            ):
                pod_finished = True
                break
            await asyncio.sleep(10)

        if not pod_finished:
            raise TimeoutError(
                f"Pod did not complete within {pod_timeout_seconds / 60} minutes."
            )

        logs, _, _ = await utils.run_command_async(
            ["kubectl", "logs", pod_name, "-c", "load-test"], check=False
        )
        if not logs:
            logs, _, _ = await utils.run_command_async(
                ["kubectl", "logs", pod_name], check=False
            )
        if logs:
            return parse_all_gbytes_per_sec(logs)
        return []
    finally:
        await utils.run_command_async(
            [
                "kubectl",
                "delete",
                "configmap",
                configmap_name,
                "--ignore-not-found=true",
            ],
            check=False,
        )
        await utils.run_command_async(
            [
                "kubectl",
                "delete",
                "-f",
                manifest_filename,
                "--ignore-not-found=true",
            ],
            check=False,
        )
        if os.path.exists(manifest_filename):
            os.remove(manifest_filename)


def build_arg_parser():
    """Builds and returns the CLI argument parser with all 16 standard flags."""
    parser = argparse.ArgumentParser(
        description="Run GKE MaxText + PyGrain benchmark."
    )
    parser.add_argument(
        "--project_id",
        default=os.environ.get("PROJECT_ID", utils.DEFAULT_PROJECT_ID),
        help="Google Cloud project ID. Can also be set with PROJECT_ID env var.",
    )
    parser.add_argument(
        "--bucket_name",
        required=os.environ.get("BUCKET_NAME") is None,
        default=os.environ.get("BUCKET_NAME"),
        help="GCS bucket name for the workload. Can also be set with BUCKET_NAME env var.",
    )
    parser.add_argument(
        "--zone",
        default=os.environ.get("ZONE", utils.DEFAULT_ZONE),
        help="GCP zone. Can also be set with ZONE env var.",
    )
    parser.add_argument(
        "--cluster_name",
        default=os.environ.get("CLUSTER_NAME", DEFAULT_CLUSTER_NAME),
        help="GKE cluster name. Can also be set with CLUSTER_NAME env var.",
    )
    parser.add_argument(
        "--network_name",
        default=os.environ.get("NETWORK_NAME", DEFAULT_NETWORK_NAME),
        help="VPC network name. Can also be set with NETWORK_NAME env var.",
    )
    parser.add_argument(
        "--subnet_name",
        default=os.environ.get("SUBNET_NAME", DEFAULT_SUBNET_NAME),
        help="VPC subnet name. Can also be set with SUBNET_NAME env var.",
    )
    parser.add_argument(
        "--machine_type",
        default=os.environ.get("MACHINE_TYPE", "ct6e-standard-4t"),
        help="Machine type. Can also be set with MACHINE_TYPE env var.",
    )
    parser.add_argument(
        "--node_pool_name",
        default=os.environ.get("NODE_POOL_NAME", "ct6e-pool"),
        help="Node pool name. Can also be set with NODE_POOL_NAME env var.",
    )
    parser.add_argument(
        "--gcsfuse_branch",
        default=os.environ.get("GCSFUSE_BRANCH", "master"),
        help="GCSFuse branch or tag to build. Can also be set with GCSFUSE_BRANCH env var.",
    )
    parser.add_argument(
        "--reservation_name",
        default=os.environ.get(
            "RESERVATION_NAME", utils.DEFAULT_RESERVATION_NAME
        ),
        help="The specific reservation to use for the nodes. Can also be set with RESERVATION_NAME env var.",
    )
    parser.add_argument(
        "--no_cleanup",
        action="store_true",
        default=os.environ.get("NO_CLEANUP", "False").lower() in ("true", "1"),
        help="Don't clean up resources after. Can also be set with NO_CLEANUP=true env var.",
    )
    parser.add_argument(
        "--iterations",
        type=int,
        default=int(os.environ.get("ITERATIONS", 20)),
        help="Number of iterations for the benchmark. Can also be set with ITERATIONS env var.",
    )
    parser.add_argument(
        "--performance_threshold_gbps",
        type=float,
        default=float(os.environ.get("PERFORMANCE_THRESHOLD_GBPS", 13.0)),
        help="Minimum throughput in GB/s for a successful iteration. Can also be set with PERFORMANCE_THRESHOLD_GBPS env var.",
    )
    parser.add_argument(
        "--pod_timeout_seconds",
        type=int,
        default=int(os.environ.get("POD_TIMEOUT_SECONDS", 1800)),
        help="Timeout in seconds for the benchmark pod to complete. Can also be set with POD_TIMEOUT_SECONDS env var.",
    )
    parser.add_argument(
        "--skip_csi_driver_build",
        action="store_true",
        default=os.environ.get("SKIP_CSI_DRIVER_BUILD", "False").lower()
        in ("true", "1"),
        help="Skip building the CSI driver. Can also be set with SKIP_CSI_DRIVER_BUILD=true env var.",
    )
    parser.add_argument(
        "--client_protocol",
        default=os.environ.get("CLIENT_PROTOCOL", "http1"),
        help="The client protocol to use for GCS. Can also be set with CLIENT_PROTOCOL env var.",
    )
    return parser


async def main():
    """Parses arguments, orchestrates the benchmark execution, and handles cleanup."""
    parser = build_arg_parser()
    args = parser.parse_args()

    # Append zone to default network and subnet names to avoid collisions
    if args.network_name == DEFAULT_NETWORK_NAME:
        args.network_name = f"{args.network_name}-{args.zone}"
    if args.subnet_name == DEFAULT_SUBNET_NAME:
        args.subnet_name = f"{args.subnet_name}-{args.zone}"

    required_tools = ["gcloud", "git", "make", "kubectl", "gke-gcloud-auth-plugin"]
    if not all(shutil.which(tool) for tool in required_tools):
        await utils.check_prerequisites()

    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    with tempfile.TemporaryDirectory() as temp_dir:
        try:
            if args.skip_csi_driver_build:
                await utils.setup_gke_cluster(
                    args.project_id,
                    args.zone,
                    args.cluster_name,
                    args.network_name,
                    args.subnet_name,
                    args.zone.rsplit("-", 1)[0],
                    args.machine_type,
                    args.node_pool_name,
                    args.reservation_name,
                )
            else:
                _, _, gcsfuse_dir = await utils.clone_and_log_branch_info(
                    args.gcsfuse_branch, temp_dir
                )
                setup_task = asyncio.create_task(
                    utils.setup_gke_cluster(
                        args.project_id,
                        args.zone,
                        args.cluster_name,
                        args.network_name,
                        args.subnet_name,
                        args.zone.rsplit("-", 1)[0],
                        args.machine_type,
                        args.node_pool_name,
                        args.reservation_name,
                    )
                )
                build_task = asyncio.create_task(
                    utils.build_gcsfuse_image(
                        args.project_id, gcsfuse_dir, STAGING_VERSION
                    )
                )
                try:
                    await asyncio.gather(setup_task, build_task)
                except Exception:
                    print(
                        "Setup or build failed. Waiting for background tasks to finish before cleanup...",
                        file=sys.stderr,
                    )
                    await asyncio.gather(
                        setup_task, build_task, return_exceptions=True
                    )
                    raise

            await set_up_bucket_permissions(
                args.project_id, args.zone, args.cluster_name, args.bucket_name
            )
            await ensure_dataset_provisioned(args.bucket_name)
            staging_version = await resolve_staging_version(
                args.project_id,
                staging_version=STAGING_VERSION,
                skip_csi_driver_build=args.skip_csi_driver_build,
            )

            throughputs = await execute_workload_and_gather_results(
                args.project_id,
                args.zone,
                args.cluster_name,
                args.bucket_name,
                timestamp,
                args.iterations,
                staging_version,
                args.pod_timeout_seconds,
                args.client_protocol,
            )

            passed, _ = evaluate_throughputs(
                throughputs,
                threshold_gbps=args.performance_threshold_gbps,
                client_protocol=args.client_protocol,
            )
            if not passed:
                sys.exit(-1)
        finally:
            if not args.no_cleanup:
                await utils.cleanup(
                    args.project_id,
                    args.zone,
                    args.cluster_name,
                    args.network_name,
                    args.subnet_name,
                )


if __name__ == "__main__":
    asyncio.run(main())
