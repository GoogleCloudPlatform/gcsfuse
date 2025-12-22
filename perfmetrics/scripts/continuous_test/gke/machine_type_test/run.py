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

"""Run GKE Machine-type test.

This script automates the process of running the Machine-type test on a GKE
cluster.
It performs the following steps:
1.  Checks for prerequisite tools (gcloud, git, make, kubectl).
2.  Sets up a GKE cluster with a specific node pool if it doesn't exist.
3.  Builds a GCSFuse CSI driver image from a specified git branch.
4.  Deploys a Kubernetes pod that runs the test workload.
5.  Streams logs from the Kubernetes pod.
6.  Determines if the test passed based on the pod exit status.
7.  Cleans up all created cloud resources (GKE cluster, network, etc.).
"""

import argparse
import asyncio
from datetime import datetime
import os
import shlex
from string import Template
import subprocess
import sys
import tempfile

# Add the parent directory to sys.path to allow imports from common
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
sys.path.append(os.path.abspath(os.path.join(SCRIPT_DIR, "..")))
from common import utils

# The prefix prow-gob-internal-boskos- is needed to allow passing machine-type from gke csi driver to gcsfuse,
# bypassing the check at
# https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver/blob/15afd00dcc2cfe0f9753ddc53c81631ff037c3f2/pkg/csi_driver/utils.go#L532.
STAGING_VERSION = "prow-gob-internal-boskos-machine-type-test"
DEFAULT_MTU = 8896


async def set_up_bucket_permissions(
    project_id, zone, cluster_name, bucket_name
):
  # Authenticate kubectl
  await utils.run_command_async([
      "gcloud",
      "container",
      "clusters",
      "get-credentials",
      cluster_name,
      f"--project={project_id}",
      f"--zone={zone}",
  ])

  # Fetch current-context
  await utils.run_command_async([
      "kubectl",
      "config",
      "current-context",
  ])

  # Ensure default KSA exists
  await utils.run_command_async(
      ["kubectl", "create", "serviceaccount", "default", "--namespace=default"],
      check=False,
  )

  # Set current-context
  await utils.run_command_async([
      "kubectl",
      "config",
      "set-context",
      "--current",
      "--namespace=default",
  ])

  # Get Project Number
  project_number, _, _ = await utils.run_command_async(
      [
          "gcloud",
          "projects",
          "describe",
          project_id,
          "--format=value(projectNumber)",
      ],
      check=True,
  )

  # Grant Storage objectUser role to the 'default' KSA principal
  principal = f"principal://iam.googleapis.com/projects/{project_number}/locations/global/workloadIdentityPools/{project_id}.svc.id.goog/subject/ns/default/sa/default"
  print(
      f"Granting roles/storage.objectUser to {principal} on bucket"
      f" {bucket_name}..."
  )

  await utils.run_command_async([
      "gcloud",
      "storage",
      "buckets",
      "add-iam-policy-binding",
      f"gs://{bucket_name}",
      f"--member={principal}",
      "--role=roles/storage.objectUser",
      f"--project={project_id}",
  ])


def is_tpu_machine_type(machine_type):
  """Checks if the machine type is a TPU machine type."""
  # Heuristic: check for "ct" (Cloud TPU) or "tpu" in the name.
  return machine_type.startswith("ct") or "tpu" in machine_type


