This folder contains sample YAML files for GKE GCSFuse CSI driver with recommendations for GPUs/TPUs.

The following workflow samples are added
1. Training
2. Serving or Inference
3. Checkpointing

For JAX Jit Cache workflows, use the Checkpointing workflow yaml configs.

For e.g. To set up the serving workload.

Deploy the PVC/PV first (this is required because the GKE pod webhook inspects the PV volume attributes for additional optimizations like injection of additional containers)
e.g.
``` 
kubectl apply -f serving-pv.yaml
```

Then, deploy the pod spec that accesses the PVC
e.g.
```
kubectl apply -f serving-pod.yaml
```

Note:
* The service account needs to be created before use.
* Replace placeholders with actual values.

Read https://cloud.google.com/kubernetes-engine/docs/how-to/cloud-storage-fuse-csi-driver-perf#best-practices-for-performance-tuning  for more tuning details.
