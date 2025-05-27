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
from collections.abc import Sequence
import json
import os
from os import path
import pprint
import re
import subprocess
import sys

# local library imports
sys.path.append("../")
import fio_workload
from utils.parse_logs_common import ensure_directory_exists, download_gcs_objects, parse_arguments, SUPPORTED_SCENARIOS, fetch_cpu_memory_data
from fio.bq_utils import FioBigqueryExporter, FioTableRow, Timestamp
from utils.utils import unix_to_timestamp, convert_size_to_bytes

_LOCAL_LOGS_LOCATION = "../../bin/fio-logs"
_EPOCH_FILENAME_REGEX = "^epoch[1-9][0-9]*.json$"
_EPOCH_NUMBER_MATCH_REGEX = ".*epoch([1-9][0-9]*).json"

record = {
    "pod_name": "",
    "epoch": 0,
    "scenario": "",
    "duration": 0,
    "IOPS": 0,
    "throughput_mib_per_second": 0.0,  # in 1024^2 bytes/second
    "throughput_bytes_per_second": 0.0,
    "throughput_mb_per_second": 0.0,  # output metric in 10^6 bytes/second
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
    "e2e_latency_ns_max": 0,
    "e2e_latency_ns_p50": 0,
    "e2e_latency_ns_p90": 0,
    "e2e_latency_ns_p99": 0,
    "e2e_latency_ns_p99.9": 0,
    "machine_type": "",
    "bucket_name": "",
}


def download_fio_outputs(fioWorkloads: set, experimentID: str) -> int:
  """Downloads experimentID-specific fio outputs for each fioWorkload locally.

  Outputs in the bucket are in the following object naming format
  (details in ./loading-test/templates/fio-tester.yaml).
    gs://<bucket>/fio-output/<experimentID>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/epoch[N].json
    gs://<bucket>/fio-output/<experimentID>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/pod_name
    gs://<bucket>/fio-output/<experimentID>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/gcsfuse-generic/<readType>/gcsfuse_mount_options

  These are downloaded locally as:
    <_LOCAL_LOGS_LOCATION>/<experimentID>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/epoch[N].json
    <_LOCAL_LOGS_LOCATION>/<experimentID>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/pod_name
    <_LOCAL_LOGS_LOCATION>/<experimentID>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/gcsfuse-generic/<readType>/gcsfuse_mount_options
  """

  # Find all the unique set of buckets, and then only
  # download output files from them to avoid runtime
  # wastage for multiple workloads using the same bucket.
  buckets = set()
  for fioWorkload in fioWorkloads:
    buckets.add(fioWorkload.bucket)

  for bucket in buckets:
    dstDir = path.join(_LOCAL_LOGS_LOCATION, experimentID, bucket)
    ensure_directory_exists(dstDir)

    srcObjects = f"gs://{bucket}/fio-output/{experimentID}/*"
    print(f"Downloading FIO outputs from {srcObjects} ...")
    returncode, errorStr = download_gcs_objects(srcObjects, dstDir)
    if returncode < 0:
      print(f"Failed to download FIO outputs from {srcObjects}: {errorStr}")
      return returncode
  return 0


