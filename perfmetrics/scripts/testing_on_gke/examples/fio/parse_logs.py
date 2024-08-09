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
import fio_workload

sys.path.append("../")
from utils.utils import get_memory, get_cpu, unix_to_timestamp, is_mash_installed

LOCAL_LOGS_LOCATION = "../../bin/fio-logs"

record = {
    "pod_name": "",
    "epoch": 0,
    "scenario": "",
    "duration": 0,
    "IOPS": 0,
    "throughput_mb_per_second": 0,
    "throughput_over_local_ssd": 0,
    "start": "",
    "end": "",
    "highest_memory": 0,
    "lowest_memory": 0,
    "highest_cpu": 0.0,
    "lowest_cpu": 0.0,
    "gcsfuse_mount_options": "",
    "blockSize": "",
    "filesPerThread": 0,
    "numThreads": 0,
}


def downloadFioOutputs(fioWorkloads):
  for fioWorkload in fioWorkloads:
    try:
      os.makedirs(LOCAL_LOGS_LOCATION + "/" + fioWorkload.fileSize)
    except FileExistsError:
      pass

    print(f"Downloading FIO outputs from {fioWorkload.bucket}...")
    result = subprocess.run(
        [
            "gsutil",
            "-m",  # download multiple files parallelly
            "-q",  # download silently without any logs
            "cp",
            "-r",
            f"gs://{fioWorkload.bucket}/fio-output",
            LOCAL_LOGS_LOCATION + "/" + fioWorkload.fileSize,
        ],
        capture_output=False,
        text=True,
    )
    if result.returncode < 0:
      print(f"failed to fetch FIO output, error: {result.stderr}")


