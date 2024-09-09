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
# python3 renaming_benchmark.py config.json bucket_type  [--upload_gs] \
# [--num_samples NUM_SAMPLES]
# where dir-config.json file contains the directory structure details for the test.

import os
import socket
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
from vm_metrics import vm_metrics

WORKSHEET_NAME_FLAT = 'rename_metrics_flat'
WORKSHEET_NAME_HNS = 'rename_metrics_hns'
WORKSHEET_VM_METRICS_FLAT = 'vm_metrics_flat'
WORKSHEET_VM_METRICS_HNS = 'vm_metrics_hns'
SPREADSHEET_ID = '1UVEvsf49eaDJdTGLQU1rlNTIAxg8PZoNQCy_GX6Nw-A'
INSTANCE=socket.gethostname()
PERIOD_SEC=120

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()


def _upload_to_gsheet(worksheet, data,vm_worksheet,vm_data, spreadsheet_id) -> (int):
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
    gsheet.write_to_google_sheet(vm_worksheet, vm_data, spreadsheet_id)
  # Changing the directory back to current directory.
  os.chdir('./hns_rename_folders_metrics')
  return exit_code


def _calculate_num_files(folder_structure):
  """
  Calculate the total number of files across folders specified in folder structure.
  Args:
    folder_structure: JSON list containing JSON objects representing folders.
  Returns:
    Total count of files.
  Examples:
    folder_structure:[
      {
      ...
      num_files:1 ,
      ...
      },
      {
      ...
      num_files:1,
      ...
      }
    ]
    For the above structure, the function returns 2.
  """
  count=0
  for folder in folder_structure:
    count+=folder["num_files"]
  return count


def _create_row_of_values(operation,test_type,num_files,num_folders,metrics):
  """
  Creates rows of values from the metrics dict to be uploaded to gsheet.
  Args:
    operation: Type of rename operation (whether involves nested folders or not)
    test_type: flat or hns
    num_files: Total number of files involved in the rename operation(filepath got affected)
    num_folders: Total number of folders renamed/folderpath got affected
    metrics: Dict object containing metrics to be uploaded.
  Returns:
    A row containing values to be uploaded.
  """
  row = [
      operation,
      test_type,
      num_files,
      num_folders,
      metrics['Number of samples'],
      metrics['Mean'],
      metrics['Median'],
      metrics['Standard Dev'],
      metrics['Min'],
      metrics['Max'],
      metrics['Quantiles']['0 %ile'],
      metrics['Quantiles']['20 %ile'],
      metrics['Quantiles']['50 %ile'],
      metrics['Quantiles']['90 %ile'],
      metrics['Quantiles']['95 %ile'],
      metrics['Quantiles']['98 %ile'],
      metrics['Quantiles']['99 %ile'],
      metrics['Quantiles']['99.5 %ile'],
      metrics['Quantiles']['99.9 %ile'],
      metrics['Quantiles']['100 %ile']

  ]
  return row


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

    row=_create_row_of_values('Renaming Operation',test_type,num_files,num_folders,metrics[folder["name"]])
    metrics_data.append(row)

  nested_folder_name=dir["nested_folders"]["folder_name"]
  num_files= _calculate_num_files(dir["nested_folders"]["folder_structure"])
  num_folders=dir["nested_folders"]["num_folders"]

  row=_create_row_of_values('Renaming Operation Nested',test_type,num_files,num_folders,metrics[nested_folder_name])
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
  metrics[dir["nested_folders"]["folder_name"]]= _compute_metrics_from_time_of_operation(
      num_samples, results[dir["nested_folders"]["folder_name"]])
  return metrics


def _record_time_for_folder_rename(parent_dir,folder,num_samples):
  """
  This function records the time of rename operation for folder,for num_samples
  number of test runs.

  Args:
    parent_dir: Parent directory for the folder.
    folder: JSON object representing the folder being renamed.
    num_samples: Number of samples to collect for each test.

  Returns:
    A list containing time of rename operations in seconds.
  """
  folder_name= '{}/{}'.format(parent_dir,folder["name"])
  folder_rename = folder_name+"_renamed"
  time_intervals_list=[]
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
    time_intervals_list.append([start_time_sec,end_time_sec])
    time_op.append(end_time_sec - start_time_sec)

  # If the number of samples is odd, we need another unrecorded rename operation
  # to restore test data back to its original form.
  if num_samples % 2 == 1:
    rename_from = folder_rename
    rename_to = folder_name
    subprocess.call('mv ./{} ./{}'.format(rename_from, rename_to), shell=True)

  return time_op,time_intervals_list


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
  time_interval_for_vm_metrics={}
  # Collecting metrics for non-nested folders.
  for folder in dir["folders"]["folder_structure"]:
    results[folder["name"]],time_interval = _record_time_for_folder_rename(mount_point,folder,num_samples)
    time_interval_for_vm_metrics[folder["name"]]=[time_interval[0][0],time_interval[-1][-1]]

  nested_folder={
      "name": dir["nested_folders"]["folder_name"]
  }
  results[dir["nested_folders"]["folder_name"]],time_interval = _record_time_for_folder_rename(mount_point,nested_folder,num_samples)
  time_interval_for_vm_metrics[dir["nested_folders"]["folder_name"]]=[time_interval[0][0],time_interval[-1][-1]]
  return results,time_interval_for_vm_metrics


