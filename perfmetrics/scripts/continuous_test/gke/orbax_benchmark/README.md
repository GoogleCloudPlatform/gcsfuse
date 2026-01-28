# GKE Orbax Benchmark

This script automates the process of running the Orbax benchmark on a GKE cluster. It handles the entire workflow, including GKE cluster setup, GCSFuse CSI driver building, workload execution, result gathering, and resource cleanup.

## Prerequisites

Before running the script, ensure you have the following tools installed and configured. The script will check for these and attempt to install `kubectl` if it's missing.

-   `gcloud`: The Google Cloud CLI, authenticated with a project. Ensure the following APIs are enabled in your project:
    -   Kubernetes Engine API (`container.googleapis.com`)
    -   Cloud Storage API (`storage.googleapis.com`)
-   `kubectl`: The Kubernetes command-line tool.
-   `git`: The version control system.
-   `make`: The build automation tool.
-   `python3` with the `asyncio` library (standard in Python 3.7+).

## Workflow

The script performs the following steps:

1.  **Prerequisite Check**: Verifies that `gcloud`, `git`, `make`, and `kubectl` are installed.
2.  **VPC Network and Subnet Setup**: Creates a VPC network and subnet if they don't already exist.
3.  **GKE Cluster Setup**: Creates a new GKE cluster with a dedicated node pool if one doesn't already exist. If the node pool is unhealthy, it's recreated.
4.  **Build GCSFuse CSI Driver**: Concurrently with cluster setup, it clones the specified GCSFuse repository branch and builds the GCSFuse CSI driver container image.
5.  **Run Benchmark**: Deploys the Orbax benchmark workload as a Kubernetes Pod.
6.  **Gather and Parse Results**: Fetches the logs from the completed benchmark pod and parses them to extract throughput metrics.
7.  **Evaluate Performance**: Compares the results against a performance threshold to determine if the benchmark passed.
8.  **Cleanup**: Deletes the GKE cluster and other created resources like the VPC network, subnet, and associated firewall rules, unless the `--no_cleanup` flag is specified.

## Usage

The script is controlled via command-line arguments.

```
usage: run_benchmark.py [-h] [--project_id PROJECT_ID] --bucket_name BUCKET_NAME [--zone ZONE] \
                        [--cluster_name CLUSTER_NAME] [--network_name NETWORK_NAME] \
                        [--subnet_name SUBNET_NAME] [--machine_type MACHINE_TYPE] \
                        [--node_pool_name NODE_POOL_NAME] [--gcsfuse_branch GCSFUSE_BRANCH] \
                        [--reservation_name RESERVATION_NAME] [--no_cleanup] \
                        [--iterations ITERATIONS] \
                        [--performance_threshold_gbps PERFORMANCE_THRESHOLD_GBPS] \
                        [--pod_timeout_seconds POD_TIMEOUT_SECONDS] [--skip_csi_driver_build]
```

### Argument Reference

Argument                  | Description                                                      | Default Value
:------------------------ | :--------------------------------------------------------------- | :------------
`--project_id`            | Google Cloud project ID.                                         | `gcs-fuse-test-ml` (Env: `PROJECT_ID`)
`--bucket_name`           | **(Required)** GCS bucket name for the workload.                 | `None` (Env: `BUCKET_NAME`)
`--zone`                  | GCP zone.                                                        | `europe-west4-a` (Env: `ZONE`)
`--cluster_name`          | GKE cluster name.                                                | `gke-orbax-benchmark-cluster`
`--network_name`          | VPC network name.                                                | `gke-orbax-benchmark-network-<ZONE>`
`--subnet_name`           | VPC subnet name.                                                 | `gke-orbax-benchmark-subnet-<ZONE>`
`--machine_type`          | Machine type for the node pool.                                  | `ct6e-standard-4t` (TPU v6e)
`--node_pool_name`        | Node pool name.                                                  | `ct6e-pool`
`--gcsfuse_branch`        | GCSFuse branch to build.                                         | `master`
`--reservation_name`      | Specific reservation to use for the nodes.                       | `cloudtpu-20251107233000-76736260`
`--no_cleanup`            | If set, resources will NOT be deleted after the test.            | `False`
`--iterations`            | Number of iterations for the benchmark.                          | `20`
`--performance_threshold_gbps` | Minimum throughput in GB/s for a successful iteration.      | `13.0`
`--pod_timeout_seconds`   | Timeout in seconds for the pod to complete.                      | `1800` (30 mins)
`--skip_csi_driver_build` | If set, skips building the CSI driver image (assumes it exists). | `False`
```

## Example

To run the benchmark with default settings, you only need to provide your Google Cloud project ID and the GCS bucket name for the workload:

```bash
python3 perfmetrics/scripts/gke_orbax_benchmark/run_benchmark.py \
  --project_id "your-gcp-project-id" \
  --bucket_name "your-gcs-bucket-name"
```

To run on a specific GCSFuse branch and prevent cleanup after the run:

```bash
python3 perfmetrics/scripts/gke_orbax_benchmark/run_benchmark.py \
  --project_id "your-gcp-project-id" \
  --bucket_name "your-gcs-bucket-name" \
  --gcsfuse_branch "my-feature-branch" \
  --no_cleanup
```