def create_output_scenarios_from_downloaded_files(args: dict) -> dict:
  """Creates output records from the downloaded local files.

  The following creates a dict called 'output'
  from the downloaded fio output files, which are in the following format.

    <_LOCAL_LOGS_LOCATION>/<experimentID>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/epoch[N].json
    where N=1-#epochs
    <_LOCAL_LOGS_LOCATION>/<experimentID>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/<scenario>/<readType>/pod_name
    <_LOCAL_LOGS_LOCATION>/<experimentID>/<fileSize>/<fileSize>-<blockSize>-<numThreads>-<filesPerThread>-<hash>/gcsfuse-generic/<readType>/gcsfuse_mount_options

    Output dict structure:
    "{read_type}-{file_size}-{bs}-{numjobs}-{nrfiles}":
        "file_size": str
        "read_type": str
        "records":
            "local-ssd": [record1, record2, record3, ...]
            "gcsfuse-generic": [record1, record2, record3, ...]
  """

  output = {}
  for root, _, files in os.walk(
      _LOCAL_LOGS_LOCATION + "/" + args.experiment_id
  ):
    print(f"Parsing directory {root} ...")

    metadata_values = dict()
    if not files:
      # Ignore intermediate directories.
      # Don't combine it with the else or it will lead to a lot of unnecessary
      # logs.
      continue
    else:
      # Skip this directory if doesn't have any epoch[N].json file in it.
      if not any(re.search(_EPOCH_FILENAME_REGEX, file) for file in files):
        print(
            f"Directory {root} does not have any"
            " epoch[N].json files in it, so skipping it."
        )
        continue
      # Skip this directory if it is missing any of metadata files
      # i.e. gcsfuse_mount_options, pod_name, bucket_name, machine_type etc.
      skip_this_directory = False
      for metadata in [
          "gcsfuse_mount_options",
          "bucket_name",
          "machine_type",
          "pod_name",
      ]:
        metadata_file = root + f"/{metadata}"
        if os.path.isfile(metadata_file):
          with open(metadata_file) as f:
            metadata_values[metadata] = f.read().strip()
        else:
          print(
              f"Warning: Directory {root} does not have file {metadata} in it"
              " as expected, so skipping it."
          )
          skip_this_directory = True
          break

      if skip_this_directory:
        continue

    for file in files:
      if not re.search(_EPOCH_FILENAME_REGEX, file):
        continue

      per_epoch_output = root + f"/{file}"
      with open(per_epoch_output, "r") as f:
        try:
          per_epoch_output_data = json.load(f)
        except Exception as e:
          print(
              f"Warning: failed to json-parse {per_epoch_output}, so skipping"
              f" it. Error: {e}"
          )
          continue

      # This print is for debugging in case something goes wrong.
      print(f"Now parsing file {per_epoch_output} ...")

      # Get epoch number, and key from the file path.
      root_split = root.split("/")
      key = root_split[
          -3
      ]  # key is unique identifier for a single FIO workload.
      scenario = root_split[-2]
      epoch = int(
          re.match(_EPOCH_NUMBER_MATCH_REGEX, per_epoch_output).group(1)
      )

      # Confirm that the per_epoch_output_data has all the required attributes.
      for attr in {"global options", "jobs", "timestamp_ms"}:
        if not attr in per_epoch_output_data:
          print(
              f"Warning: '{attr}' missing in FIO output file"
              f" {per_epoch_output}, so skipping this file."
          )
          continue
      global_options = per_epoch_output_data["global options"]
      jobs = per_epoch_output_data["jobs"]
      timestamp_ms = per_epoch_output_data["timestamp_ms"]

      for attr in {"nrfiles", "numjobs", "ioengine", "direct", "iodepth"}:
        if not attr in global_options:
          print(
              f"Warning: '{attr}' missing in 'global options' in FIO output"
              f" file {per_epoch_output}, so skipping this file."
          )
          continue
      nrfiles = int(global_options["nrfiles"])
      numjobs = int(global_options["numjobs"])

      if not isinstance(jobs, Sequence) or len(jobs) == 0:
        print(
            f"Warning: No jobs found in FIO output file {per_epoch_output}, so"
            " skipping this file."
        )
        continue
      elif len(jobs) > 1:
        print(
            "Warning: More than 1 job found in FIO output file"
            f" {per_epoch_output}, which is not supported, so skipping this"
            " file."
        )
        continue
      job0 = jobs[0]

      for attr in ["job options", "read", "job_start"]:
        if not attr in job0:
          print(
              f'Warning: Did not find "[jobs][0][{attr}]" in'
              f" {per_epoch_output}, so skipping this file."
          )
          continue
      job0_options = job0["job options"]
      job0_read_metrics = job0["read"]
      job0_start = job0["job_start"]

      if not isinstance(job0_options, dict):
        print(
            'Warning: "[jobs][0][job options]" is not of type dict in'
            f" {per_epoch_output}, so skipping this file"
        )
        continue

      for attr in ["bs", "filesize", "rw"]:
        if not attr in job0_options:
          print(
              f'Warning: Did not find "[jobs][0][job options][{attr}]" in'
              f" {per_epoch_output}, so skipping this file"
          )
          continue
      bs = job0_options["bs"]
      file_size = job0_options["filesize"]
      read_type = job0_options["rw"]

      # If the record for this key has not been added, create a new entry
      # for it.
      if key not in output:
        output[key] = {
            "file_size": file_size,
            "read_type": read_type,
            "records": {
                scenario: [],
            },
        }

      if not isinstance(job0_read_metrics, dict):
        print(
            "Warning: jobs[0]['read'] is not of type dict in"
            f" {per_epoch_output}, so skipping this file."
        )
        continue

      # Create a record for this key.
      r = record.copy()
      try:
        for metadata, metadata_value in metadata_values.items():
          r[metadata] = metadata_value
        r["epoch"] = epoch
        r["scenario"] = scenario
        r["duration"] = int(job0_read_metrics["runtime"] / 1000)
        r["IOPS"] = int(job0_read_metrics["iops"])
        r["throughput_bytes_per_second"] = job0_read_metrics["bw_bytes"]
        r["throughput_mib_per_second"] = round(
            (r["throughput_bytes_per_second"] / (1024**2)), 2
        )
        r["throughput_mb_per_second"] = round(
            (r["throughput_bytes_per_second"] / 1e6), 2
        )
        r["start_epoch"] = job0_start // 1000
        r["end_epoch"] = timestamp_ms // 1000
        r["start"] = unix_to_timestamp(job0_start)
        r["end"] = unix_to_timestamp(timestamp_ms)

        fetch_cpu_memory_data(args=args, record=r)

        r["blockSize"] = bs
        r["filesPerThread"] = nrfiles
        r["numThreads"] = numjobs
        clat_ns = job0_read_metrics["clat_ns"]
        r["e2e_latency_ns_max"] = clat_ns["max"]
        clat_ns_percentile = clat_ns["percentile"]
        r["e2e_latency_ns_p50"] = clat_ns_percentile["50.000000"]
        r["e2e_latency_ns_p90"] = clat_ns_percentile["90.000000"]
        r["e2e_latency_ns_p99"] = clat_ns_percentile["99.000000"]
        r["e2e_latency_ns_p99.9"] = clat_ns_percentile["99.900000"]
      except Exception as e:
        print(
            f"Warning: failed to create record for key {key} for FIO output"
            f" file {per_epoch_output} with error: {e}, metadata:"
            f" {repr(metadata_values)}, so skipping this file."
        )
        # This print is for debugging in case something goes wrong.
        pprint.pprint(r)
        continue

      # If a slot for record for this particular epoch has not been created yet,
      # append enough empty records to make a slot.
      while len(output[key]["records"][scenario]) < epoch:
        output[key]["records"][scenario].append({})

      # Insert the record at the appropriate slot.
      output[key]["records"][scenario][epoch - 1] = r

  return output


