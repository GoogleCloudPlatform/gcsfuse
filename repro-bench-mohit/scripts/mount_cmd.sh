#!/bin/bash
gcsfuse \
  --cache-dir=/mnt/disks/lssd \
  --config-file=gcsfuse_config.yaml \
  repro-bench-mohit /mnt/gcs
