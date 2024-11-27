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

# local library imports
sys.path.append("../")
import fio_workload
from utils.utils import get_memory, get_cpu, unix_to_timestamp, is_mash_installed, get_memory_from_monitoring_api, get_cpu_from_monitoring_api
from utils.parse_logs_common import ensure_directory_exists, download_gcs_objects, parse_arguments, SUPPORTED_SCENARIOS

_LOCAL_LOGS_LOCATION = "../../bin/fio-logs"

record = {
    "pod_name": "",
    "epoch": 0,
    "scenario": "",
    "duration": 0,
    "IOPS": 0,
    "throughput_mb_per_second": 0,
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
    "blockSize": "",
    "filesPerThread": 0,
    "numThreads": 0,
}


def downloadFioOutputs(fioWorkloads: set, instanceId: str) -> int:
  """Downloads instanceId-specific fio outputs for each fioWorkload locally.

  Outputs in the bucket are in the following object naming format
  (details in ./loading-test/templates/fio-tester.yaml).
    gs://<bucket>/fio-output/<instanceId>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/epoch[N].json
    gs://<bucket>/fio-output/<instanceId>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/pod_name
    gs://<bucket>/fio-output/<instanceId>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/gcsfuse-generic/<readType>/gcsfuse_mount_options

  These are downloaded locally as:
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/epoch[N].json
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/pod_name
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/gcsfuse-generic/<readType>/gcsfuse_mount_options
  """

  for fioWorkload in fioWorkloads:
    dstDir = (
        _LOCAL_LOGS_LOCATION + "/" + instanceId + "/" + fioWorkload.fileSize
    )
    ensure_directory_exists(dstDir)

    srcObjects = f"gs://{fioWorkload.bucket}/fio-output/{instanceId}/*"
    print(f"Downloading FIO outputs from {srcObjects} ...")
    returncode, errorStr = download_gcs_objects(srcObjects, dstDir)
    if returncode < 0:
      print(f"Failed to download FIO outputs from {srcObjects}: {errorStr}")
      return returncode
  return 0


def createOutputScenariosFromDownloadedFiles(args: dict) -> dict:
  """Creates output records from the downloaded local files.

  The following creates a dict called 'output'
  from the downloaded fio output files, which are in the following format.

    <_LOCAL_LOGS_LOCATION>/<instanceId>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/epoch[N].json
    where N=1-#epochs
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/pod_name
    <_LOCAL_LOGS_LOCATION>/<instanceId>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/gcsfuse-generic/<readType>/gcsfuse_mount_options

    Output dict structure:
    "{read_type}-{mean_file_size}-{bs}-{numjobs}-{nrfiles}":
        "mean_file_size": str
        "read_type": str
        "records":
            "local-ssd": [record1, record2, record3, ...]
            "gcsfuse-generic": [record1, record2, record3, ...]
            "gcsfuse-file-cache": [record1, record2, record3, ...]
            "gcsfuse-no-file-cache": [record1, record2, record3, ...]
  """

  output = {}
  for root, _, files in os.walk(_LOCAL_LOGS_LOCATION + "/" + args.instance_id):
    print(f"Parsing directory {root} ...")

    if not files:
      # ignore intermediate directories.
      continue

    # if directory contains gcsfuse_mount_options file, then parse gcsfuse
    # mount options from it in record.
    gcsfuse_mount_options = ""
    gcsfuse_mount_options_file = root + "/gcsfuse_mount_options"
    if os.path.isfile(gcsfuse_mount_options_file):
      with open(gcsfuse_mount_options_file) as f:
        gcsfuse_mount_options = f.read().strip()
        print(f"gcsfuse_mount_options={gcsfuse_mount_options}")

    # if directory has files, it must also contain pod_name file,
    # and we should extract pod-name from it in the record.
    pod_name = ""
    pod_name_file = root + "/pod_name"
    with open(pod_name_file) as f:
      pod_name = f.read().strip()
    print(f"pod_name={pod_name}")

    for file in files:
      # Ignore non-json files to avoid unnecessary failure.
      if not file.endswith(".json"):
        continue

      per_epoch_output = root + f"/{file}"
      with open(per_epoch_output, "r") as f:
        try:
          per_epoch_output_data = json.load(f)
        except:
          print(f"failed to json-parse {per_epoch_output}, so skipping it.")
          continue

      # Confirm that the per_epoch_output_data has ["jobs"][0]["job options"]['"bs"]
      # for determining blocksize.
      if (
          not "jobs" in per_epoch_output_data
          or not per_epoch_output_data["jobs"]
          or not "job options" in per_epoch_output_data["jobs"][0]
          or not "bs" in per_epoch_output_data["jobs"][0]["job options"]
      ):
        print(
            'Did not find "[jobs][0][job options][bs]" in'
            f" {per_epoch_output}, so ignoring this file"
        )
        continue
      # Confirm that the per_epoch_output_data has ["global options"] for
      # determining nrfiles and numjobs in it.
      if "global options" not in per_epoch_output_data:
        print(f"field: 'global options' missing in {per_epoch_output}")
        continue

      # This print is for debugging in case something goes wrong.
      print(f"Now parsing file {per_epoch_output} ...")

      # Get fileSize, readType, echo number from the file path.
      root_split = root.split("/")
      mean_file_size = root_split[-4]
      key = root_split[
          -3
      ]  # key is unique for a given combination of of fileSize,blockSize,numThreads(numjobs),filesPerThread(nrfiles).
      scenario = root_split[-2]
      read_type = root_split[-1]
      epoch = int(file.split(".")[0][-1])

      # Get nrfiles,numjobs, blocksize from ["global options"] and ["job options"].
      global_options = per_epoch_output_data["global options"]
      nrfiles = int(global_options["nrfiles"])
      numjobs = int(global_options["numjobs"])
      bs = per_epoch_output_data["jobs"][0]["job options"]["bs"]

      # If the record for this key has not been added, create a new entry
      # for it.
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

      # Create a record for this key.
      r = record.copy()
      bs = per_epoch_output_data["jobs"][0]["job options"]["bs"]
      r["pod_name"] = pod_name
      r["epoch"] = epoch
      r["scenario"] = scenario
      r["duration"] = int(
          per_epoch_output_data["jobs"][0]["read"]["runtime"] / 1000
      )
      r["IOPS"] = int(per_epoch_output_data["jobs"][0]["read"]["iops"])
      r["throughput_mb_per_second"] = int(
          per_epoch_output_data["jobs"][0]["read"]["bw_bytes"] / (1024**2)
      )
      r["start_epoch"] = per_epoch_output_data["jobs"][0]["job_start"] // 1000
      r["end_epoch"] = per_epoch_output_data["timestamp_ms"] // 1000
      r["start"] = unix_to_timestamp(
          per_epoch_output_data["jobs"][0]["job_start"]
      )
      r["end"] = unix_to_timestamp(per_epoch_output_data["timestamp_ms"])

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

      r["gcsfuse_mount_options"] = gcsfuse_mount_options
      r["blockSize"] = bs
      r["filesPerThread"] = nrfiles
      r["numThreads"] = numjobs

      # This print is for debugging in case something goes wrong.
      pprint.pprint(r)

      # If a slot for record for this particular epoch has not been created yet,
      # append enough empty records to make a slot.
      while len(output[key]["records"][scenario]) < epoch:
        output[key]["records"][scenario].append({})

      # Insert the record at the appropriate slot.
      output[key]["records"][scenario][epoch - 1] = r

  return output


