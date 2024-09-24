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
from utils.utils import run_command

SUPPORTED_SCENARIOS = [
    "local-ssd",
    "gcsfuse-generic",
    "gcsfuse-no-file-cache",
    "gcsfuse-file-cache",
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


def parse_arguments() -> object:
  parser = argparse.ArgumentParser(
      prog="DLIO Unet3d test output parser",
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
      "--instance-id",
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
  return parser.parse_args()
