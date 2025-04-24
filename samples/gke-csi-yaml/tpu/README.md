# GKE GCSFuse CSI Driver Sample Configurations for TPU Workloads

This directory provides sample Kubernetes YAML configuration files for utilizing the GKE GCSFuse CSI driver, specifically optimized for workloads running on Tensor Processing Units (TPUs).

The configurations include recommendations tailored for the following common machine learning workflows:

1.  **Model Training:** Configurations suitable for training machine learning models.
2.  **Model Serving/Inference:** Configurations optimized for deploying trained models to serve predictions.
3.  **Checkpointing:** Configurations designed for saving model state during training processes. These configurations are also applicable to JAX Just-In-Time (JIT) Cache workflows.

## Deployment Instructions

**Note: The sample files are for GCSfuse GKE CSI driver running on GKE clusters of GKE version 1.32.2-gke.1297001 or greater.**

To utilize these sample configurations, follow the specified deployment order. For instance, to set up the serving workload:

1.  **Deploy the PersistentVolume (PV) and PersistentVolumeClaim (PVC):**
    Apply the `*-pv.yaml` file first. This step is crucial as the GKE pod admission webhook inspects the PV's volume attributes to apply potential optimizations, such as the injection of sidecar containers, before the pod is scheduled.
    ```bash
    kubectl apply -f serving-pv.yaml
    ```

2.  **Deploy the Pod:**
    After the PV and PVC are successfully created, deploy the pod specification that references the PVC.
    ```bash
    kubectl apply -f serving-pod.yaml
    ```

## Prerequisites and Notes

*   **Service Account:** Ensure the specified Kubernetes Service Account (e.g., `<YOUR_K8S_SA>` in the pod YAML) exists and possesses the necessary permissions to access the target Google Cloud Storage bucket *before* deploying the pod.
*   **Placeholders:** Replace all placeholder values (e.g., `<customer-namespace>`, `<checkpoint-bucket>`, `<YOUR_K8S_SA>`) within the YAML files with your specific environment details before application.

## Further Information

For comprehensive details on performance tuning and best practices for the GCS FUSE CSI driver, please consult the Google Cloud documentation:
[Best practices for performance tuning](https://cloud.google.com/kubernetes-engine/docs/how-to/cloud-storage-fuse-csi-driver-perf#best-practices-for-performance-tuning)