def write_records_to_csv_output_file(output: dict, output_file_path: str):
  with open(output_file_path, "a") as output_file_fwr:
    # Write a new header.
    output_file_fwr.write(
        "File Size,Read Type,Scenario,Epoch,Duration"
        " (s),Throughput (MiB/s),IOPS,Throughput over Local SSD (%),GCSFuse"
        " Lowest"
        " Memory (MiB),GCSFuse Highest Memory (MiB),GCSFuse Lowest CPU"
        " (core),GCSFuse Highest CPU"
        " (core),Pod,Start,End,GcsfuseMoutOptions,BlockSize,FilesPerThread,NumThreads,ExperimentID,"
        "e2e_latency_ns_max,e2e_latency_ns_p50,e2e_latency_ns_p90,e2e_latency_ns_p99,e2e_latency_ns_p99.9,"
        "bucket_name,machine_type,"  #
        "Throughput (MB/s)"
        "\n",
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
                  r["throughput_mib_per_second"]
                  / record_set["records"]["local-ssd"][i][
                      "throughput_mib_per_second"
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
              f"{record_set['file_size']},{record_set['read_type']},{scenario},{r['epoch']},{r['duration']},{r['throughput_mib_per_second']},{r['IOPS']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']},\"{r['gcsfuse_mount_options']}\",{r['blockSize']},{r['filesPerThread']},{r['numThreads']},{args.experiment_id},"
          )
          output_file_fwr.write(
              f"{r['e2e_latency_ns_max']},{r['e2e_latency_ns_p50']},{r['e2e_latency_ns_p90']},{r['e2e_latency_ns_p99']},{r['e2e_latency_ns_p99.9']},"
          )
          output_file_fwr.write(f"{r['bucket_name']},{r['machine_type']},")
          output_file_fwr.write(f"{r['throughput_mb_per_second']}\n")

    output_file_fwr.close()
    print(
        "\nSuccessfully published outputs of FIO test runs to"
        f" {output_file_path} !!!\n"
    )


def fio_workload_id(row: FioTableRow) -> str:
  return f"{row.experiment_id}_{row.operation}_{row.file_size}_{row.block_size}_{row.num_threads}_{row.files_per_thread}_{row.start_epoch}"


def write_records_to_bq_table(
    output: dict,
    experiment_id: str,
    bq_project_id: str,
    bq_dataset_id: str,
    bq_table_id: str,
):
  # list of FioRowTable objects to be populated to be inserted into BigQuery
  # table using the above exporter.
  rows = []

  for key in output:
    record_set = output[key]

    for scenario in record_set["records"]:
      if scenario not in SUPPORTED_SCENARIOS:
        print(f"Unknown scenario: {scenario}. Ignoring it...")
        continue

      for epoch in range(len(record_set["records"][scenario])):
        r = record_set["records"][scenario][epoch]
        row = FioTableRow()
        row.experiment_id = experiment_id
        row.epoch = r["epoch"]
        row.operation = record_set["read_type"]
        row.file_size = record_set["file_size"]
        row.file_size_in_bytes = convert_size_to_bytes(row.file_size)
        row.block_size = r["blockSize"]
        row.block_size_in_bytes = convert_size_to_bytes(row.block_size)
        row.num_threads = r["numThreads"]
        row.files_per_thread = r["filesPerThread"]
        row.bucket_name = r["bucket_name"]
        row.machine_type = r["machine_type"]
        row.gcsfuse_mount_options = r["gcsfuse_mount_options"]
        row.start_time = Timestamp(r["start"])
        row.end_time = Timestamp(r["end"])
        row.start_epoch = r["start_epoch"]
        row.end_epoch = r["end_epoch"]
        row.duration_in_seconds = r["duration"]
        row.lowest_cpu_usage = r["lowest_cpu"]
        row.highest_cpu_usage = r["highest_cpu"]
        row.lowest_memory_usage = r["lowest_memory"]
        row.highest_memory_usage = r["highest_memory"]
        row.pod_name = r["pod_name"]
        row.scenario = scenario
        row.e2e_latency_ns_max = r["e2e_latency_ns_max"]
        row.e2e_latency_ns_p50 = r["e2e_latency_ns_p50"]
        row.e2e_latency_ns_p90 = r["e2e_latency_ns_p90"]
        row.e2e_latency_ns_p99 = r["e2e_latency_ns_p99"]
        row.e2e_latency_ns_p99_9 = r["e2e_latency_ns_p99.9"]
        row.iops = r["IOPS"]
        row.throughput_in_mbps = r["throughput_mb_per_second"]
        row.fio_workload_id = fio_workload_id(row)

        rows.append(row)

  if len(rows) == 0:
    print("No output rows to insert into the BQ table.")
    return

  fioBqExporter = FioBigqueryExporter(bq_project_id, bq_dataset_id, bq_table_id)
  fioBqExporter.insert_rows(fioTableRows=rows)
  print(
      "\nSuccessfully exported outputs of FIO test runs to"
      f" BigQuery table {bq_project_id}:{bq_dataset_id}.{bq_table_id} !!!\n"
  )


if __name__ == "__main__":
  args = parse_arguments(fio_or_dlio="FIO", add_bq_support=True)
  ensure_directory_exists(_LOCAL_LOGS_LOCATION)

  if not args.predownloaded_output_files:
    fioWorkloads = fio_workload.parse_test_config_for_fio_workloads(
        args.workload_config
    )
    download_fio_outputs(fioWorkloads, args.experiment_id)

  output = create_output_scenarios_from_downloaded_files(args)
  if len(output) == 0:
    print(f"\nNo FIO outputs found for experiment_id {args.experiment_id} !\n")
  else:
    # Export output dict to CSV.
    output_file_path = args.output_file
    # Create the parent directory of output_file_path if doesn't exist already.
    ensure_directory_exists(os.path.dirname(output_file_path))
    write_records_to_csv_output_file(output, output_file_path)

    # Export output dict to bigquery table.
    if (
        args.bq_project_id
        and args.bq_project_id.strip()
        and args.bq_dataset_id
        and args.bq_dataset_id.strip()
        and args.bq_table_id
        and args.bq_table_id.strip()
    ):
      write_records_to_bq_table(
          output=output,
          bq_project_id=args.bq_project_id,
          bq_dataset_id=args.bq_dataset_id,
          bq_table_id=args.bq_table_id,
          experiment_id=args.experiment_id,
      )
