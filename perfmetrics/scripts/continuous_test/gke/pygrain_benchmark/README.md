# GKE MaxText + PyGrain TPU Data Loading Benchmark

This directory contains the continuous GKE TPU v6e (`ct6e-standard-4t`) benchmark suite for evaluating GCSFuse read throughput under a multi-worker **MaxText + PyGrain Multimodal (`ArrayRecord`)** training data pipeline.

## Overview

- **`run_benchmark.py`**: End-to-end orchestrator that integrates with `perfmetrics/scripts/continuous_test/gke/common/utils.py` to:
  1. Verify prerequisite CLI tools (`gcloud`, `git`, `make`, `kubectl`, `gke-gcloud-auth-plugin`).
  2. Set up a GKE cluster and Cloud TPU v6e node pool (`ct6e-standard-4t`, `2x2` topology).
  3. Build a staging GCSFuse CSI driver sidecar image (`prow-gob-internal-boskos-pygrain-benchmark`) unless `--skip_csi_driver_build` is set.
  4. Configure Workload Identity bucket IAM bindings (`roles/storage.objectUser` for the `default` KSA) and check for the 128 synthetic `ArrayRecord` dataset shards (`train-00000-of-00128.arrayrecord` .. `train-00127-of-00128.arrayrecord`).
  5. Deploy a Kubernetes ConfigMap (`pygrain-benchmark`) and Pod (`gcsfuse-test` rendered from `pod.yaml.template`).
  6. Parse per-iteration throughput metrics (`gbytes_per_sec: <value> Bytes/s`) and enforce the **5/8 threshold rule** (`>= 13.0` GB/s default, with a warning fallback when `client_protocol == "grpc"`).
  7. Clean up temporary Kubernetes resources (`pod/gcsfuse-test`, `configmap/pygrain-benchmark`) and GKE cluster/VPC resources (unless `--no_cleanup` is set).
- **`generate_data.py`**: Self-contained synthetic Multimodal LLM (VLM) `ArrayRecord` dataset generator. Synthesizes 128 x 512 MiB `.arrayrecord` shards (`64.04 GiB` total, `2.0 MiB` records containing `tokens`, `positions`, `image_masks`, and `image_payload`) using two-phase `/dev/shm/staging` template synthesis + 32-worker parallel GCSFuse fan-out. Automatically skips regeneration and reuses existing shards if all 128 shards are already present in `--output-dir`.
- **`maxtext_benchmark_runner.py`**: Multi-worker PyGrain (`ArrayRecordDataSource`, `IndexSampler`, `UnpackMultimodalRecord`, `Batch`) + MaxText TPU data-loader runner with VFS page-cache eviction (`POSIX_FADV_DONTNEED` + `/proc/sys/vm/drop_caches`) and `eth0` NIC telemetry.
- **`pod.yaml.template`**: GKE TPU v6e Pod template mounting the GCSFuse CSI driver sidecar and `/gcs/data` volume.
- **`continuous.cfg`** and **`../pygrain_benchmark_grpc/continuous.cfg`**: Kokoro continuous test configurations for `http1` and `grpc` protocols.

## Running the Benchmark

### Continuous Test (`http1` / `grpc` defaults)

```bash
python3 perfmetrics/scripts/continuous_test/gke/pygrain_benchmark/run_benchmark.py \
  --project_id=gcs-fuse-test-ml \
  --zone=europe-west4-a \
  --bucket_name=llama_europe_west4
```

### Running on an Existing TPU v6e Cluster

```bash
python3 perfmetrics/scripts/continuous_test/gke/pygrain_benchmark/run_benchmark.py \
  --project_id=gcs-fuse-test \
  --zone=us-east5-b \
  --cluster_name=kislayk-orbax-checkpoint-cluster \
  --node_pool_name=tpu \
  --bucket_name=kislayk-pygrain-rapid-useast5b \
  --iterations=8 \
  --no_cleanup \
  --skip_csi_driver_build
```

## Running Unit Tests

```bash
python3 -m unittest discover -s perfmetrics/scripts/continuous_test/gke/pygrain_benchmark -p "test_*.py" -v
```
