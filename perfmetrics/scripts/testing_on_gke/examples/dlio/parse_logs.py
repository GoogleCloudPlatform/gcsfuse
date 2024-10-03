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

# standard library imports
import argparse
import json
import os
import pprint
import subprocess
import sys
from typing import List, Tuple

# local library imports
sys.path.append("../")
import dlio_workload
from utils.utils import get_memory, get_cpu, unix_to_timestamp, standard_timestamp, is_mash_installed, get_memory_from_monitoring_api, get_cpu_from_monitoring_api, timestamp_to_epoch
from utils.parse_logs_common import ensure_directory_exists, download_gcs_objects, parse_arguments, SUPPORTED_SCENARIOS, default_service_account_key_file, export_to_csv, export_to_gsheet

_LOCAL_LOGS_LOCATION = "../../bin/dlio-logs/logs"

record = {
    "pod_name": "",
    "epoch": 0,
    "scenario": "",
    "train_au_percentage": 0,
    "duration": 0,
    "train_throughput_samples_per_second": 0,
    "train_throughput_mb_per_second": 0,
    "throughput_over_local_ssd": 0,
    "start_epoch": "",
    "end_epoch": "",
    "start": "",
    "end": "",
    "highest_memory": 0,
    "lowest_memory": 0,
    "highest_cpu": 0.0,
    "lowest_cpu": 0.0,
    "gcsfuse_mount_options": "",
}

mash_installed = False

_HEADER = (
    "File Size",
    "File #",
    "Total Size (GB)",
    "Batch Size",
    "Scenario",
    "Epoch",
    "Duration (s)",
    "GPU Utilization (%)",
    "Throughput (sample/s)",
    "Throughput (MB/s)",
    "Throughput over Local SSD (%)",
    "GCSFuse Lowest Memory (MB)",
    "GCSFuse Highest Memory (MB)",
    "GCSFuse Lowest CPU (core)",
    "GCSFuse Highest CPU (core)",
    "Pod-name",
    "Start",
    "End",
    "GcsfuseMountOptions",
    "InstanceID",
)


def downloadDlioOutputs(dlioWorkloads: set, instanceId: str) -> int:
  """Downloads instanceId-specific dlio outputs for each dlioWorkload locally.

  Outputs in the bucket are in the following object naming format
  (details in ./unet3d-loading-test/templates/dlio-tester.yaml).
    gs://<bucket>/logs/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/<scenario>/per_epoch_stats.json
    gs://<bucket>/logs/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/<scenario>/summary.json
    gs://<bucket>/logs/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/gcsfuse-generic/gcsfuse_mount_options

  These are downloaded locally as:
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/<scenario>/per_epoch_stats.json
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/<scenario>/summary.json
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/gcsfuse-generic/gcsfuse_mount_options
  """

  for dlioWorkload in dlioWorkloads:
    srcObjects = f"gs://{dlioWorkload.bucket}/logs/{instanceId}"
    print(f"Downloading DLIO logs from the {srcObjects} ...")
    returncode = download_gcs_objects(srcObjects, _LOCAL_LOGS_LOCATION)
    if returncode < 0:
      print(f"Failed to download DLIO logs from {srcObjects}: {returncode}")
      return returncode
  return 0


