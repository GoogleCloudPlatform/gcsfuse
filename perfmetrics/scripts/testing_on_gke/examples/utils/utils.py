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
from typing import Tuple


def is_mash_installed() -> bool:
  try:
    subprocess.run(
        ["mash", "--version"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=True,
    )
    return True
  except subprocess.CalledProcessError:
    return False
  except FileNotFoundError:
    return False


def get_memory(
    pod_name: str, start: str, end: str, project_number: int
) -> Tuple[int, int]:
  # for some reason, the mash filter does not always work, so we fetch all the metrics for all the pods and filter later.
  result = subprocess.run(
      [
          "mash",
          "--namespace=cloud_prod",
          "--output=csv",
          (
              "Query(Fetch(Raw('cloud.kubernetes.K8sContainer',"
              " 'kubernetes.io/container/memory/used_bytes'), {'project':"
              f" '{project_number}', 'metric:memory_type': 'non-evictable'}})|"
              " Window(Align('10m'))| GroupBy(['pod_name', 'container_name'],"
              f" Max()), TimeInterval('{start}', '{end}'), '5s')"
          ),
      ],
      capture_output=True,
      text=True,
  )

  data_points_int = []
  data_points_by_pod_container = result.stdout.strip().split("\n")
  for data_points in data_points_by_pod_container[1:]:
    data_points_split = data_points.split(",")
    if len(data_points_split) < 6:
      continue
    pn = data_points_split[4]
    container_name = data_points_split[5]
    if pn == pod_name and container_name == "gke-gcsfuse-sidecar":
      try:
        data_points_int = [int(d) for d in data_points_split[7:]]
      except:
        print(
            f"failed to parse memory for pod {pod_name}, {start}, {end}, data"
            f" {data_points_int}"
        )
      break
  if not data_points_int:
    return 0, 0

  return int(min(data_points_int) / 1024**2), int(
      max(data_points_int) / 1024**2
  )


def get_cpu(
    pod_name: str, start: str, end: str, project_number: int
) -> Tuple[float, float]:
  # for some reason, the mash filter does not always work, so we fetch all the metrics for all the pods and filter later.
  result = subprocess.run(
      [
          "mash",
          "--namespace=cloud_prod",
          "--output=csv",
          (
              "Query(Fetch(Raw('cloud.kubernetes.K8sContainer',"
              " 'kubernetes.io/container/cpu/core_usage_time'), {'project':"
              f" '{project_number}'}})| Window(Rate('10m'))|"
              " GroupBy(['pod_name', 'container_name'], Max()),"
              f" TimeInterval('{start}', '{end}'), '5s')"
          ),
      ],
      capture_output=True,
      text=True,
  )

  data_points_float = []
  data_points_by_pod_container = result.stdout.split("\n")
  for data_points in data_points_by_pod_container[1:]:
    data_points_split = data_points.split(",")
    if len(data_points_split) < 6:
      continue
    pn = data_points_split[4]
    container_name = data_points_split[5]
    if pn == pod_name and container_name == "gke-gcsfuse-sidecar":
      try:
        data_points_float = [float(d) for d in data_points_split[6:]]
      except:
        print(
            f"failed to parse CPU for pod {pod_name}, {start}, {end}, data"
            f" {data_points_float}"
        )

      break

  if not data_points_float:
    return 0.0, 0.0

  return round(min(data_points_float), 5), round(max(data_points_float), 5)


def unix_to_timestamp(unix_timestamp: int) -> str:
  # Convert Unix timestamp to a datetime object (aware of UTC)
  datetime_utc = datetime.datetime.fromtimestamp(
      unix_timestamp / 1000, tz=datetime.timezone.utc
  )

  # Format the datetime object as a string (if desired)
  utc_timestamp_string = datetime_utc.strftime("%Y-%m-%d %H:%M:%S UTC")

  return utc_timestamp_string


def standard_timestamp(timestamp: int) -> str:
  return timestamp.split(".")[0].replace("T", " ") + " UTC"


class UnknownMachineTypeError(Exception):
  """Defines custom exception for unknown machine-type scenario.

  It holds value of machineType as str.
  """

  def __init__(self, message, machineType: str):
    super().__init__(message)
    self.machineType = machineType


def resource_limits(nodeType: str) -> Tuple[dict, dict]:
  if nodeType == "n2-standard-96":
    return {"cpu": 96, "memory": "384Gi"}, {"cpu": 90, "memory": "300Gi"}
  elif nodeType == "n2-standard-48":
    return {"cpu": 48, "memory": "192Gi"}, {"cpu": 45, "memory": "150Gi"}
  elif nodeType == "n2-standard-32":
    return {"cpu": 32, "memory": "128Gi"}, {"cpu": 30, "memory": "100Gi"}
  else:
    raise UnknownMachineTypeError(
        f"Unsupported machine-type: {nodeType}. Unable to decide the"
        " resource-limit for it.",
        nodeType,
    )
