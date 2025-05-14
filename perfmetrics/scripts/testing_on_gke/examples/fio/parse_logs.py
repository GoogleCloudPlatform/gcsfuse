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
from utils.parse_logs_common import ensure_directory_exists, download_gcs_objects, parse_arguments, SUPPORTED_SCENARIOS, fetch_cpu_memory_data
from fio.bq_utils import FioBigqueryExporter, FioTableRow, Timestamp
from utils.utils import unix_to_timestamp, convert_size_to_bytes

_LOCAL_LOGS_LOCATION = "../../bin/fio-logs"

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


def download_fio_outputs(fioWorkloads: set, instanceId: str) -> int:
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


def create_output_scenarios_from_downloaded_files(args: dict) -> dict:
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

    # if directory contains bucket_name file, then parse it.
    bucket_name = ""
    bucket_name_file = root + "/bucket_name"
    if os.path.isfile(bucket_name_file):
      with open(bucket_name_file) as f:
        bucket_name = f.read().strip()
        print(f"bucket_name={bucket_name}")

    # if directory contains machine_type file, then parse it.
    machine_type = ""
    machine_type_file = root + "/machine_type"
    if os.path.isfile(machine_type_file):
      with open(machine_type_file) as f:
        machine_type = f.read().strip()
        print(f"machine_type={machine_type}")

    # if directory has files, it must also contain pod_name file,
    # and we should extract pod-name from it in the record.
    pod_name = ""
    pod_name_file = root + "/pod_name"
    if os.path.isfile(pod_name_file):
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
      job0 = per_epoch_output_data["jobs"][0]

      bs = job0["job options"]["bs"]

      # If the record for this key has not been added, create a new entry
      # for it.
      if key not in output:
        output[key] = {
            "mean_file_size": mean_file_size,
            "read_type": read_type,
            "records": {
                "local-ssd": [],
                "gcsfuse-generic": [],
            },
        }

      job0_read_metrics = job0["read"]
      bs = job0["job options"]["bs"]

      # Create a record for this key.
      r = record.copy()
      try:
        r["pod_name"] = pod_name
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
        r["start_epoch"] = job0["job_start"] // 1000
        r["end_epoch"] = per_epoch_output_data["timestamp_ms"] // 1000
        r["start"] = unix_to_timestamp(job0["job_start"])
        r["end"] = unix_to_timestamp(per_epoch_output_data["timestamp_ms"])

        fetch_cpu_memory_data(args=args, record=r)

        r["gcsfuse_mount_options"] = gcsfuse_mount_options
        r["bucket_name"] = bucket_name
        r["machine_type"] = machine_type
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
        print(f"Failed to create following record with error: {e}")
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
        " (core),Pod,Start,End,GcsfuseMoutOptions,BlockSize,FilesPerThread,NumThreads,InstanceID,"
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
              f"{record_set['mean_file_size']},{record_set['read_type']},{scenario},{r['epoch']},{r['duration']},{r['throughput_mib_per_second']},{r['IOPS']},{r['throughput_over_local_ssd']},{r['lowest_memory']},{r['highest_memory']},{r['lowest_cpu']},{r['highest_cpu']},{r['pod_name']},{r['start']},{r['end']},\"{r['gcsfuse_mount_options']}\",{r['blockSize']},{r['filesPerThread']},{r['numThreads']},{args.instance_id},"
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
  fioBqExporter = FioBigqueryExporter(bq_project_id, bq_dataset_id, bq_table_id)

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
        row.file_size = record_set["mean_file_size"]
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
    download_fio_outputs(fioWorkloads, args.instance_id)

  output = create_output_scenarios_from_downloaded_files(args)

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
        experiment_id=args.instance_id,
    )
