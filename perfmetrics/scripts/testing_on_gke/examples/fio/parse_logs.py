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

import json, os, pprint, subprocess
import sys

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
}

if __name__ == "__main__":
  logLocations = [
      ("gke-fio-64k-1m", "64K"),
      ("gke-fio-128k-1m", "128K"),
      ("gke-fio-1mb-1m", "1M"),
      ("gke-fio-100mb-50k", "100M"),
      ("gke-fio-200gb-1", "200G"),
  ]

  try:
    os.makedirs(LOCAL_LOGS_LOCATION)
  except FileExistsError:
    pass

  for folder, fileSize in logLocations:
    try:
      os.makedirs(LOCAL_LOGS_LOCATION + "/" + fileSize)
    except FileExistsError:
      pass
    print(f"Download FIO output from the folder {folder}...")
    result = subprocess.run(
        [
            "gsutil",
            "-m",
            "cp",
            "-r",
            f"gs://{folder}/fio-output",
            LOCAL_LOGS_LOCATION + "/" + fileSize,
        ],
        capture_output=False,
        text=True,
    )
    if result.returncode < 0:
      print(f"failed to fetch FIO output, error: {result.stderr}")

  """
    "{read_type}-{mean_file_size}":
        "mean_file_size": str
        "read_type": str
        "records":
            "local-ssd": [record1, record2, record3, record4]
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
      root_split = root.split("/")
      mean_file_size = root_split[-4]
      scenario = root_split[-2]
      read_type = root_split[-1]
      epoch = int(file.split(".")[0][-1])

      with open(per_epoch_output, "r") as f:
        per_epoch_output_data = json.load(f)

      key = "-".join([read_type, mean_file_size])
      if key not in output:
        output[key] = {
            "mean_file_size": mean_file_size,
            "read_type": read_type,
            "records": {
                "local-ssd": [],
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
            r["pod_name"], r["start"], r["end"]
        )
        r["lowest_cpu"], r["highest_cpu"] = get_cpu(
            r["pod_name"], r["start"], r["end"]
        )

      pprint.pprint(r)

      while len(output[key]["records"][scenario]) < epoch:
        output[key]["records"][scenario].append({})

      output[key]["records"][scenario][epoch - 1] = r

  output_order = [
      "read-64K",
      "read-128K",
      "read-1M",
      "read-100M",
      "read-200G",
      "randread-1M",
      "randread-100M",
      "randread-200G",
  ]
  scenario_order = ["local-ssd", "gcsfuse-no-file-cache", "gcsfuse-file-cache"]

  output_file = open("./output.csv", "a")
  output_file.write(
      "File Size,Read Type,Scenario,Epoch,Duration (s),Throughput"
      " (MB/s),IOPS,Throughput over Local SSD (%),GCSFuse Lowest Memory"
      " (MB),GCSFuse Highest Memory (MB),GCSFuse Lowest CPU (core),GCSFuse"
      " Highest CPU (core),Pod,Start,End\n"
  )

  for key in output_order:
    if key not in output:
      continue
    record_set = output[key]

    for scenario in scenario_order:
      for i in range(len(record_set["records"][scenario])):
        r = record_set["records"][scenario][i]
        r["throughput_over_local_ssd"] = round(
            r["throughput_mb_per_second"]
            / record_set["records"]["local-ssd"][i]["throughput_mb_per_second"]
            * 100,
            2,
        )
        output_file.write(
            f"{record_set['mean_file_size']},{record_set['read_type']},{scenario},{r['epoch']},{r['duration']},{r['throughput_mb_per_second']},{r['IOPS']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']}\n"
        )

  output_file.close()