# Workload Execution and Result Gathering
async def execute_test_workload(
    project_id,
    zone,
    cluster_name,
    bucket_name,
    timestamp,
    staging_version,
    pod_timeout_seconds,
    machine_type,
    gcsfuse_branch,
):
  """Executes the workload pod, gathers results, and cleans up workload resources.

  This function creates a Kubernetes Pod to run the test.
  It waits for the pod to complete, collects its logs,
  and then deletes the created Kubernetes resources.

  Args:
      project_id: The Google Cloud project ID.
      zone: The GCP zone of the cluster.
      cluster_name: The name of the GKE cluster.
      bucket_name: The GCS bucket to use for the test.
      timestamp: A unique timestamp string for manifest naming.
      staging_version: The version tag for the GCSFuse CSI driver image.
      pod_timeout_seconds: The timeout in seconds for the pod to complete.
      machine_type: The machine type of the node pool.
      gcsfuse_branch: The gcsfuse branch to clone.

  Returns:
      True if the test passed, False otherwise.
  """
  print(f"Executing workload for machine type: {machine_type}...")

  if not is_tpu_machine_type(machine_type):
    raise ValueError(
        f"Machine type {machine_type} is not supported. Only TPU machine types"
        " are supported."
    )

  template_file = "pod_tpu.yaml.template"

  template_path = os.path.join(SCRIPT_DIR, template_file)
  print(f"Using pod template: {template_path}")

  with open(template_path, "r") as f:
    pod_template = Template(f.read())

  # Use timestamp to make pod name unique to avoid conflict
  pod_name = f"gcsfuse-gke-machine-type-test-{timestamp}"
  print(f"Pod name: {pod_name}")

  manifest = pod_template.safe_substitute(
      project_id=project_id,
      bucket_name=bucket_name,
      staging_version=staging_version,
      gcsfuse_branch=gcsfuse_branch,
      machine_type=machine_type,
  )
  # Update the pod name in the manifest content dynamically
  manifest = manifest.replace(
      "name: gcsfuse-gke-machine-type-test", f"name: {pod_name}"
  )

  manifest_filename = f"manifest-{timestamp}.yaml"

  try:
    await utils.run_command_async([
        "kubectl",
        "create",
        "configmap",
        "machine-type-test-scripts",
        f"--from-file={os.path.join(SCRIPT_DIR, 'run_test.sh')}",
    ])

    with open(manifest_filename, "w") as f:
      f.write(manifest)

    # Check if pod exists and delete it (just in case, though name is unique now)
    # We ignore the error if it doesn't exist
    print("Checking for existing pod...")
    await utils.run_command_async(
        ["kubectl", "delete", "pod", pod_name, "--ignore-not-found=true"],
        check=False,
    )

    print(f"Applying manifest: {manifest_filename}")
    await utils.run_command_async(["kubectl", "apply", "-f", manifest_filename])

    start_time = datetime.now()
    pod_finished = False
    success = False

    print(
        f"Waiting for pod {pod_name} to complete (timeout:"
        f" {pod_timeout_seconds}s)..."
    )

    # Wait for pod to be schedulable/running to avoid immediate exit of logs -f
    while (datetime.now() - start_time).total_seconds() < pod_timeout_seconds:
      status, stderr, _ = await utils.run_command_async(
          [
              "kubectl",
              "get",
              "pod",
              pod_name,
              "-o",
              "jsonpath='{.status.phase}'",
          ],
          check=False,
      )
      # jsonpath output comes with quotes, remove them
      status = status.strip("'")
      if status in ["Running", "Succeeded", "Failed"]:
        break
      await asyncio.sleep(5)

    # Stream logs using subprocess.Popen (blocking loop)
    # This waits for the pod to finish (as kubectl logs -f exits on termination)
    log_process = subprocess.Popen(
        ["kubectl", "logs", "-f", pod_name, "-c", "machine-type-test"],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
    )

    while True:
      line = log_process.stdout.readline()
      if not line and log_process.poll() is not None:
        break
      if line:
        print(line, end="")

    # Wait for the pod status to update to Succeeded or Failed
    # kubectl logs -f exits when the container stops, but the pod status update might be delayed.
    for _ in range(12):  # Retry for up to 60 seconds (12 * 5s)
      status, stderr, _ = await utils.run_command_async(
          [
              "kubectl",
              "get",
              "pod",
              pod_name,
              "-o",
              "jsonpath='{.status.phase}'",
          ],
          check=False,
      )
      status = status.strip("'")
      if status in ["Succeeded", "Failed"]:
        break
      print(f"Pod status is '{status}'. Waiting for final status...")
      await asyncio.sleep(5)

    if status == "Succeeded":
      print(f"Pod {pod_name} succeeded.")
      success = True
    elif status == "Failed":
      print(f"Pod {pod_name} failed.")
      success = False
    else:
      # If logs finished but pod is not Succeeded/Failed (e.g. timeout or crash loop?), check timeout
      if (datetime.now() - start_time).total_seconds() > pod_timeout_seconds:
        print(
            f"Pod did not complete within {pod_timeout_seconds / 60} minutes.",
            file=sys.stderr,
        )
        success = False
      else:
        # Fallback: assume failure if logs ended but status isn't clear?
        # Or maybe it just finished and status update is pending?
        # Let's assume if logs exited, the container stopped.
        success = False

    return success
  finally:
    print("Cleaning up pod resources...")
    await utils.run_command_async(
        ["kubectl", "delete", "configmap", "machine-type-test-scripts"],
        check=False,
    )
    await utils.run_command_async(
        ["kubectl", "delete", "-f", manifest_filename], check=False
    )
    if os.path.exists(manifest_filename):
      os.remove(manifest_filename)


