# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Python script for benchmarking stat operation.

This python script benchmarks and compares the latency of stat operation in
persistent disk vs GCS bucket. It creates the necessary directory structure,
containing files and folders, needed to test the stat operation.

Typical usage example:

  $ python3 stat_benchmark.py [--keep_files] [--num_samples NUM_SAMPLES] [--message MESSAGE] --gcsfuse_flags GCSFUSE_FLAGS config_file

"""

import argparse
import json
import logging
import os
import statistics as stat
import subprocess
import sys
import time
import numpy as np

import directory_pb2 as directory_proto

sys.path.insert(0, '..')
import generate_files
from google.protobuf.json_format import ParseDict

from utils.mount_unmount_util import mount_gcs_bucket, unmount_gcs_bucket
from utils.checks_util import check_dependencies

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()

RUN_1M_TEST=False

def _list_directory(path) -> list:
  """Returns the list containing path of all the contents present in the current directory.

  Args:
    path: Path of the directory.

  Returns:
    A list containing path of all contents present in the input path.
  """

  contents = subprocess.check_output(
      'gsutil -m ls {}'.format(path), shell=True)
  contents_url = contents.decode('utf-8').split('\n')[:-1]
  return contents_url


def _compare_directory_structure(url, directory_structure) -> bool:
  """Compares the directory structure present in the GCS bucket with the structure present in the JSON config file.

  Args:
    url: Path of the directory to compare in the GCS bucket.
    directory_structure: Protobuf of the current directory.

  Returns:
    True if GCS bucket contents matches the directory structure.
  """

  contents_url = _list_directory(url)
  # gsutil in some cases return the contents_url list with the current
  # directory in the first index. We dont want the current directory so
  # we remove it manually.
  if contents_url and contents_url[0] == url:
    contents_url = contents_url[1:]

  files = []
  folders = []
  for content in contents_url:
    if content[-1] == '/':
      folders.append(content)
    else:
      files.append(content)

  if len(folders) != directory_structure.num_folders:
    return False

  if len(files) != directory_structure.num_files:
    return False

  result = True
  for folder in directory_structure.folders:
    if not RUN_1M_TEST and folder.name == "1KB_1000000files_0subdir":
      # Excluding test case with 1m files from HNS in daily periodic tests.
      continue
    new_url = url + folder.name + '/'
    if new_url not in folders:
      return False
    result = result and _compare_directory_structure(new_url, folder)

  return result

def _create_directory_structure(
    gcs_bucket_url, persistent_disk_url, directory_structure,
    create_files_in_gcs) -> int:
  """Creates new directory structure using generate_files.py as a library.

  Args:
   gcs_bucket_url: Path of the directory in GCS bucket.
   persistent_disk_url: Path in the persistent disk.
   directory_structure: Protobuf of the current directory.
   create_files_in_gcs: Bool value which is True if we have to create files
                        in GCS bucket.

  Returns:
    1 if an error is encountered while uploading files to the GCS bucket.
    0 if no error is encountered.
  """

  # Create a directory in the persistent disk.
  subprocess.call('mkdir -p {}'.format(persistent_disk_url), shell=True)

  if directory_structure.num_files != 0:
    file_size = int(directory_structure.file_size[:-2])
    file_size_unit = directory_structure.file_size[-2:]
    exit_code = generate_files.generate_files_and_upload_to_gcs_bucket(
        gcs_bucket_url, directory_structure.num_files, file_size_unit,
        file_size, directory_structure.file_name_prefix, persistent_disk_url,
        create_files_in_gcs)
    if exit_code != 0:
      return exit_code

  result = 0
  for folder in directory_structure.folders:
    if not RUN_1M_TEST and folder.name == "1KB_1000000files_0subdir":
      continue
    result += _create_directory_structure(gcs_bucket_url + folder.name + '/',
                                          persistent_disk_url + folder.name + '/',
                                          folder, create_files_in_gcs)

  return int(result > 0)


def _generate_all_paths(base_path, directory_proto, paths_list):
    """Generates a list of all file paths expected in the directory structure.

    Args:
        base_path: The root path to traverse (local path).
        directory_proto: The directory structure protobuf.
        paths_list: The list to append paths to.
    """
    # Generate file paths
    # Note: generate_files.py creates files with .txt extension
    # and 1-based indexing for file_num
    prefix = directory_proto.file_name_prefix
    num_files = directory_proto.num_files
    for i in range(1, num_files + 1):
        name = '{}_{}.txt'.format(prefix, i)
        paths_list.append(os.path.join(base_path, name))

    # Recurse for folders
    for folder in directory_proto.folders:
        folder_path = os.path.join(base_path, folder.name)
        # Add folder path itself if we want to stat folders?
        # listing_benchmark focuses on files usually, but stat works on folders too.
        # Let's include folders.
        paths_list.append(folder_path)
        _generate_all_paths(folder_path, folder, paths_list)


def _measure_stat_latencies(paths, num_samples) -> list:
  """Runs stat on the given paths for given num_samples times.

  Args:
    paths: List of file/folder paths to stat.
    num_samples: Number of times to run the test loop.

  Returns:
    A list containing the latencies of all stat operations in milliseconds.
  """

  latencies = []
  for _ in range(num_samples):
    for file_path in paths:
        try:
            start_time = time.time()
            os.stat(file_path)
            end_time = time.time()
            latencies.append((end_time - start_time) * 1000)
        except OSError as e:
            log.warning("Failed to stat %s: %s", file_path, e)

  return latencies


def _perform_testing(
    folders, gcs_bucket, persistent_disk, num_samples):
  """Tests the stat operation.

  Args:
    folders: List of protobufs containing the testing folders.
    gcs_bucket: Name of the directory to which GCS bucket is mounted.
    persistent_disk: Name of the directory in persistent disk.
    num_samples: Number of times to run each test.

  Returns:
    gcs_bucket_results: A dictionary containing list of latencies for each folder.
    persistent_disk_results: A dictionary containing list of latencies for each folder.
  """

  gcs_bucket_results = {}
  persistent_disk_results = {}

  for testing_folder in folders:
    if not RUN_1M_TEST and testing_folder.name == "1KB_1000000files_0subdir":
      continue

    log.info('Testing started for testing folder: %s\n', testing_folder.name)
    local_dir_path = os.path.join(persistent_disk, testing_folder.name)
    gcs_bucket_path = os.path.join(gcs_bucket, testing_folder.name)

    # Generate expected paths for this folder structure
    # We create a dummy proto for the current testing folder to generate relative paths
    # properly or just pass the full path?
    # _generate_all_paths is recursive.
    # We should generate paths relative to the testing_folder.

    pd_paths = []
    _generate_all_paths(local_dir_path, testing_folder, pd_paths)

    gcs_paths = []
    _generate_all_paths(gcs_bucket_path, testing_folder, gcs_paths)

    # We should probably shuffle or just iterate? Iterating is fine.

    log.info("Running stat on Persistent Disk (%d paths)...", len(pd_paths))
    persistent_disk_results[testing_folder.name] = _measure_stat_latencies(
        pd_paths, num_samples)

    log.info("Running stat on GCS Bucket (%d paths)...", len(gcs_paths))
    gcs_bucket_results[testing_folder.name] = _measure_stat_latencies(
        gcs_paths, num_samples)

  log.info('Testing completed. Generating output.\n')
  return gcs_bucket_results, persistent_disk_results


def _print_metrics(folders, results_list, message, num_samples) -> dict:
  """Outputs the results on the console."""

  metrics = dict()
  print("-" * 60)
  print(f"Results for: {message}")
  print("-" * 60)

  for testing_folder in folders:
    if not RUN_1M_TEST and testing_folder.name == "1KB_1000000files_0subdir":
      continue

    folder_name = testing_folder.name
    data = results_list.get(folder_name, [])

    if not data:
        print(f"No data for {folder_name}")
        continue

    metrics[folder_name] = dict()
    metrics[folder_name]['Test Desc.'] = message
    metrics[folder_name]['Number of samples'] = num_samples # This is samples of the loop, but data has len(paths)*num_samples points

    # Sorting based on time.
    data = sorted(data)

    mean_val = stat.mean(data)
    median_val = stat.median(data)
    stdev_val = stat.stdev(data) if len(data) > 1 else 0.0

    metrics[folder_name]['Mean'] = round(mean_val, 3)
    metrics[folder_name]['Median'] = round(median_val, 3)
    metrics[folder_name]['Standard Dev'] = round(stdev_val, 3)

    metrics[folder_name]['Quantiles'] = dict()
    sample_set = [0, 20, 50, 90, 95, 98, 99, 99.5, 99.9, 100]
    for percentile in sample_set:
      metrics[folder_name]['Quantiles'][
        '{} %ile'.format(percentile)] = round(
          np.percentile(data, percentile), 3)

    print(f"Folder: {folder_name}")
    print(f"  Count: {len(data)}")
    print(f"  Mean: {mean_val:.3f} ms")
    print(f"  Median: {median_val:.3f} ms")
    print(f"  P99: {np.percentile(data, 99):.3f} ms")
    print("-" * 60)

  return metrics


def _parse_arguments(argv):
  """Parses the arguments provided to the script via command line."""

  argv = sys.argv
  parser = argparse.ArgumentParser()
  parser.add_argument(
      'config_file',
      help='Provide path of the config file.',
      action='store'
  )
  parser.add_argument(
      '--keep_files',
      help='Does not delete the directory structure in persistent disk.',
      action='store_true',
      default=False,
      required=False,
  )
  parser.add_argument(
      '--message',
      help='Puts a message/title describing the test.',
      action='store',
      nargs=1,
      default=['Stat Latency Benchmark'],
      required=False,
  )
  parser.add_argument(
      '--num_samples',
      help='Number of times to iterate over the directory structure.',
      action='store',
      nargs=1,
      default=[1],
      required=False,
  )
  parser.add_argument(
      '--gcsfuse_flags',
      help='Gcsfuse flags for mounting the bucket.',
      action='store',
      nargs=1,
      required=True,
  )
  parser.add_argument(
      '--run_1m_test',
      help='Perform benchmark on 1m files directory? [True/False]',
      action='store_true',
      default=False,
      required=False,
  )

  return parser.parse_args(argv[1:])


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
     # Just a basic check, argparse will handle details
     pass

  args = _parse_arguments(argv)

  check_dependencies(['gsutil', 'gcsfuse'], log)

  with open(os.path.abspath(args.config_file)) as file:
    config_json = json.load(file)
  directory_structure = ParseDict(config_json, directory_proto.Directory())

  log.info('Started checking the directory structure in the bucket.\n')
  directory_structure_present = _compare_directory_structure(
      'gs://{}/'.format(directory_structure.name), directory_structure)

  persistent_disk = 'persistent_disk'
  if os.path.exists('./{}'.format(persistent_disk)):
    subprocess.call('rm -rf {}'.format(persistent_disk), shell=True)

  if not directory_structure_present:
    log.info(
        """Similar directory structure not found in the GCS bucket.
        Creating a new one.\n""")
    log.info('Deleting previously present directories in the GCS bucket.\n')
    subprocess.call(
        'gsutil -m rm -r gs://{}/*'.format(directory_structure.name),
        shell=True, stdout=subprocess.DEVNULL, stderr=subprocess.STDOUT)

  # Creating a temp directory which will be needed by the generate_files
  temp_dir = generate_files.TEMPORARY_DIRECTORY
  if os.path.exists(os.path.dirname(temp_dir)):
    subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)
  subprocess.call('mkdir -p {}'.format(temp_dir), shell=True)

  exit_code = _create_directory_structure(
      'gs://{}/'.format(directory_structure.name),
      './{}/'.format(persistent_disk), directory_structure,
      not directory_structure_present)

  subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)

  if exit_code != 0:
    log.error('Cannot create files in the GCS bucket. Error encountered.\n')
    sys.exit(1)

  log.info('Directory Structure Created.\n')

  gcs_bucket = mount_gcs_bucket(directory_structure.name,
                                args.gcsfuse_flags[0], log)

  RUN_1M_TEST=args.run_1m_test

  try:
      gcs_bucket_results, persistent_disk_results = _perform_testing(
          directory_structure.folders, gcs_bucket, persistent_disk,
          int(args.num_samples[0]))

      print("\n=== Persistent Disk Results ===")
      _print_metrics(
          directory_structure.folders, persistent_disk_results, args.message[0],
          int(args.num_samples[0]))

      print("\n=== GCS Bucket Results ===")
      _print_metrics(
          directory_structure.folders, gcs_bucket_results, args.message[0],
          int(args.num_samples[0]))

  finally:
      unmount_gcs_bucket(gcs_bucket, log)
      if not args.keep_files:
        log.info('Deleting files from persistent disk.\n')
        subprocess.call('rm -rf {}'.format(persistent_disk), shell=True)
