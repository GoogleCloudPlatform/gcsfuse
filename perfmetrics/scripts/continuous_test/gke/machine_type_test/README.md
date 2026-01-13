# GKE Machine Type Test

This script automates the process of running the Machine Type Test on a GKE
cluster. It handles the entire workflow, including GKE cluster setup, GCSFuse
CSI driver building, workload execution, result gathering, and resource cleanup.

## Overview

The `run.py` script automates the end-to-end testing process on Google
Kubernetes Engine (GKE). It handles:
1. **Cluster Management**: Creating/configuring GKE clusters and node pools (including Workload Identity).
2. **Driver Build**: Building the GCSFuse CSI driver from source.
3. **Workload Deployment**: Deploying a Kubernetes Pod to run `go test` integration tests.
4. **Result Verification**: Checking test success/failure.
5. **Cleanup**: Removing cloud resources.

## Prerequisites

Before running the script, ensure you have the following tools installed and
configured. The script will check for these and attempt to install `kubectl` if
it's missing.

### Tools

-   `gcloud`: The Google Cloud CLI, authenticated with a project. Ensure the
    following APIs are enabled in your project:
    -   Kubernetes Engine API (`container.googleapis.com`)
    -   Cloud Storage API (`storage.googleapis.com`)
-   `kubectl`: The Kubernetes command-line tool.
-   `git`: The version control system.
-   `make`: The build automation tool.
-   `python3` with the `asyncio` library (standard in Python 3.7+).

### Workload Identity Setup (Critical)

The test uses GKE Workload Identity Federation. You must grant the Kubernetes
Service Account (KSA) direct access to the GCS bucket. For this test, we use the
`default` KSA in the `default` namespace.

**1. Grant Bucket Permissions to the KSA Principal:** You need the **Project
Number** (not ID) of the project hosting the GKE cluster.

```bash
# Get Project Number
PROJECT_NUMBER=$(gcloud projects describe <PROJECT_ID> --format="value(projectNumber)")

# Grant Storage Object User role to the 'default' KSA principal
gcloud storage buckets add-iam-policy-binding gs://<BUCKET_NAME> \
    --member="principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/<PROJECT_ID>.svc.id.goog/subject/ns/default/sa/default" \
    --role=roles/storage.objectUser \
    --project=<BUCKET_PROJECT_ID>
```

*Note: This has already been configured for the test environment
(project: `gcs-fuse-test-ml`, bucket: `gcsfuse_gke_machine_type_test_flat_euw4`).*

## Workflow

The script performs the following steps:

1.  **Prerequisite Check**: Verifies that `gcloud`, `git`, `make`, `kubectl`,
    and `gke-gcloud-auth-plugin` are installed.
2.  **VPC Network and Subnet Setup**: Creates a VPC network and subnet if they
    don't already exist.
3.  **GKE Cluster Setup**: Creates a new GKE cluster with a dedicated node pool
    if one doesn't already exist. If the node pool is unhealthy, it's recreated.
4.  **Build GCSFuse CSI Driver**: Concurrently with cluster setup, it clones the
    specified GCSFuse repository branch and builds the GCSFuse CSI driver
    container image.
5.  **Run Test**: Deploys the test workload as a Kubernetes Pod.
    *   It automatically selects the appropriate node-pool based on the machine
        type.
6.  **Gather Results**: Fetches the logs from the completed test pod.
7.  **Evaluate Success**: Checks if the pod completed successfully.
8.  **Cleanup**: Deletes the GKE cluster and other created resources like the
    VPC network, subnet, and associated firewall rules, unless the
    `--no_cleanup` flag is specified.

## Usage

The script is controlled via command-line arguments.

```
usage: run.py [-h] --project_id PROJECT_ID --bucket_name BUCKET_NAME --zone ZONE \
                   [--cluster_name CLUSTER_NAME] [--network_name NETWORK_NAME] [--subnet_name SUBNET_NAME] \
                   [--machine_type MACHINE_TYPE] [--node_pool_name NODE_POOL_NAME] \
                   [--gcsfuse_branch GCSFUSE_BRANCH] [--reservation_name RESERVATION_NAME] \
                   [--no_cleanup] [--skip_csi_driver_build] [--pod_timeout_seconds POD_TIMEOUT_SECONDS]
```

### Argument Reference

Argument                  | Description                                                      | Default Value
:------------------------ | :--------------------------------------------------------------- | :------------
`--project_id`            | Google Cloud project ID.                                         | `gcs-fuse-test-ml` (Env: `PROJECT_ID`)
`--bucket_name`           | **(Required)** GCS bucket name for the workload.                 | `None` (Env: `BUCKET_NAME`)
`--zone`                  | GCP zone.                                                        | `europe-west4-a` (Env: `ZONE`)
`--cluster_name`          | GKE cluster name.                                                | `gke-machine-type-test-cluster`
`--network_name`          | VPC network name.                                                | `gke-machine-type-test-network-<ZONE>`
`--subnet_name`           | VPC subnet name.                                                 | `gke-machine-type-test-subnet-<ZONE>`
`--machine_type`          | Machine type for the node pool.                                  | `ct6e-standard-4t` (TPU v6e)
`--node_pool_name`        | Node pool name.                                                  | `ct6e-pool`
`--gcsfuse_branch`        | GCSFuse branch to build.                                         | `master`
`--reservation_name`      | Specific reservation to use for the nodes.                       | `cloudtpu-20251107233000-76736260`
`--no_cleanup`            | If set, resources will NOT be deleted after the test.            | `False`
`--skip_csi_driver_build` | If set, skips building the CSI driver image (assumes it exists). | `False`
`--pod_timeout_seconds`   | Timeout in seconds for the pod to complete.                      | `1800` (30 mins)

## Examples

To run the test with default settings (TPU v6e), you only need to provide the
required arguments:

```bash
python3 perfmetrics/scripts/continuous_test/gke/machine_type_test/run.py \
  --project_id "your-gcp-project-id" \
  --bucket_name "your-gcs-bucket-name" \
  --zone "us-central1-a"
```

To run on a **TPU Machine Type** (`ct6e-standard-4t`) using a reservation and a
specific branch:

```bash
python3 perfmetrics/scripts/continuous_test/gke/machine_type_test/run.py \
  --project_id "your-gcp-project-id" \
  --bucket_name "your-gcs-bucket-name" \
  --zone "europe-west4-a" \
  --machine_type ct6e-standard-4t \
  --node_pool_name tpu-v6-pool \
  --reservation_name "your-reservation-name" \
  --gcsfuse_branch "my-feature-branch" \
  --no_cleanup \
  --skip_csi_driver_build
```

## Troubleshooting

*   **403 Forbidden**: Check Workload Identity setup. Ensure `default` KSA
    exists and has the read/write access to the bucket.
*   **Init:ErrImagePull**: Check if the CSI driver image exists in GCR/Artifact
    Registry. If running locally, you might need to authenticate docker.
