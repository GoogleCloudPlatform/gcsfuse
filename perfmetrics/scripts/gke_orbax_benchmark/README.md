# GKE Orbax Benchmark

This script automates the process of running the Orbax benchmark on a GKE cluster. It handles the entire workflow, including GKE cluster setup, GCSFuse CSI driver building, workload execution, result gathering, and resource cleanup.

## Prerequisites

Before running the script, ensure you have the following tools installed and configured:

-   `gcloud`: The Google Cloud CLI, authenticated with a project.
-   `kubectl`: The Kubernetes command-line tool.
-   `git`: The version control system.
-   `make`: The build automation tool.
-   `python3` with the `asyncio` library (standard in Python 3.7+).

## Usage

The script is controlled via command-line arguments.

```
usage: run_benchmark.py [-h] --project_id PROJECT_ID --bucket_name BUCKET_NAME [--zone ZONE] [--cluster_name CLUSTER_NAME] [--network_name NETWORK_NAME] [--subnet_name SUBNET_NAME]
                        [--machine_type MACHINE_TYPE] [--node_pool_name NODE_POOL_NAME] [--gcsfuse_branch GCSFUSE_BRANCH] [--no_cleanup]

Run GKE Orbax benchmark.

options:
  -h, --help            show this help message and exit
  --project_id PROJECT_ID
                        Google Cloud project ID.
  --bucket_name BUCKET_NAME
                        GCS bucket name for the workload.
  --zone ZONE           GCP zone.
  --cluster_name CLUSTER_NAME
                        GKE cluster name.
  --network_name NETWORK_NAME
                        VPC network name.
  --subnet_name SUBNET_NAME
                        VPC subnet name.
  --machine_type MACHINE_TYPE
                        Machine type for the node pool.
  --node_pool_name NODE_POOL_NAME
                        Node pool name.
  --gcsfuse_branch GCSFUSE_BRANCH
                        GCSFuse branch or tag to build.
  --no_cleanup          Don't clean up resources after the benchmark.
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