def _perform_testing(dir, test_type, num_samples):
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
  """
  if test_type == "hns":
    # Creating config file for mounting with hns enabled.
    with open("/tmp/config.yml",'w') as mount_config:
      mount_config.write("enable-hns: true")
    mount_flags="--config-file=/tmp/config.yml --stackdriver-export-interval=30s"
  else :
    mount_flags = "--implicit-dirs --rename-dir-limit=1000000 --stackdriver-export-interval=30s"

  # Mounting the gcs bucket.
  bucket_name = mount_gcs_bucket(dir["name"], mount_flags, log)
  # Record time of operation and populate the results dict.
  results,time_intervals = _record_time_of_operation(bucket_name, dir, num_samples)
  # Unmounting the bucket.
  unmount_gcs_bucket(dir["name"], log)

  return results,time_intervals


def _parse_arguments(argv):
  argv = sys.argv
  parser = argparse.ArgumentParser()

  parser.add_argument(
      'config_file',
      help='Provide path of the config file for GCS bucket.',
      action='store',
  )
  parser.add_argument(
      'bucket_type',
      help='Provide bucket type - hns or flat ',
      action='store',
      choices=['hns','flat']
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


def _extract_vm_metrics(time_intervals_list,folders_list):
  """
  Function to extract the VM metrics given the timestamps from the rename operations.
  Args:
     time_intervals_list : List of time intervals for each folder on which rename
      operation is performed, each interval contains [ start time for the first
      sample , end time of the last sample]
     folders_list : List of names of folders on which the rename operation is done.
  Returns:
      vm_metrics_data : Dictionary of  VM metrics. For examples:
      {
        'folder_name': [CPU_UTI_PEAK, CPU_UTI_MEAN,REC_BYTES_PEAK, REC_BYTES_MEAN,
        SENT_BYTES_PEAK,SENT_BYTES_MEAN,OPS_ERROR_COUNT,MEMORY_USAGE_PEAK,
        MEMORY_USAGE_MEAN,LOAD_AVG_OS_THREADS_MEAN]
      }
  """
  vm_metrics_obj = vm_metrics.VmMetrics()
  vm_metrics_data = {}

  for folder in folders_list:
    start_time = time_intervals_list[folder][0]
    end_time = time_intervals_list[folder][1]
    vm_metrics_data[folder] = vm_metrics_obj.fetch_metrics(start_time,
                                                                   end_time,
                                                                   INSTANCE,
                                                                   PERIOD_SEC,
                                                                   'rename')[0]

  return vm_metrics_data


def _get_upload_value_for_vm_metrics(vm_metrics):
  """
  Function to take input of dictionary of Vm metrics and returns a list of values
  which can be uploaded to google sheet.
  Args:
    vm_metrics:  Dictionary of Vm metrics corresponding to each folder renamed.
  Returns:
    upload_values: List of values to upload .For example:
    ['folder_name',CPU_UTI_PEAK, CPU_UTI_MEAN,REC_BYTES_PEAK,...etc.]
  """
  upload_values = []
  for key, values in vm_metrics.items():
    row = [key] + values
    upload_values.append(row)
  return upload_values


def _run_rename_benchmark(test_type,dir_config,num_samples,upload_gs):
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

  # Getting latency related metrics
  results,time_intervals=_perform_testing(dir_str, test_type, num_samples)
  parsed_metrics = _parse_results(dir_str, results, num_samples)
  upload_values = _get_values_to_export(dir_str, parsed_metrics,
                                             test_type)

  print('Waiting for 360 seconds for metrics to be updated on VM...')
  # It takes up to 240 seconds for sampled data to be visible on the VM metrics graph
  # So, waiting for 360 seconds to ensure the returned metrics are not empty.
  # Intermittently custom metrics are not available after 240 seconds, hence
  # waiting for 360 secs instead of 240 secs
  time.sleep(360)

  # Getting VM related metrics
  folders_list=[]
  for folder in dir_str["folders"]["folder_structure"]:
    folders_list.append(folder["name"])
  folders_list.append(dir_str["nested_folders"]["folder_name"])

  vm_metrics_data= _extract_vm_metrics(time_intervals,folders_list)
  upload_values_vm_metrics= _get_upload_value_for_vm_metrics(vm_metrics_data)

  if upload_gs:
    log.info('Uploading files to the Google Sheet\n')
    if test_type == "flat":
      worksheet= WORKSHEET_NAME_FLAT
      vm_worksheet= WORKSHEET_VM_METRICS_FLAT
    else:
      worksheet= WORKSHEET_NAME_HNS
      vm_worksheet= WORKSHEET_VM_METRICS_HNS

    exit_code = _upload_to_gsheet(worksheet, upload_values,vm_worksheet,upload_values_vm_metrics,
                                  SPREADSHEET_ID)
    if exit_code != 0 :
      log.error("Upload to gsheet failed!")
  else:
    print('Latency related metrics: {}'.format(upload_values))
    print('VM metrics: {}'.format(upload_values_vm_metrics))


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 3:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 renaming_benchmark.py  [--upload_gs] [--num_samples NUM_SAMPLES] config_file bucket_type')

  args = _parse_arguments(argv)
  check_dependencies(['gcloud', 'gcsfuse'], log)
  _run_rename_benchmark(args.bucket_type, args.config_file, args.num_samples,
                          args.upload_gs)