def writeRecordsToCsvOutputFile(output: dict, output_file_path: str):
  with open(output_file_path, "a") as output_file_fwr:
    # Write a new header.
    output_file_fwr.write(
        "File Size,Read Type,Scenario,Epoch,Duration"
        " (s),Throughput (MB/s),IOPS,Throughput over Local SSD (%),GCSFuse"
        " Lowest"
        " Memory (MB),GCSFuse Highest Memory (MB),GCSFuse Lowest CPU"
        " (core),GCSFuse Highest CPU"
        " (core),Pod,Start,End,GcsfuseMoutOptions,BlockSize,FilesPerThread,NumThreads,InstanceID\n"
    )

    for key in output:
      record_set = output[key]

      for scenario in record_set["records"]:
        if scenario not in SUPPORTED_SCENARIOS:
          print(f"Unknown scenario: {scenario}. Ignoring it...")
          continue

        for i in range(len(record_set["records"][scenario])):
          r = record_set["records"][scenario][i]

          try:
            if ("local-ssd" in record_set["records"]) and (
                len(record_set["records"]["local-ssd"])
                == len(record_set["records"][scenario])
            ):
              r["throughput_over_local_ssd"] = round(
                  r["throughput_mb_per_second"]
                  / record_set["records"]["local-ssd"][i][
                      "throughput_mb_per_second"
                  ]
                  * 100,
                  2,
              )
            else:
              r["throughput_over_local_ssd"] = "NA"

          except Exception as e:
            print(
                "Error: failed to parse/write record-set for"
                f" scenario: {scenario}, i: {i}, record: {r}, exception: {e}"
            )
            continue

          output_file_fwr.write(
              f"{record_set['mean_file_size']},{record_set['read_type']},{scenario},{r['epoch']},{r['duration']},{r['throughput_mb_per_second']},{r['IOPS']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']},\"{r['gcsfuse_mount_options']}\",{r['blockSize']},{r['filesPerThread']},{r['numThreads']},{args.instance_id}\n"
          )

    output_file_fwr.close()


if __name__ == "__main__":
  args = parse_arguments()
  ensure_directory_exists(_LOCAL_LOGS_LOCATION)

  fioWorkloads = fio_workload.ParseTestConfigForFioWorkloads(
      args.workload_config
  )
  downloadFioOutputs(fioWorkloads, args.instance_id)

  mash_installed = is_mash_installed()
  if not mash_installed:
    print("Mash is not installed, will skip parsing CPU and memory usage.")

  output = createOutputScenariosFromDownloadedFiles(args)

  output_file_path = args.output_file
  # Create the parent directory of output_file_path if doesn't
  # exist already.
  ensure_directory_exists(os.path.dirname(output_file_path))
  writeRecordsToCsvOutputFile(output, output_file_path)
  print(
      "\n\nSuccessfully published outputs of FIO test runs to"
      f" {output_file_path} !!!\n\n"
  )
