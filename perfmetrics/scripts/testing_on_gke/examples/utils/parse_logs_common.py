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
from typing import List, Tuple
from utils.gsheet import append_data_to_gsheet, url
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


def download_gcs_objects(src: str, dst: str) -> int:
  print(f"Downloading files from {src} to {os.path.abspath(dst)} ...")
  returncode = run_command(
      " ".join([
          "gcloud",
          "-q",  # ignore prompts
          "storage",
          "cp",
          "-r",
          "--no-user-output-enabled",  # do not print names of objects being copied
          src,
          dst,
      ])
  )
  return returncode


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
  parser.add_argument(
      "--output-gsheet-id",
      metavar="ID of a googlesheet for exporting output to.",
      help=(
          "File path of the output metrics (in CSV format). This is the id in"
          " https://docs.google.com/spreadsheets/d/<id> ."
      ),
      required=True,
      type=str,
  )
  parser.add_argument(
      "--output-worksheet-name",
      metavar=(
          "Name of a worksheet (page) in the googlesheet specified by"
          " --output-gsheet-id"
      ),
      help="File path of the output metrics (in CSV format)",
      required=True,
      type=str,
  )
  parser.add_argument(
      "--output-gsheet-keyfile",
      metavar=(
          "Path of a GCS keyfile for read/write access to output google sheet."
      ),
      help=(
          "For this to work, the google-sheet should be shared with the"
          " client_email/service-account of the keyfile."
      ),
      required=True,
      type=str,
  )

  return parser.parse_args()


def default_service_account_key_file(project_id: str) -> str:
  if project_id == "gcs-fuse-test":
    return "/usr/local/google/home/gargnitin/work/cloud/storage/client/gcsfuse/src/gcsfuse/perfmetrics/scripts/testing_on_gke/examples/20240919-gcs-fuse-test-bc1a2c0aac45.json"
  elif project_id == "gcs-fuse-test-ml":
    return "/usr/local/google/home/gargnitin/work/cloud/storage/client/gcsfuse/src/gcsfuse/perfmetrics/scripts/testing_on_gke/examples/20240919-gcs-fuse-test-ml-d6e0247b2cf1.json"
  else:
    raise Exception(f"Unknown project-id: {project_id}")


def export_to_csv(output_file_path: str, header: str, rows: List):
  if output_file_path and output_file_path.strip():
    ensure_directory_exists(os.path.dirname(output_file_path))
    with open(output_file_path, "a") as output_file_fwr:
      # Write a new header.
      output_file_fwr.write(f"{','.join(header)}\n")
      for row in rows:
        output_file_fwr.write(f"{','.join([f'{val}' for val in row])}\n")
      output_file_fwr.close()
      print(
          "\nSuccessfully published outputs of test runs to"
          f" {output_file_path} !!!"
      )


def export_to_gsheet(
    header: str,
    rows: List,
    output_gsheet_id: str,
    output_worksheet_name: str,
    output_gsheet_keyfile: str,
):
  if (
      output_gsheet_id
      and output_gsheet_id.strip()
      and output_worksheet_name
      and output_worksheet_name.strip()
  ):
    append_data_to_gsheet(
        data={"header": header, "values": rows},
        worksheet=output_worksheet_name,
        gsheet_id=output_gsheet_id,
        serviceAccountKeyFile=output_gsheet_keyfile,
    )
    print(
        "\nSuccessfully published outputs of test runs at worksheet"
        f" '{output_worksheet_name}' in {url(output_gsheet_id)}"
    )
