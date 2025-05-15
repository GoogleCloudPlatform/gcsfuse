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

import argparse
import os
import subprocess
from typing import Tuple
from utils.utils import get_cpu_from_monitoring_api, get_memory_from_monitoring_api

SUPPORTED_SCENARIOS = [
    "local-ssd",
    "gcsfuse-generic",
]


def ensure_directory_exists(dirpath: str):
  try:
    os.makedirs(dirpath)
  except FileExistsError:
    pass


def download_gcs_objects(src: str, dst: str) -> Tuple[int, str]:
  result = subprocess.run(
      [
          "gcloud",
          "-q",  # ignore prompts
          "storage",
          "cp",
          "-r",
          "--no-user-output-enabled",  # do not print names of objects being copied
          src,
          dst,
      ],
      capture_output=False,
      text=True,
  )
  if result.returncode < 0:
    return (result.returncode, f"error: {result.stderr}")
  return result.returncode, ""


# Common argument parser for both fio and dlio
# output parsers.
def parse_arguments(
    fio_or_dlio: str = "DLIO", add_bq_support: bool = False
) -> object:
  parser = argparse.ArgumentParser(
      prog=f"{fio_or_dlio} output parser",
      description=(
          "This program takes in a json workload configuration file and parses"
          " it for valid FIO workloads and the locations of their test outputs"
          " on GCS. It downloads each such output object locally to"
          " {_LOCAL_LOGS_LOCATION} and parses them for FIO test runs, and then"
          " dumps their output metrics into a CSV report file."
      ),
  )
  parser.add_argument(
      "--workload-config",
      help=(
          "A json configuration file to define workloads that were run to"
          " generate the outputs that should be parsed."
      ),
      required=True,
  )
  parser.add_argument(
      "--project-id",
      metavar="GCP Project ID/name",
      help=(
          "project-id (e.g. gcs-fuse-test) is needed to fetch the cpu/memory"
          " utilization data from GCP."
      ),
      required=True,
  )
  parser.add_argument(
      "--project-number",
      metavar="GCP Project Number",
      help=(
          "project-number (e.g. 93817472919) is needed to fetch the cpu/memory"
          " utilization data from GCP."
      ),
      required=True,
  )
  parser.add_argument(
      "--experiment-id",
      help="unique string ID for current test-run",
      required=True,
  )
  parser.add_argument(
      "--cluster-name",
      help="Name of GKE cluster where the current test was run",
      required=True,
  )
  parser.add_argument(
      "--namespace-name",
      help="kubernestes namespace used for the current test-run",
      required=True,
  )
  parser.add_argument(
      "-o",
      "--output-file",
      metavar="Output file (CSV) path",
      help="File path of the output metrics (in CSV format)",
      default="output.csv",
  )
  parser.add_argument(
      "--predownloaded-output-files",
      help="If true, output files will not be downloaded. False by default.",
      required=False,
      default=False,
      action="store_true",
  )
  if add_bq_support:
    parser.add_argument(
        "--bq-project-id",
        metavar="GCP Project ID/name",
        help="Bigquery project ID",
        required=False,
    )
    parser.add_argument(
        "--bq-dataset-id",
        help="Bigquery dataset id",
        required=False,
    )
    parser.add_argument(
        "--bq-table-id",
        help="Bigquery table name",
        required=False,
    )

  return parser.parse_args()


def fetch_cpu_memory_data(args, record):
  if record["scenario"] != "local-ssd":
    record["lowest_memory"], record["highest_memory"] = (
        get_memory_from_monitoring_api(
            pod_name=record["pod_name"],
            start_epoch=record["start_epoch"],
            end_epoch=record["end_epoch"],
            project_id=args.project_id,
            cluster_name=args.cluster_name,
            namespace_name=args.namespace_name,
        )
    )
    record["lowest_cpu"], record["highest_cpu"] = get_cpu_from_monitoring_api(
        pod_name=record["pod_name"],
        start_epoch=record["start_epoch"],
        end_epoch=record["end_epoch"],
        project_id=args.project_id,
        cluster_name=args.cluster_name,
        namespace_name=args.namespace_name,
    )
