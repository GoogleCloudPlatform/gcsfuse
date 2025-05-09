#!/usr/bin/env python

# Copyright 2018 The Kubernetes Authors.
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import datetime, subprocess
import math
import time
from typing import Tuple
from google.cloud import monitoring_v3

_GCSFUSE_CONTAINER_NAME = "gke-gcsfuse-sidecar"


def unix_to_timestamp(unix_timestamp: int) -> str:
  # Convert Unix timestamp to a datetime object (aware of UTC)
  datetime_utc = datetime.datetime.fromtimestamp(
      unix_timestamp / 1000, tz=datetime.timezone.utc
  )

  # Format the datetime object as a string (if desired)
  utc_timestamp_string = datetime_utc.strftime("%Y-%m-%d %H:%M:%S UTC")

  return utc_timestamp_string


def standard_timestamp(timestamp: str) -> str:
  return timestamp.split(".")[0].replace("T", " ") + " UTC"


def timestamp_to_epoch(timestamp: str) -> int:
  try:
    return int(
        datetime.datetime.strptime(
            timestamp, "%Y-%m-%dT%H:%M:%S.%f"
        ).timestamp()
    )
  except ValueError:
    return int(
        datetime.datetime.strptime(timestamp, "%Y-%m-%dT%H:%M:%S").timestamp()
    )


class UnknownMachineTypeError(Exception):
  """Defines custom exception for unknown machine-type scenario.

  It holds value of machineType as str.
  """

  def __init__(self, message, machineType: str):
    super().__init__(message)
    self.machineType = machineType


def resource_limits(nodeType: str) -> Tuple[dict, dict]:
  """Returns resource limits and requests for cpu/memory for different machine types."""
  if nodeType == "n2-standard-96":
    return {"cpu": 96, "memory": "384Gi"}, {"cpu": 90, "memory": "300Gi"}
  elif nodeType == "n2-standard-48":
    return {"cpu": 48, "memory": "192Gi"}, {"cpu": 45, "memory": "150Gi"}
  elif nodeType == "n2-standard-32":
    return {"cpu": 32, "memory": "128Gi"}, {"cpu": 30, "memory": "100Gi"}
  elif nodeType == "c3-standard-176" or nodeType == "c3-standard-176-lssd":
    return {"cpu": 176, "memory": "704Gi"}, {"cpu": 100, "memory": "400Gi"}
  else:
    raise UnknownMachineTypeError(
        f"Unknown machine-type: {nodeType}. Unable to decide the"
        " resource-limits for it.",
        nodeType,
    )


def _is_relevant_monitoring_result(
    result,
    cluster_name: str,
    pod_name: str,
    namespace_name: str,
) -> bool:
  return (
      True
      if (
          hasattr(result, "resource")
          and hasattr(result.resource, "type")
          and result.resource.type == "k8s_container"
          and hasattr(result.resource, "labels")
          and "cluster_name" in result.resource.labels
          and result.resource.labels["cluster_name"] == cluster_name
          and "pod_name" in result.resource.labels
          and result.resource.labels["pod_name"] == pod_name
          and "container_name" in result.resource.labels
          and result.resource.labels["container_name"]
          == _GCSFUSE_CONTAINER_NAME
          and "namespace_name" in result.resource.labels
          and result.resource.labels["namespace_name"] == namespace_name
          and hasattr(result, "points")
      )
      else False
  )


def get_memory_from_monitoring_api(
    project_id: str,
    cluster_name: str,
    pod_name: str,
    namespace_name: str,
    start_epoch: int,
    end_epoch: int,
) -> Tuple[int, int]:
  """Returns min,max memory usage of the given gke-cluster/namespace/pod/container/start/end scenario in MiB ."""
  client = monitoring_v3.MetricServiceClient()
  project_name = f"projects/{project_id}"

  interval = monitoring_v3.TimeInterval({
      "start_time": {"seconds": start_epoch, "nanos": 0},
      "end_time": {"seconds": end_epoch, "nanos": 0},
  })
  aggregation = monitoring_v3.Aggregation({
      "alignment_period": {"seconds": 600},  # 10 minutes
      "per_series_aligner": monitoring_v3.Aggregation.Aligner.ALIGN_MAX,
  })

  results = client.list_time_series(
      request={
          "name": project_name,
          "filter": (
              'metric.type = "kubernetes.io/container/memory/used_bytes"'
              ' AND metric.labels.memory_type = "non-evictable"'
              f" AND resource.labels.cluster_name = {cluster_name}"
              f" AND resource.labels.pod_name = {pod_name}"
              f" AND resource.labels.container_name = {_GCSFUSE_CONTAINER_NAME}"
              f" AND resource.labels.namespace_name = {namespace_name}"
          ),
          "interval": interval,
          "view": monitoring_v3.ListTimeSeriesRequest.TimeSeriesView.FULL,
          "aggregation": aggregation,
      }
  )

  relevant_results = [
      result
      for result in results
      if _is_relevant_monitoring_result(
          result,
          cluster_name,
          pod_name,
          namespace_name,
      )
  ]
  if len(relevant_results) > 0:
    return round(
        min(
            min(
                (point.value.int64_value if point.value.int64_value >= 0 else 0)
                for point in result.points
            )
            for result in relevant_results
        )
        / 2**20,  # convert to MiB/s
        0,  # round to integer.
    ), round(
        max(
            max(
                (point.value.int64_value if point.value.int64_value > 0 else 0)
                for point in result.points
            )
            for result in relevant_results
        )
        / 2**20,  # convert to MiB/s
        0,  # round to integer.
    )
  else:
    print(
        f"Warning: No memory data found for epoch in time-range [{start_epoch},"
        f" {end_epoch}) in namespace={namespace_name}, cluster={cluster_name},"
        f" pod={pod_name}, so marking -1,-1 for"
        " it !"
    )
    return -1, -1


