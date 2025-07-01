# **GCSFuse Sample Configurations**

This directory contains sample GCSFuse configuration files optimized for common machine learning workloads. These configurations are intended for use with vanilla GCSFuse and may require minor adjustments for your specific environment.

The configurations are organized into two main directories:

* **GPU:** Configurations optimized for GPU-based workloads.  
* **TPU:** Configurations optimized for TPU-based workloads.

Within each directory, you will find configurations tailored for the following workflows:

1. **Model Training:** Optimized for training machine learning models.  
2. **Model Serving/Inference:** Optimized for serving predictions with trained models.  
3. **Checkpointing:** Designed for saving model state during training, also applicable to JAX Just-In-Time (JIT) Cache workflows.

Choose the configuration that best matches your hardware platform (GPU or TPU) and workload.

## **Deployment Instructions**

To use these sample configurations, you can use the `gcsfuse` command with the `--config-file` flag.

For example, to mount a bucket for model training on a GPU machine:

```
gcsfuse --config-file GPU/training.yaml <your-bucket-name> <mount-point>
```

Replace `<your-bucket-name>` with the name of your GCS bucket and `<mount-point>` with the desired mount point on your local filesystem.

### **Usage with GCE, SLURM, and Ansible**

These configurations are designed to be versatile and can be used in various environments, including Google Compute Engine (GCE) instances and SLURM clusters.

#### **GCE and SLURM**

In a GCE or SLURM environment, you can use the same `gcsfuse` command to mount your GCS buckets on your compute nodes. It's recommended to automate this process as part of your instance startup scripts or SLURM job prolog scripts. This ensures that the required GCS buckets are mounted and available to your applications before they start.

Here is an [example](https://github.com/GoogleCloudPlatform/cluster-toolkit/blob/51c51f2c83383a8f241cd0ef8a8998413393bff5/examples/hypercompute_clusters/a3u-slurm-ubuntu-gcs/a3u-slurm-ubuntu-gcs.yaml#L193) of using config with Ansible.

## **Key Considerations**

* **File Cache:**

  * For GPU-based workloads, it is highly recommended to use a Local SSD (LSSD) for the file cache directory for optimal performance.  
  * For TPU-based workloads, using a RAM disk for the file cache can provide a significant speed advantage.  
* **Metadata Cache TTL:**

  * The sample configurations may use a long or infinite Time-to-Live (TTL) for the metadata cache and there might be consistency implications. More details can be found [here](https://cloud.google.com/storage/docs/cloud-storage-fuse/performance#increase-metadata-cache-values).
* **Pre-populating Metadata Cache:**

  * To improve the performance of subsequent file and directory lookups, you can pre-populate the metadata cache after mounting the bucket by running the following command. More details can be found [here](https://cloud.google.com/storage/docs/cloud-storage-fuse/performance#pre-populate-the-metadata-cache):

```
      ls -R <mount-point> > /dev/null
```
* **Tuning `read-ahead-kb`** 

  * The `read-ahead-kb` flag controls the size of kernel read-ahead buffer which in turns impacts number of requests to GCSFuse. Tuning this value can significantly improve performance for sequential reads of large files.
  * A good starting point for `read-ahead-kb` is to set it to a value slightly larger than the average read size of your application. Typical recommendation is to set it to `1024`.  
```
      export GCSFUSEMOUNT=/your/container/mountpoint
      echo 1024 | sudo tee /sys/class/bdi/0:$(stat -c "%d" $GCSFUSEMOUNT)/read_ahead_kb
```


## **Prerequisites and Notes**

* **Placeholders:** Remember to replace all placeholder values within the YAML configuration files with your specific environment details before use.  
* **Permissions:** Ensure that the service account or user running GCSFuse has the necessary permissions to access the specified GCS bucket.

## **Further Information**

For comprehensive details on GCSFuse configuration and performance tuning, please consult the official documentation: [https://cloud.google.com/storage/docs/gcsfuse](https://cloud.google.com/storage/docs/gcsfuse)