def createOutputScenariosFromDownloadedFiles(args: dict) -> dict:
  """Creates output records from the downloaded local files.

  The following creates a dict called 'output'
  from the downloaded dlio output files, which are in the following format.

  <_LOCAL_LOGS_LOCATION>/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/<scenario>/per_epoch_stats.json
  <_LOCAL_LOGS_LOCATION>/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/<scenario>/summary.json
  <_LOCAL_LOGS_LOCATION>/<instanceId>/<numFilesTrain>-<recordLength>-<batchSize>-<hash>/gcsfuse-generic/gcsfuse_mount_options

  Output dict structure:
    "{num_files_train}-{mean_file_size}-{batch_size}":
        "mean_file_size": str
        "num_files_train": str
        "batch_size": str
        "records":
            "local-ssd": [record1, record2, record3, record4]
            "gcsfuse-generic": [record1, record2, record3, record4]
            "gcsfuse-file-cache": [record1, record2, record3, record4]
            "gcsfuse-no-file-cache": [record1, record2, record3, record4]
  """

  output = {}
  for root, _, files in os.walk(_LOCAL_LOGS_LOCATION + "/" + args.instance_id):
    print(f"Parsing directory {root} ...")
    if files:
      # If directory contains gcsfuse_mount_options file, then parse gcsfuse
      # mount options from it in record.
      gcsfuse_mount_options = ""
      gcsfuse_mount_options_file = root + "/gcsfuse_mount_options"
      if os.path.isfile(gcsfuse_mount_options_file):
        with open(gcsfuse_mount_options_file) as f:
          gcsfuse_mount_options = f.read().strip()

      per_epoch_stats_file = root + "/per_epoch_stats.json"
      summary_file = root + "/summary.json"

      # Load per_epoch_stats.json .
      with open(per_epoch_stats_file, "r") as f:
        try:
          per_epoch_stats_data = json.load(f)
        except:
          print(f"failed to json-parse {per_epoch_stats_file}")
          continue

      # Load summary.json .
      with open(summary_file, "r") as f:
        try:
          summary_data = json.load(f)
        except:
          print(f"failed to json-parse {summary_file}")
          continue

      for i in range(summary_data["epochs"]):
        # Get numFilesTrain, recordLength, batchSize from the file/dir path.
        key = root.split("/")[-2]
        key_split = key.split("-")

        if key not in output:
          output[key] = {
              "num_files_train": key_split[-4],
              "mean_file_size": key_split[-3],
              "batch_size": key_split[-2],
              "records": {
                  "local-ssd": [],
                  "gcsfuse-generic": [],
                  "gcsfuse-file-cache": [],
                  "gcsfuse-no-file-cache": [],
              },
          }

        # Create a record for this key.
        r = record.copy()
        r["pod_name"] = summary_data["hostname"]
        r["epoch"] = i + 1
        r["scenario"] = root.split("/")[-1]
        r["train_au_percentage"] = round(
            summary_data["metric"]["train_au_percentage"][i], 2
        )
        r["duration"] = int(float(per_epoch_stats_data[str(i + 1)]["duration"]))
        r["train_throughput_samples_per_second"] = int(
            summary_data["metric"]["train_throughput_samples_per_second"][i]
        )
        r["train_throughput_mb_per_second"] = int(
            r["train_throughput_samples_per_second"]
            * int(output[key]["mean_file_size"])
            / (1024**2)
        )
        r["start_epoch"] = timestamp_to_epoch(
            per_epoch_stats_data[str(i + 1)]["start"]
        )
        r["end_epoch"] = timestamp_to_epoch(
            per_epoch_stats_data[str(i + 1)]["end"]
        )
        r["start"] = standard_timestamp(
            per_epoch_stats_data[str(i + 1)]["start"]
        )
        r["end"] = standard_timestamp(per_epoch_stats_data[str(i + 1)]["end"])

        def fetch_cpu_memory_data():
          if r["scenario"] != "local-ssd":
            if mash_installed:
              r["lowest_memory"], r["highest_memory"] = get_memory(
                  r["pod_name"],
                  r["start"],
                  r["end"],
                  project_number=args.project_number,
              )
              r["lowest_cpu"], r["highest_cpu"] = get_cpu(
                  r["pod_name"],
                  r["start"],
                  r["end"],
                  project_number=args.project_number,
              )
            else:
              r["lowest_memory"], r["highest_memory"] = (
                  get_memory_from_monitoring_api(
                      pod_name=r["pod_name"],
                      start_epoch=r["start_epoch"],
                      end_epoch=r["end_epoch"],
                      project_id=args.project_id,
                      cluster_name=args.cluster_name,
                      namespace_name=args.namespace_name,
                  )
              )
              r["lowest_cpu"], r["highest_cpu"] = get_cpu_from_monitoring_api(
                  pod_name=r["pod_name"],
                  start_epoch=r["start_epoch"],
                  end_epoch=r["end_epoch"],
                  project_id=args.project_id,
                  cluster_name=args.cluster_name,
                  namespace_name=args.namespace_name,
              )

        fetch_cpu_memory_data()

        r["gcsfuse_mount_options"] = gcsfuse_mount_options

        # This print is for debugging in case something goes wrong.
        pprint.pprint(r)

        # If a slot for record for this particular epoch has not been created yet,
        # append enough empty records to make a slot.
        while len(output[key]["records"][r["scenario"]]) < i + 1:
          output[key]["records"][r["scenario"]].append({})

        # Insert the record at the appropriate slot.
        output[key]["records"][r["scenario"]][i] = r

  return output