def get_cpu_from_monitoring_api(
    project_id: str,
    cluster_name: str,
    pod_name: str,
    namespace_name: str,
    start_epoch: int,
    end_epoch: int,
) -> Tuple[float, float]:
  """Returns min,max cpu usage of the given gke-cluster/namespace/pod/container/start/end scenario."""
  client = monitoring_v3.MetricServiceClient()
  project_name = f"projects/{project_id}"

  interval = monitoring_v3.TimeInterval({
      "start_time": {"seconds": start_epoch, "nanos": 0},
      "end_time": {"seconds": end_epoch, "nanos": 0},
  })
  aggregation = monitoring_v3.Aggregation({
      "alignment_period": {"seconds": 600},  # 10 minutes
      "per_series_aligner": monitoring_v3.Aggregation.Aligner.ALIGN_RATE,
  })

  results = client.list_time_series(
      request={
          "name": project_name,
          "filter": (
              'metric.type = "kubernetes.io/container/cpu/core_usage_time"'
              f" AND resource.labels.cluster_name = {cluster_name}"
              f" AND resource.labels.pod_name = {pod_name}"
              f" AND resource.labels.container_name = {_GCSFUSE_CONTAINER_NAME}"
              f" AND resource.labels.namespace_name = {namespace_name}"
          ),
          "interval": interval,
          "view": monitoring_v3.ListTimeSeriesRequest.TimeSeriesView.FULL,
          "aggregation": aggregation,
      }
  )

  relevant_results = [
      result
      for result in results
      if _is_relevant_monitoring_result(
          result,
          cluster_name,
          pod_name,
          namespace_name,
      )
  ]
  if len(relevant_results) > 0:
    return round(
        min(
            min(
                (
                    point.value.double_value
                    if point.value.double_value != math.nan
                    else 0
                )
                for point in result.points
            )
            for result in relevant_results
        ),
        5,  # round up to 5 decimal places.
    ), round(
        max(
            max(
                (
                    point.value.double_value
                    if point.value.double_value != math.nan
                    else 0
                )
                for point in result.points
            )
            for result in relevant_results
        ),
        5,  # round up to 5 decimal places.
    )
  else:
    print(
        f"Warning: No cpu data found for epoch in time-range [{start_epoch},"
        f" {end_epoch}) in namespace={namespace_name}, cluster={cluster_name},"
        f" pod={pod_name}, so marking -1,-1 for"
        " it !"
    )
    return -1, -1


def convert_size_to_bytes(size_in_string: str) -> int:
  """Converts string-form numbers like 1k, 1m, 1g, 1K, 1M, 1G etc.

  to normal numbers i.e. 10^3, 10^6, 10^9, 2^10, 2^20, 2^30 respectively.

  The conversion is based on how FIO represents sizes e.g. small codes i.e.
  k,m,g for powers of 10, and capital codes i.e. K,M,G for powers of 1024.

  Arguments:

  size_in_string: strings like '1k', '1m', '1g', '1K', '1M', '1G' etc.

  Returns:
  1000, 1000000, 1000000000 for input '1k','1m','1g' etc. respectively.
  1024, 1048576 for input '1K', '1M' etc. respectively.
  """
  if size_in_string is None:
    return 0

  # Remove all leading and trailing spaces.
  size_in_string = size_in_string.strip()
  if len(size_in_string) == 0:
    return 0
  multiplier_char = size_in_string[-1]
  multiplier_map = {
      "k": 10**3,
      "m": 10**6,
      "g": 10**9,
      "K": 2**10,
      "M": 2**20,
      "G": 2**30,
  }
  if multiplier_char in multiplier_map:
    multiplier = multiplier_map[multiplier_char]
    base_value = 1 if len(size_in_string) == 1 else int(size_in_string[:-1])
    return base_value * multiplier
  else:
    return int(size_in_string)
