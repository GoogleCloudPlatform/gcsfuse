# User Manual: GKE Machine Type Test

This document outlines the workflow for running the GKE Machine Type Test using the Python-based automation script.

## Overview

The `run.py` script automates the end-to-end testing process on Google Kubernetes Engine (GKE). It handles:
1.  **Cluster Management**: Creating/configuring GKE clusters and node pools (including Workload Identity).
2.  **Driver Build**: Building the GCSFuse CSI driver from source.
3.  **Workload Deployment**: Deploying a Kubernetes Pod to run `go test` integration tests.
4.  **Result Verification**: Checking test success/failure.
5.  **Cleanup**: Removing cloud resources.

## Prerequisites

### Tools
Ensure you have the following tools installed (the script checks for these):
*   `gcloud` CLI
*   `kubectl`
*   `git`
*   `make`
*   `python3`

### Workload Identity Setup (Critical)
The test requires a Kubernetes Service Account (KSA) bound to a Google Service Account (GSA) with permissions to access the GCS bucket.

**1. Create GSA:**
```bash
gcloud iam service-accounts create gcsfuse-machine-type-test-gsa \
    --project=<PROJECT_ID> \
    --display-name="GCSFuse Machine Type Test GSA"
```

**2. Grant Bucket Permissions:**
```bash
gcloud storage buckets add-iam-policy-binding gs://<BUCKET_NAME> \
    --member="serviceAccount:gcsfuse-machine-type-test-gsa@<PROJECT_ID>.iam.gserviceaccount.com" \
    --role=roles/storage.objectUser \
    --project=<BUCKET_PROJECT_ID>
```

**3. Create & Bind KSA (After Cluster Creation):**
You must connect to the cluster first.
```bash
kubectl create serviceaccount gcsfuse-ksa --namespace default

gcloud iam service-accounts add-iam-policy-binding gcsfuse-machine-type-test-gsa@<PROJECT_ID>.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:<PROJECT_ID>.svc.id.goog[default/gcsfuse-ksa]" \
    --project=<PROJECT_ID>

kubectl annotate serviceaccount gcsfuse-ksa \
    --namespace default \
    iam.gke.io/gcp-service-account=gcsfuse-machine-type-test-gsa@<PROJECT_ID>.iam.gserviceaccount.com \
    --overwrite
```
*Note: The `run.py` script does NOT perform these IAM steps automatically. You must ensure the `gcsfuse-ksa` exists in the cluster before running the test, or modify the script/template to use a different SA.*

## Execution Steps

Navigate to the script directory:
`src/gcsfuse/perfmetrics/scripts/continuous_test/gke/machine_type_test/`

### Running the Test

Run the script with the required arguments.

**Recommended Command (Non-TPU):**

```bash
python3 run.py \
  --project_id gcs-fuse-test-ml \
  --bucket_name gargnitin_machine_type_test_hns_euw4 \
  --zone europe-west4-a \
  --machine_type n2-standard-8 \
  --node_pool_name n2-standard-8-pool \
  --no_cleanup \
  --skip_csi_driver_build
```

**Arguments:**
*   `--machine_type`: Use `n2-standard-8` or larger (min 2 vCPU required for sidecar).
*   `--node_pool_name`: Specify a unique name to create a new pool with correct scopes if default exists.
*   `--no_cleanup`: Keeps resources for debugging.
*   `--skip_csi_driver_build`: Skips build if image already exists.

## Troubleshooting

*   **403 Forbidden**: Check Workload Identity setup. Ensure `gcsfuse-ksa` is annotated correctly.
*   **Insufficient CPU**: Ensure the machine type has at least 4 vCPUs (sidecar requests 2, load-test needs some).
*   **Init:ErrImagePull**: Check if the CSI driver image exists in GCR.