def writeOutput(
    output: dict,
    args: dict,
):
  rows = []

  for key in output:
    record_set = output[key]
    total_size = int(
        int(record_set["mean_file_size"])
        * int(record_set["num_files_train"])
        / (1024**3)
    )

    for scenario in SUPPORTED_SCENARIOS:
      if scenario not in record_set["records"]:
        print(f"{scenario} not in output so skipping")
        continue

      for i in range(len(record_set["records"][scenario])):
        r = record_set["records"][scenario][i]

        try:
          if "local-ssd" in record_set["records"] and (
              len(record_set["records"]["local-ssd"])
              == len(record_set["records"][scenario])
          ):
            r["throughput_over_local_ssd"] = round(
                r["train_throughput_mb_per_second"]
                / record_set["records"]["local-ssd"][i][
                    "train_throughput_mb_per_second"
                ]
                * 100,
                2,
            )
          else:
            r["throughput_over_local_ssd"] = "NA"

        except ZeroDivisionError:
          print("Got ZeroDivisionError. Ignoring it.")
          r["throughput_over_local_ssd"] = 0

        except Exception as e:
          print(
              "Error: failed to parse/write record-set for"
              f" scenario: {scenario}, i: {i}, record: {r}, exception: {e}"
          )
          continue

        new_row = (
            record_set["mean_file_size"],
            record_set["num_files_train"],
            total_size,
            record_set["batch_size"],
            scenario,
            r["epoch"],
            r["duration"],
            r["train_au_percentage"],
            r["train_throughput_samples_per_second"],
            r["train_throughput_mb_per_second"],
            r["throughput_over_local_ssd"],
            r["lowest_memory"],
            r["highest_memory"],
            r["lowest_cpu"],
            r["highest_cpu"],
            r["pod_name"],
            r["start"],
            r["end"],
            f'"{r["gcsfuse_mount_options"].strip()}"',  # need to wrap in quotes to encapsulate commas in the value.
            args.instance_id,
        )
        rows.append(new_row)

  export_to_csv(output_file_path=args.output_file, header=_HEADER, rows=rows)
  export_to_gsheet(
      output_gsheet_id=args.output_gsheet_id,
      output_worksheet_name=args.output_worksheet_name,
      output_gsheet_keyfile=args.output_gsheet_keyfile,
      header=_HEADER,
      rows=rows,
  )


if __name__ == "__main__":
  args = parse_arguments()
  ensure_directory_exists(_LOCAL_LOGS_LOCATION)

  dlioWorkloads = dlio_workload.ParseTestConfigForDlioWorkloads(
      args.workload_config
  )
  ret = downloadDlioOutputs(dlioWorkloads, args.instance_id)
  if ret != 0:
    print(f"failed to download dlio outputs: {ret}")

  mash_installed = is_mash_installed()
  if not mash_installed:
    print("Mash is not installed, will skip parsing CPU and memory usage.")

  output = createOutputScenariosFromDownloadedFiles(args)
  writeOutput(output, args)