if __name__ == "__main__":
  parser = argparse.ArgumentParser(
      prog="DLIO Unet3d test output parser",
      description=(
          "This program takes in a json test-config file and parses it for"
          " output buckets. From each output bucket, it downloads all the FIO"
          " output logs from gs://<bucket>/logs/ locally to"
          f" {LOCAL_LOGS_LOCATION} and parses them for FIO test runs and their"
          " output metrics."
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
      "--project-number",
      help=(
          "project-number (e.g. 93817472919) is needed to fetch the cpu/memory"
          " utilization data from GCP."
      ),
      required=True,
  )
  args = parser.parse_args()

  try:
    os.makedirs(LOCAL_LOGS_LOCATION)
  except FileExistsError:
    pass

  fioWorkloads = fio_workload.ParseTestConfigForFioWorkloads(
      args.workload_config
  )
  downloadFioOutputs(fioWorkloads)

  """
    "{read_type}-{mean_file_size}":
        "mean_file_size": str
        "read_type": str
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
    for file in files:
      per_epoch_output = root + f"/{file}"
      if not per_epoch_output.endswith(".json"):
        print(f"ignoring file {per_epoch_output} as it's not a json file")
        continue

      gcsfuse_mount_options = ""
      gcsfuse_mount_options_file = root + "/gcsfuse_mount_options"
      if os.path.isfile(gcsfuse_mount_options_file):
        with open(gcsfuse_mount_options_file) as f:
          gcsfuse_mount_options = f.read().strip()

      print(f"Now parsing file {per_epoch_output} ...")
      root_split = root.split("/")
      mean_file_size = root_split[-4]
      scenario = root_split[-2]
      read_type = root_split[-1]
      epoch = int(file.split(".")[0][-1])

      with open(per_epoch_output, "r") as f:
        try:
          per_epoch_output_data = json.load(f)
        except:
          print(f"failed to json-parse {per_epoch_output}, so skipping it.")
          continue

      if "global options" not in per_epoch_output_data:
        print(f"field: 'global options' missing in {per_epoch_output}")
        continue
      global_options = per_epoch_output_data["global options"]
      nrfiles = int(global_options["nrfiles"])
      numjobs = int(global_options["numjobs"])

      key = "-".join([read_type, mean_file_size])
      if key not in output:
        output[key] = {
            "mean_file_size": mean_file_size,
            "read_type": read_type,
            "records": {
                "local-ssd": [],
                "gcsfuse-generic": [],
                "gcsfuse-file-cache": [],
                "gcsfuse-no-file-cache": [],
            },
        }

      r = record.copy()
      bs = per_epoch_output_data["jobs"][0]["job options"]["bs"]
      r["pod_name"] = (
          f"fio-tester-{read_type}-{mean_file_size.lower()}-{bs.lower()}-{scenario}"
      )
      r["epoch"] = epoch
      r["scenario"] = scenario
      r["duration"] = int(
          per_epoch_output_data["jobs"][0]["read"]["runtime"] / 1000
      )
      r["IOPS"] = int(per_epoch_output_data["jobs"][0]["read"]["iops"])
      r["throughput_mb_per_second"] = int(
          per_epoch_output_data["jobs"][0]["read"]["bw_bytes"] / (1024**2)
      )
      r["start"] = unix_to_timestamp(
          per_epoch_output_data["jobs"][0]["job_start"]
      )
      r["end"] = unix_to_timestamp(per_epoch_output_data["timestamp_ms"])
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
      r["blockSize"] = bs
      r["filesPerThread"] = nrfiles
      r["numThreads"] = numjobs

      pprint.pprint(r)

      while len(output[key]["records"][scenario]) < epoch:
        output[key]["records"][scenario].append({})

      output[key]["records"][scenario][epoch - 1] = r

  scenario_order = [
      "local-ssd",
      "gcsfuse-generic",
      "gcsfuse-no-file-cache",
      "gcsfuse-file-cache",
  ]

  output_file = open("./output.csv", "a")
  output_file.write(
      "File Size,Read Type,Scenario,Epoch,Duration"
      " (s),Throughput (MB/s),IOPS,Throughput over Local SSD (%),GCSFuse Lowest"
      " Memory (MB),GCSFuse Highest Memory (MB),GCSFuse Lowest CPU"
      " (core),GCSFuse Highest CPU"
      " (core),Pod,Start,End,GcsfuseMoutOptions,BlockSize,FilesPerThread,NumThreads\n"
  )

  for key in output:
    record_set = output[key]

    for scenario in scenario_order:
      for i in range(len(record_set["records"][scenario])):
        if ("local-ssd" in record_set["records"]) and (
            len(record_set["records"]["local-ssd"])
            == len(record_set["records"][scenario])
        ):
          try:
            r = record_set["records"][scenario][i]
            r["throughput_over_local_ssd"] = round(
                r["throughput_mb_per_second"]
                / record_set["records"]["local-ssd"][i][
                    "throughput_mb_per_second"
                ]
                * 100,
                2,
            )
          except:
            print(
                "failed to parse record-set for throughput_over_local_ssd."
                f" record: {r}"
            )
            continue
          else:
            output_file.write(
                f"{record_set['mean_file_size']},{record_set['read_type']},{scenario},{r['epoch']},{r['duration']},{r['throughput_mb_per_second']},{r['IOPS']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']},\"{r['gcsfuse_mount_options']}\",{r['blockSize']},{r['filesPerThread']},{r['numThreads']}\n"
            )
        else:
          try:
            r = record_set["records"][scenario][i]
            r["throughput_over_local_ssd"] = "NA"
          except:
            print(
                "failed to parse record-set for throughput_over_local_ssd."
                f" record: {r}"
            )
            continue
          else:
            output_file.write(
                f"{record_set['mean_file_size']},{record_set['read_type']},{scenario},'Unknown',{r['epoch']},{r['duration']},{r['throughput_mb_per_second']},{r['IOPS']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']},\"{r['gcsfuse_mount_options']}\",{r['blockSize']},{r['filesPerThread']},{r['numThreads']}\n"
            )
  output_file.close()