# Main function
async def main():
  """Parses arguments, orchestrates the test execution, and handles cleanup.

  This is the main entry point of the script.
  """
  parser = argparse.ArgumentParser(
      description="Run GKE Machine-type test.",
      formatter_class=argparse.ArgumentDefaultsHelpFormatter,
  )
  parser.add_argument(
      "--project_id",
      required=os.environ.get("PROJECT_ID") is None,
      default=os.environ.get("PROJECT_ID"),
      help="Google Cloud project ID. Can also be set with PROJECT_ID env var.",
  )
  parser.add_argument(
      "--bucket_name",
      required=os.environ.get("BUCKET_NAME") is None,
      default=os.environ.get("BUCKET_NAME"),
      help=(
          "GCS bucket name for the workload. Can also be set with BUCKET_NAME"
          " env var."
      ),
  )
  parser.add_argument(
      "--zone",
      required=os.environ.get("ZONE") is None,
      default=os.environ.get("ZONE"),
      help="GCP zone. Can also be set with ZONE env var.",
  )
  parser.add_argument(
      "--cluster_name",
      default=os.environ.get("CLUSTER_NAME", "gke-machine-type-test-cluster"),
      help="GKE cluster name. Can also be set with CLUSTER_NAME env var.",
  )
  parser.add_argument(
      "--network_name",
      default=os.environ.get("NETWORK_NAME", "gke-machine-type-test-network"),
      help="VPC network name. Can also be set with NETWORK_NAME env var.",
  )
  parser.add_argument(
      "--subnet_name",
      default=os.environ.get("SUBNET_NAME", "gke-machine-type-test-subnet"),
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
      help=(
          "GCSFuse branch or tag to build. Can also be set with GCSFUSE_BRANCH"
          " env var."
      ),
  )
  parser.add_argument(
      "--reservation_name",
      default=os.environ.get("RESERVATION_NAME"),
      help=(
          "The specific reservation to use for the nodes. Can also be set with"
          " RESERVATION_NAME env var."
      ),
  )
  parser.add_argument(
      "--no_cleanup",
      action="store_true",
      default=os.environ.get("NO_CLEANUP", "False").lower() in ("true", "1"),
      help=(
          "Don't clean up resources after. Can also be set with NO_CLEANUP=true"
          " env var."
      ),
  )
  parser.add_argument(
      "--pod_timeout_seconds",
      type=int,
      default=int(os.environ.get("POD_TIMEOUT_SECONDS", 1800)),
      help=(
          "Timeout in seconds for the test pod to complete. Can also be"
          " set with POD_TIMEOUT_SECONDS env var."
      ),
  )
  parser.add_argument(
      "--skip_csi_driver_build",
      action="store_true",
      default=os.environ.get("SKIP_CSI_DRIVER_BUILD", "False").lower()
      in ("true", "1"),
      help=(
          "Skip building the CSI driver. Can also be set with"
          " SKIP_CSI_DRIVER_BUILD=true env var."
      ),
  )
  args = parser.parse_args()

  # Append zone to default network and subnet names to avoid collisions
  if args.network_name == "gke-machine-type-test-network":
    args.network_name = f"{args.network_name}-{args.zone}"
  if args.subnet_name == "gke-machine-type-test-subnet":
    args.subnet_name = f"{args.subnet_name}-{args.zone}"

  await utils.check_prerequisites()

  timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
  return_code = 0
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
            DEFAULT_MTU,
            args.reservation_name,
        )
      else:
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
                DEFAULT_MTU,
                args.reservation_name,
            )
        )
        build_task = asyncio.create_task(
            utils.build_gcsfuse_image(
                args.project_id, args.gcsfuse_branch, temp_dir, STAGING_VERSION
            )
        )
        await asyncio.gather(setup_task, build_task)

      await set_up_bucket_permissions(
          args.project_id, args.zone, args.cluster_name, args.bucket_name
      )

      success = await execute_test_workload(
          args.project_id,
          args.zone,
          args.cluster_name,
          args.bucket_name,
          timestamp,
          STAGING_VERSION,
          args.pod_timeout_seconds,
          args.machine_type,
          args.gcsfuse_branch,
      )

      if success:
        print("Test passed successfully.")
      else:
        print("Test failed.", file=sys.stderr)
        return_code = 1
    finally:
      if not args.no_cleanup:
        await utils.cleanup(
            args.project_id,
            args.zone,
            args.cluster_name,
            args.network_name,
            args.subnet_name,
        )

  sys.exit(return_code)


if __name__ == "__main__":
  asyncio.run(main())
