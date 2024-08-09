# Copyright 2024 Google Inc. All Rights Reserved.
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
# To run the script,run in terminal:
# python3 renaming_benchmark.py <dir-config.json> [--upload_gs]  [--num_samples NUM_SAMPLES]
# where dir-config.json file contains the directory structure details for the test.

import os
import re
import sys
import argparse
import logging
import subprocess
import time
import json

sys.path.insert(0, '..')
from generate_folders_and_files import _check_for_config_file_inconsistency,_check_if_dir_structure_exists
from utils.mount_unmount_util import mount_gcs_bucket, unmount_gcs_bucket
from utils.checks_util import check_dependencies

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()


def _record_time_for_folder_rename(mount_point,folder,num_samples):
  """
  This function records the time of rename operation for folder,for num_samples
  number of test runs.

  Args:
    mount_point: Mount point for the GCS bucket.
    folder: JSON object representing the folder being renamed.
    num_samples: Number of samples to collect for each test.

  Returns:
    A list containing time of rename operations in seconds.
  """
  folder_name= '{}/{}'.format(mount_point,folder["name"])
  folder_rename = folder_name+"_renamed"
  time_op = []
  for iter in range(num_samples):
    # For the even iterations, we rename from folder_name to folder_name_renamed.
    if iter %2==0:
      rename_from = folder_name
      rename_to = folder_rename
    # For the odd iterations, we rename from folder_name_renamed to folder_name.
    else:
      rename_from = folder_rename
      rename_to = folder_name
    start_time_sec = time.time()
    subprocess.call('mv ./{} ./{}'.format(rename_from, rename_to), shell=True)
    end_time_sec = time.time()
    time_op.append(end_time_sec - start_time_sec)

  # If the number of samples is odd, we need another unrecorded rename operation
  # to restore test data back to its original form.
  if num_samples % 2 == 1:
    rename_from = folder_rename
    rename_to = folder_name
    subprocess.call('mv ./{} ./{}'.format(rename_from, rename_to), shell=True)

  return time_op


def _record_time_of_operation(mount_point, dir, num_samples):
  """
  This function records the time of rename operation for each testing folder
  inside dir directory,for num_samples number of test runs.

  Args:
    mount_point: Mount point for the GCS bucket.
    dir: JSON object representing the folders structure.
    num_samples: Number of samples to collect for each test.

  Returns:
    A dictionary with lists containing time of rename operations in seconds,
    corresponding to each folder.
  """
  results = dict()
  # Collecting metrics for non-nested folders.
  for folder in dir["folders"]["folder_structure"]:
    results[folder["name"]] = _record_time_for_folder_rename(mount_point,folder,num_samples)
  #TODO Add metric collection logic for nested-folders
  return results


def _perform_testing(dir, test_type, num_samples, results):
  """
  This function performs rename operations and records time of operation .
  Args:
    dir: JSON object representing the structure of data in bucket.Example:
      {
      "name": "example-bucket" ,
      "folders" : {
        "num_folders": 1,
        "folder_structure" : [
          {
            "name": "1k_files_folder" ,
            "num_files": 1000 ,
            "file_name_prefix": "file" ,
            "file_size": "1kb"
          }
        ]
        },
       "nested_folders": {
        "folder_name": "nested_folder",
        "num_folders": 1,
        "folder_structure" :  [
          {
            "name": "1k_files_nested_folder" ,
            "num_files": 1000 ,
            "file_name_prefix": "file" ,
            "file_size": "1kb"
          }
          ]
        }
      }
    test_type : flat or hns.
    num_samples: Number of samples to collect for each test.
    results: Dictionary to store the results corresponding to each test type
  """
  if test_type == "hns":
    # TODO add mount function for test type hns
    return

  # Mounting the gcs bucket.
  flat_mount_flags = "--implicit-dirs --rename-dir-limit=1000000"
  flat_bucket_name = mount_gcs_bucket(dir["name"], flat_mount_flags, log)

  # Record time of operation and populate the results dict.
  flat_results = _record_time_of_operation(flat_bucket_name, dir, num_samples)
  results["flat"] = flat_results

  unmount_gcs_bucket(dir["name"], log)


def _parse_arguments(argv):
  argv = sys.argv
  parser = argparse.ArgumentParser()

  parser.add_argument(
      'dir_config_file',
      help='Provide path of the config file.',
      action='store'
  )
  parser.add_argument(
      '--upload_gs',
      help='Upload the results to the Google Sheet.',
      action='store_true',
      default=False,
      required=False,
  )
  parser.add_argument(
      '--num_samples',
      help='Number of samples to collect of each test.',
      action='store',
      default=10,
      required=False,
      type=int,
  )

  return parser.parse_args(argv[1:])


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 renaming_benchmark.py  [--upload_gs] [--num_samples NUM_SAMPLES] config_file ')

  args = _parse_arguments(argv)
  check_dependencies(['gcloud', 'gcsfuse'], log)

  with open(os.path.abspath(args.dir_config_file)) as file:
    dir_str = json.load(file)

  exit_code = _check_for_config_file_inconsistency(dir_str)
  if exit_code != 0:
    log.error('Exited with code {}'.format(exit_code))
    sys.exit(1)

  # Check if test data exists.
  dir_structure_present = _check_if_dir_structure_exists(dir_str)
  if not dir_structure_present:
    log.error("Test data does not exist.To create test data, run : \
    python3 generate_folders_and_files.py <dir_config.json> ")
    sys.exit(1)

  results = dict()  # Dict object to store the results corresonding to the test types.
  _perform_testing(dir_str, "flat", args.num_samples, results)
