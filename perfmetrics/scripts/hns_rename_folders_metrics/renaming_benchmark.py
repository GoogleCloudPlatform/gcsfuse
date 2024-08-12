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
# python3 renaming_benchmark.py [--flat_dir_config_file flat_dir-config.json] \
# [--hns_dir_config_file hns_dir-config.json] [--upload_gs]  [--num_samples NUM_SAMPLES]
# where dir-config.json file contains the directory structure details for the test.

import os
import sys
import argparse
import logging
import subprocess
import time
import json
import statistics as stat
import numpy as np

sys.path.insert(0, '..')
from generate_folders_and_files import _check_for_config_file_inconsistency,_check_if_dir_structure_exists
from utils.mount_unmount_util import mount_gcs_bucket, unmount_gcs_bucket
from utils.checks_util import check_dependencies
from gsheet import gsheet

WORKSHEET_NAME_FLAT = 'rename_metrics_flat'
WORKSHEET_NAME_GCS = 'rename_metrics_hns'
SPREADSHEET_ID = '1UVEvsf49eaDJdTGLQU1rlNTIAxg8PZoNQCy_GX6Nw-A'

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()


def _upload_to_gsheet(worksheet, data, spreadsheet_id) -> (int):
  """
  Writes rename results to Google Spreadsheets.
  Args:
    worksheet (str): Google sheet name to which results will be uploaded.
    data (list): metrics to be uploaded.
    spreadsheet_id: Google spreadsheet id.
  """
  # Changing directory to comply with "cred.json" path in "gsheet.py".
  os.chdir('..')
  exit_code = 0
  if spreadsheet_id == "":
    log.error('Empty spreadsheet id passed!')
    exit_code = 1
  else:
    gsheet.write_to_google_sheet(worksheet, data, spreadsheet_id)
  # Changing the directory back to current directory.
  os.chdir('./hns_rename_folders_metrics')
  return exit_code


def _get_values_to_export(dir, metrics, test_type):
  """
  This function takes in extracted metrics data, filters it, rearranges it,
   and returns the modified data to export to Google Sheet.

   Args:
     dir: JSON object containing details of testing folders.
     metrics: A dictionary containing all the result metrics for each
              testing folder.
     test_type: flat or hns ,which the metrics are related to.

   Returns:
     list: List of results to upload to GSheet.
   """
  metrics_data = []
  # Getting values corrresponding to non nested folders.
  for folder in dir["folders"]["folder_structure"]:
    num_files = folder["num_files"]
    num_folders = 1

    row = [
        'Renaming Operation',
        test_type,
        num_files,
        num_folders,
        metrics[folder["name"]]['Number of samples'],
        metrics[folder["name"]]['Mean'],
        metrics[folder["name"]]['Median'],
        metrics[folder["name"]]['Standard Dev'],
        metrics[folder["name"]]['Min'],
        metrics[folder["name"]]['Max'],
        metrics[folder["name"]]['Quantiles']['0 %ile'],
        metrics[folder["name"]]['Quantiles']['20 %ile'],
        metrics[folder["name"]]['Quantiles']['50 %ile'],
        metrics[folder["name"]]['Quantiles']['90 %ile'],
        metrics[folder["name"]]['Quantiles']['95 %ile'],
        metrics[folder["name"]]['Quantiles']['98 %ile'],
        metrics[folder["name"]]['Quantiles']['99 %ile'],
        metrics[folder["name"]]['Quantiles']['99.5 %ile'],
        metrics[folder["name"]]['Quantiles']['99.9 %ile'],
        metrics[folder["name"]]['Quantiles']['100 %ile']

    ]

    metrics_data.append(row)

  return metrics_data


def _compute_metrics_from_time_of_operation(num_samples,results):
  """
   This function takes in a list containing the time of operation for num_samples
   test runs.This will be used to generate various metrics like Mean,Median,Standard
   Dev,Minimum,Maximum,etc.

   Args:
     num_samples: Number of samples collected for each test.
     results: list containing time of operation corresponding to num_samples
              number of tests.

   Returns:
     A dictionary containing the various metrics in a JSON format.
   """
  metrics=dict()
  metrics['Number of samples'] = num_samples

  # Sorting based on time to get the mean,median,etc.
  results = sorted(results)

  metrics['Mean'] = round(
      stat.mean(results), 3)
  metrics['Median'] = round(
      stat.median(results), 3)
  metrics['Standard Dev'] = round(
      stat.stdev(results), 3)
  metrics['Min']= round(
      np.min(results),3)
  metrics['Max']= round(
      np.max(results),3)
  metrics['Quantiles'] = dict()
  sample_set = [0, 20, 50, 90, 95, 98, 99, 99.5, 99.9, 100]
  for percentile in sample_set:
    metrics['Quantiles'][
      '{} %ile'.format(percentile)] = round(
        np.percentile(results, percentile), 3)

  return metrics


