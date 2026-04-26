# GCSFuse Metrics

For instructions on how to enable and use Cloud Storage FUSE metrics, refer to the metrics guide at <https://cloud.google.com/storage/docs/cloud-storage-fuse/metrics>.

---

## Health Check Endpoints

GCSFuse exposes two HTTP health check endpoints on the same port as the Prometheus metrics endpoint (`--prometheus-port`). These endpoints are designed for use with Kubernetes liveness/readiness probes, load balancers, and monitoring systems.

> **Prerequisite:** Both endpoints are only available when `--prometheus-port` is set to a value greater than `0`. They are not available if only `--cloud-metrics-export-interval-secs` is used.

### `/healthz` — Liveness

Reports whether the GCSFuse process is alive and the FUSE mount is active.

| Response | Meaning |
|---|---|
| `200 OK` — body `ok` | The mount has completed successfully and has not yet been torn down. |
| `503 Service Unavailable` | The mount has not yet succeeded, or is in the process of shutting down. |

This check is **purely in-memory** — it reads an atomic flag set by the mount lifecycle. It makes no GCS API calls and will not wake idle COS bindings.

```bash
curl http://localhost:9999/healthz
# 200: ok
```

### `/readyz` — Readiness

Reports whether GCSFuse is ready to serve requests — i.e., it is live and its recent filesystem error rate is below a configurable threshold.

| Response | Meaning |
|---|---|
| `200 OK` — body `ok` | Mount is live and the error rate is at or below the threshold. |
| `503 Service Unavailable` | Mount is not yet live, the error rate exceeds the threshold, or metrics are unavailable. |

The error rate is computed as `fs_ops_error_count / fs_ops_count` from already-collected Prometheus metrics. No new GCS calls are made.

When no filesystem operations have been recorded yet (e.g., immediately after mount), the error rate is treated as `0` and `/readyz` returns `200`.

```bash
curl http://localhost:9999/readyz
# 200: ok
# 503: error rate 0.1200 exceeds threshold 0.0500
```

### Configuring the Error Rate Threshold

Use `--health-check-error-rate-threshold` to set the fraction of filesystem operations that may fail before `/readyz` returns `503`. The value must be in `[0.0, 1.0]`. The default is `0.05` (5%).

Via flag:
```bash
gcsfuse --prometheus-port=9999 --health-check-error-rate-threshold=0.02 my-bucket /mnt/gcs
```

Via config file:
```yaml
metrics:
  prometheus-port: 9999
  health-check-error-rate-threshold: 0.02
```

### Kubernetes Probe Configuration

The `/healthz` and `/readyz` endpoints map directly to Kubernetes liveness and readiness probes. Example sidecar container configuration:

```yaml
containers:
  - name: gcsfuse
    args:
      - --prometheus-port=9999
      - --health-check-error-rate-threshold=0.05
      - my-bucket
      - /mnt/gcs
    livenessProbe:
      httpGet:
        path: /healthz
        port: 9999
      initialDelaySeconds: 10
      periodSeconds: 30
      failureThreshold: 3
    readinessProbe:
      httpGet:
        path: /readyz
        port: 9999
      initialDelaySeconds: 5
      periodSeconds: 10
      failureThreshold: 3
```

`initialDelaySeconds` for the readiness probe can be kept short because `/readyz` returns `200` (not `503`) when no ops have been recorded yet, so it will not mark the pod unready during startup.
