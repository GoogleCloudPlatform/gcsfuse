#!/usr/bin/env python

# Copyright 2018 The Kubernetes Authors.
# Copyright 2022 Google LLC
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
import json, os, pprint, subprocess
import sys
import dlio_workload

sys.path.append("../")
from utils.utils import get_memory, get_cpu, standard_timestamp, is_mash_installed

LOCAL_LOGS_LOCATION = "../../bin/dlio-logs"

record = {
    "pod_name": "",
    "epoch": 0,
    "scenario": "",
    "train_au_percentage": 0,
    "duration": 0,
    "train_throughput_samples_per_second": 0,
    "train_throughput_mb_per_second": 0,
    "throughput_over_local_ssd": 0,
    "start": "",
    "end": "",
    "highest_memory": 0,
    "lowest_memory": 0,
    "highest_cpu": 0.0,
    "lowest_cpu": 0.0,
    "gcsfuse_mount_options": "",
}


def downloadDlioOutputs(dlioWorkloads):
  for dlioWorkload in dlioWorkloads:
    print(f"Downloading DLIO logs from the bucket {dlioWorkload.bucket}...")
    result = subprocess.run(
        [
            "gsutil",
            "-m",  # download files parallely
            "-q",  # download silently without any logs
            "cp",
            "-r",
            f"gs://{dlioWorkload.bucket}/logs",
            LOCAL_LOGS_LOCATION,
        ],
        capture_output=False,
        text=True,
    )
    if result.returncode < 0:
      print(f"failed to fetch DLIO logs, error: {result.stderr}")


if __name__ == "__main__":
  parser = argparse.ArgumentParser(
      prog="DLIO Unet3d test output parser",
      description=(
          "This program takes in a json test-config file and parses it for"
          " output buckets. From each output bucket, it downloads all the dlio"
          " output logs from gs://<bucket>/logs/ localy to"
          f" {LOCAL_LOGS_LOCATION} and parses them for dlio test runs and their"
          " output metrics."
      ),
  )
  parser.add_argument("--workload-config")
  parser.add_argument(
      "--project-number",
      help=(
          "project-number (e.g. 93817472919) is needed to fetch the cpu/memory"
          " utilization data from GCP."
      ),
  )
  args = parser.parse_args()

  try:
    os.makedirs(LOCAL_LOGS_LOCATION)
  except FileExistsError:
    pass

  dlioWorkloads = dlio_workload.ParseTestConfigForDlioWorkloads(
      args.workload_config
  )
  downloadDlioOutputs(dlioWorkloads)

  """
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
  mash_installed = is_mash_installed()
  if not mash_installed:
    print("Mash is not installed, will skip parsing CPU and memory usage.")

  for root, _, files in os.walk(LOCAL_LOGS_LOCATION):
    if files:
      print(f"Parsing directory {root} ...")
      per_epoch_stats_file = root + "/per_epoch_stats.json"
      summary_file = root + "/summary.json"

      gcsfuse_mount_options = ""
      gcsfuse_mount_options_file = root + "/gcsfuse_mount_options"
      if os.path.isfile(gcsfuse_mount_options_file):
        with open(gcsfuse_mount_options_file) as f:
          gcsfuse_mount_options = f.read().strip()

      with open(per_epoch_stats_file, "r") as f:
        try:
          per_epoch_stats_data = json.load(f)
        except:
          print(f"failed to json-parse {per_epoch_stats_file}")
          continue
      with open(summary_file, "r") as f:
        try:
          summary_data = json.load(f)
        except:
          print(f"failed to json-parse {summary_file}")
          continue

      for i in range(summary_data["epochs"]):
        test_name = summary_data["hostname"]
        part_list = test_name.split("-")
        key = "-".join(part_list[2:5])

        if key not in output:
          output[key] = {
              "num_files_train": part_list[2],
              "mean_file_size": part_list[3],
              "batch_size": part_list[4],
              "records": {
                  "local-ssd": [],
                  "gcsfuse-generic": [],
                  "gcsfuse-file-cache": [],
                  "gcsfuse-no-file-cache": [],
              },
          }

        r = record.copy()
        r["pod_name"] = summary_data["hostname"]
        r["epoch"] = i + 1
        r["scenario"] = "-".join(part_list[5:])
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
        r["start"] = standard_timestamp(
            per_epoch_stats_data[str(i + 1)]["start"]
        )
        r["end"] = standard_timestamp(per_epoch_stats_data[str(i + 1)]["end"])
        if r["scenario"] != "local-ssd" and mash_installed:
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
          pass

        r["gcsfuse_mount_options"] = gcsfuse_mount_options

        pprint.pprint(r)

        while len(output[key]["records"][r["scenario"]]) < i + 1:
          output[key]["records"][r["scenario"]].append({})

        output[key]["records"][r["scenario"]][i] = r

  scenario_order = [
      "local-ssd",
      "gcsfuse-generic",
      "gcsfuse-no-file-cache",
      "gcsfuse-file-cache",
  ]

  output_file = open("./output.csv", "a")
  output_file.write(
      "File Size,File #,Total Size (GB),Batch Size,Scenario,Epoch,Duration"
      " (s),GPU Utilization (%),Throughput (sample/s),Throughput"
      " (MB/s),Throughput over Local SSD (%),GCSFuse Lowest Memory (MB),GCSFuse"
      " Highest Memory (MB),GCSFuse Lowest CPU (core),GCSFuse Highest CPU"
      " (core),Pod,Start,End,GcsfuseMountOptions\n"
  )

  for key in output:
    record_set = output[key]
    total_size = int(
        int(record_set["mean_file_size"])
        * int(record_set["num_files_train"])
        / (1024**3)
    )

    for scenario in scenario_order:
      if scenario not in record_set["records"]:
        print(f"{scenario} not in output so skipping")
        continue
      if "local-ssd" in record_set["records"] and (
          len(record_set["records"]["local-ssd"])
          == len(record_set["records"][scenario])
      ):
        for i in range(len(record_set["records"]["local-ssd"])):
          r = record_set["records"][scenario][i]
          r["throughput_over_local_ssd"] = round(
              r["train_throughput_mb_per_second"]
              / record_set["records"]["local-ssd"][i][
                  "train_throughput_mb_per_second"
              ]
              * 100,
              2,
          )
          output_file.write(
              f"{record_set['mean_file_size']},{record_set['num_files_train']},{total_size},{record_set['batch_size']},{scenario},"
          )
          output_file.write(
              f"{r['epoch']},{r['duration']},{r['train_au_percentage']},{r['train_throughput_samples_per_second']},{r['train_throughput_mb_per_second']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']},\"{r['gcsfuse_mount_options']}\"\n"
          )
      else:
        for i in range(len(record_set["records"][scenario])):
          r = record_set["records"][scenario][i]
          r["throughput_over_local_ssd"] = "NA"
          output_file.write(
              f"{record_set['mean_file_size']},{record_set['num_files_train']},{total_size},{record_set['batch_size']},{scenario},"
          )
          output_file.write(
              f"{r['epoch']},{r['duration']},{r['train_au_percentage']},{r['train_throughput_samples_per_second']},{r['train_throughput_mb_per_second']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']},\"{r['gcsfuse_mount_options']}\"\n"
          )

  output_file.close()