def _parse_results(dir, results, num_samples):
  """
  This function takes in dictionary containing the list of results for each
  testing folder which will be used to generate various metrics.

  Args:
    dir: JSON object representing folder structure in gcs bucket.
    num_samples: Number of samples collected for each test.
    results: Dictionary containing the list of results for each testing folder.

  Returns:
    A dictionary containing the various metrics in a JSON format.
  """
  metrics = dict()
  # Parsing metrics for non-nested folder.
  for folder in dir["folders"]["folder_structure"]:
    folder_name = folder["name"]
    metrics[folder_name] = _compute_metrics_from_time_of_operation(
        num_samples, results[folder_name])
  #TODO add logic for metrics parsing for nested folder

  return metrics


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
    # Creating config file for mounting with hns enabled.
    with open("config.yml",'w') as mount_config:
      mount_config.write("enable-hns: true")
    hns_mount_flags="--config-file=config.yml"
    hns_bucket_name=mount_gcs_bucket(dir["name"], hns_mount_flags, log)

    # Record time of operation and populate the results dict.
    flat_results = _record_time_of_operation(hns_bucket_name, dir, num_samples)
    results["hns"] = flat_results

    unmount_gcs_bucket(dir["name"], log)
    # Deleting config file for hns enabled mounting.
    os.remove("config.yml")

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
      '--flat_dir_config_file',
      help='Provide path of the config file for flat bucket.',
      action='store',
      required=False,
  )
  parser.add_argument(
      '--hns_dir_config_file',
      help='Provide path of the config file for hns bucket.',
      action='store',
      required=False,
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


def _run_rename_benchmark(test_type,dir_config,num_samples,results,upload_gs):
  with open(os.path.abspath(dir_config)) as file:
    dir_str = json.load(file)

  exit_code = _check_for_config_file_inconsistency(dir_str)
  if exit_code != 0:
    log.error('Exited with code {}'.format(exit_code))
    sys.exit(1)

  # Check if test data exists.
  dir_structure_present = _check_if_dir_structure_exists(dir_str)
  if not dir_structure_present:
    log.error("Test data does not exist.To create test data, run : \
        python3 generate_folders_and_files.py {} ".format(dir_config))
    sys.exit(1)

  _perform_testing(dir_str, test_type, num_samples, results)
  parsed_metrics = _parse_results(dir_str, results[test_type], num_samples)
  upload_values = _get_values_to_export(dir_str, parsed_metrics,
                                             test_type)

  if upload_gs:
    log.info('Uploading files to the Google Sheet\n')
    if test_type == "flat":
      worksheet= WORKSHEET_NAME_FLAT
    else:
      worksheet= WORKSHEET_NAME_GCS

    exit_code = _upload_to_gsheet(worksheet, upload_values,
                                  SPREADSHEET_ID)
    if exit_code != 0:
      log.error("Upload to gsheet failed!")
  else:
    print(upload_values)


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 renaming_benchmark.py  [--upload_gs] [--num_samples NUM_SAMPLES] config_file ')

  args = _parse_arguments(argv)
  check_dependencies(['gcloud', 'gcsfuse'], log)
  results = dict()  # Dict object to store the results corresonding to the test types.

  # If the config file for flat bucket is provided, we will perform rename benchmark
  # for flat bucket.
  if args.flat_dir_config_file:
    _run_rename_benchmark("flat", args.flat_dir_config_file, args.num_samples,
                          results, args.upload_gs)

  # If the config file for hns bucket is provided, we will perform rename benchmark
  # for hns bucket.
  if args.hns_dir_config_file:
    _run_rename_benchmark("hns", args.hns_dir_config_file, args.num_samples,
                          results, args.upload_gs)
