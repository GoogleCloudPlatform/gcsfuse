"""Python script for benchmarking listing operation.

This python script benchmarks and compares the latency of listing operation in
persistent disk vs GCS bucket. It creates the necessary directory structure,
containing files and folders, needed to test the listing operation. Furthermore,
it can optionally upload the results of the test to a Google Sheet or Bigquery. It takes
input a JSON config file whcih contains the info regrading directory structure
and also through which multiple tests of different configurations can be
performed in a single run. It also extracts VM metrics for folders '1KB_100000files_0subdir'
and '1KB_200000files_0subdir' in case of mount_type 'gcs_bucket'.

Typical usage example:
  $ python3 listing_benchmark.py [-h] [--keep_files] [--upload_gs] [--upload_bq] [--num_samples NUM_SAMPLES] [--message MESSAGE] [--config_id CONFIG_ID] [--start_time_build START_TIME_BUILD] --gcsfuse_flags GCSFUSE_FLAGS --command COMMAND config_file

  Flag -h: Typical help interface of the script.
  Flag --keep_files: Do not delete the generated directory structure from the
                     persistent disk after running the tests.
  Flag --upload_gs: Uploads the results of the test to the Google Sheet.
  Flag --upload_bq: Uploads the results of the test to the BigQuery.
  Flag --num_samples: Runs each test for NUM_SAMPLES times.
  Flag --message: Takes input a message string, which describes/titles the test.
  Flag --config_id: Configuration id of the experiment in BigQuery tables.
  Flag --start_time_build: Time at which KOKORO triggered the build scripts
  Flag --gcsfuse_flags (required): GCSFUSE flags with which the list tests bucket will be mounted.
  Flag --command (required): Takes as input a string, which is the command to run
  config_file (required): Path to the JSON config file which contains the
                          details of the tests.

Note: This python script is dependent on generate_files.py.
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
from gsheet import gsheet
import fetch_metrics
from bigquery import experiments_gcsfuse_bq
from vm_metrics import vm_metrics
from bigquery import constants

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()

WORKSHEET_NAME_GCS = 'ls_metrics_gcsfuse'
WORKSHEET_NAME_PD = 'ls_metrics_persistent_disk'
VM_METRICS_EXTRACTION_DIRS = ['1KB_100000files_0subdir', '1KB_200000files_0subdir']
MOUNT_TYPE_GCS = 'gcs_bucket'
MOUNT_TYPE_PD = 'persistent_disk'

def _count_number_of_files_and_folders(directory, files, folders):
  """Count the number of files and folders in the given directory recursively.
  Args:
    directory: protobuf of the directory.
    files: Number of files.
    folders: Number of folders.
  Returns:
    Number of files and folders encountered till now.
  """

  files += directory.num_files
  folders += directory.num_folders
  for folder in directory.folders:
    child_files, child_folders = _count_number_of_files_and_folders(folder, 0, 0)
    files += child_files
    folders += child_folders
  return files, folders


def _get_values_to_export(folders, metrics, command) -> list:
  """Returns values to export to Google Sheet and BigQuery.
  Args:
    folders: List containing protobufs of testing folders.
    metrics: A dictionary containing all the result metrics for each
             testing folder.
    command: Command to run the tests on.

  Returns:
    list: List of results to upload to GSheet or BigQuery
  """

  list_metrics_data = []
  for testing_folder in folders:
    num_files, num_folders = _count_number_of_files_and_folders(
        testing_folder, 0, 0)
    row = [
        metrics[testing_folder.name]['Test Desc.'],
        command,
        num_files,
        num_folders,
        metrics[testing_folder.name]['Number of samples'],
        metrics[testing_folder.name]['Mean'],
        metrics[testing_folder.name]['Median'],
        metrics[testing_folder.name]['Standard Dev'],
        metrics[testing_folder.name]['Quantiles']['0 %ile'],
        metrics[testing_folder.name]['Quantiles']['20 %ile'],
        metrics[testing_folder.name]['Quantiles']['50 %ile'],
        metrics[testing_folder.name]['Quantiles']['90 %ile'],
        metrics[testing_folder.name]['Quantiles']['95 %ile'],
        metrics[testing_folder.name]['Quantiles']['98 %ile'],
        metrics[testing_folder.name]['Quantiles']['99 %ile'],
        metrics[testing_folder.name]['Quantiles']['99.5 %ile'],
        metrics[testing_folder.name]['Quantiles']['99.9 %ile'],
        metrics[testing_folder.name]['Quantiles']['100 %ile']
    ]
    # Only some extracted metrics will be uploaded to Google Spreadsheets and BigQuery
    row_data_to_upload = row[1:3] + [row[4]] + [row[8]] + [row[17]] + row[5:8] + row[9:13]
    list_metrics_data.append(row_data_to_upload)

  return list_metrics_data


def _parse_results(folders, time_intervals_dict, message, num_samples) -> dict:
  """Outputs the results on the console.
  This function takes in dictionary containing the list of results (for all
  samples) for each testing folder, for both the gcs bucket and persistent disk.
  Then it generates various metrics out of these lists and outputs them into
  the console.
  The metrics present in the output are (in msec):
  Mean, Median, Standard Dev, 0th %ile, 20th %ile, 50th %ile, 90th %ile,
  95th %ile, 98th %ile, 99th %ile, 99.5th %ile, 99.9th %ile, 100th %ile.
  Args:
    folders: List containing protobufs of testing folders.
    time_intervals_dict: Dictionary containing the list of start and end times
                 (for all samples) for each testing folder.
    message: String which describes/titles the test.
    num_samples: Number of samples to collect for each test.
  Returns:
    A dictionary containing the various metrics in a JSON format.
  """

  # Dictionary containing the latencies (for all samples) for each testing folder
  factor_sec_to_msec = 1000
  latency_dict = {}
  for testing_folder in folders:
    # time[1] is the end time and time[0] is the start time for the samples
    latency_dict[testing_folder.name] = [(time[1] - time[0])*factor_sec_to_msec for time in time_intervals_dict[testing_folder.name]]

  metrics = dict()

  for testing_folder in folders:
    metrics[testing_folder.name] = dict()
    metrics[testing_folder.name]['Test Desc.'] = message
    metrics[testing_folder.name]['Number of samples'] = num_samples

    # Sorting based on time.
    latency_dict[testing_folder.name] = sorted(
        latency_dict[testing_folder.name])
    metrics[testing_folder.name]['Mean'] = round(
        stat.mean(latency_dict[testing_folder.name]), 3)
    metrics[testing_folder.name]['Median'] = round(
        stat.median(latency_dict[testing_folder.name]), 3)
    metrics[testing_folder.name]['Standard Dev'] = round(
        stat.stdev(latency_dict[testing_folder.name]), 3)

    metrics[testing_folder.name]['Quantiles'] = dict()
    sample_set = [0, 20, 50, 90, 95, 98, 99, 99.5, 99.9, 100]
    for percentile in sample_set:
      metrics[testing_folder.name]['Quantiles']['{} %ile'.format(percentile)] = round(
          np.percentile(latency_dict[testing_folder.name], percentile), 3)

  print(metrics)
  return metrics


def _record_time_of_operation(command, path, num_samples) -> list:
  """Runs the command on the given path for given num_samples times.

  Args:
    command: Command to run.
    path: Path at which to run the command.
    num_samples: Number of times to run the command.

  Returns:
    A list containing the start and end epoch times of operations.
  """

  time_intervals_list = []

  for _ in range(num_samples):
    start_time_sec = time.time()
    subprocess.call('{} {}'.format(command, path), shell=True,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.STDOUT)
    end_time_sec = time.time()
    time_intervals_list.append([start_time_sec, end_time_sec])

  return time_intervals_list

def _perform_testing(
    folders, gcs_bucket, persistent_disk, num_samples, command):
  """This function tests the listing operation on the testing folders.

  Going through all the testing folders one by one for both GCS bucket and
  persistent disk, we calculate the latency (in msec) of listing operation
  and store the time intervals in a list of that particular testing folder. Reading
  are taken multiple times as specified by num_samples argument.

  Args:
    folders: List of protobufs containing the testing folders.
    gcs_bucket: Name of the directory to which GCS bucket is mounted to.
    persistent_disk: Name of the directory in persistent disk containing all
                     the testing folders.
    num_samples: Number of times to run each test.
    command: Command to run the test on.

  Returns:
    gcs_bucket_time_intervals: A dictionary containing the list of start and end
                        times (all samples) for each testing folder.
    persistent_disk_time_intervals: A dictionary containing the list of start and
                             end times (all samples) for each testing folder.
  """

  gcs_bucket_time_intervals = {}
  persistent_disk_time_intervals = {}

  for testing_folder in folders:
    log.info('Testing started for testing folder: %s\n', testing_folder.name)
    local_dir_path = './{}/{}/'.format(persistent_disk, testing_folder.name)
    gcs_bucket_path = './{}/{}/'.format(gcs_bucket, testing_folder.name)

    persistent_disk_time_intervals[testing_folder.name] = _record_time_of_operation(
        command, local_dir_path, num_samples)
    # Sleep time given to get more accurate VM metrics
    if testing_folder.name in VM_METRICS_EXTRACTION_DIRS:
      time.sleep(60)
    gcs_bucket_time_intervals[testing_folder.name] = _record_time_of_operation(
        command, gcs_bucket_path, num_samples)

  log.info('Testing completed. Generating output.\n')
  return gcs_bucket_time_intervals, persistent_disk_time_intervals


def _create_directory_structure(
    gcs_bucket_url, persistent_disk_url, directory_structure, create_files_in_gcs) -> int:
  """Creates new directory structure using generate_files.py as a library.

  This function creates new directory structure in persistent disk. If
  create_files_in_gcs is True, then it also creates the same structure in
  GCS bucket. For more info regarding how the generation of files is happening,
  please read the generate_files.py.

  Args:
   gcs_bucket_url: Path of the directory in GCS bucket in which to create
                   the files.
   persistent_disk_url: Path in the persistent disk in which to create the
                        files in.
   directory_structure: Protobuf of the current directory.
   create_files_in_gcs: Bool value which is True if we have to create files
                        in GCS bucket (similar directory structure not present).
                        Otherwise, it is False, means that we will not create
                        files in GCS bucket from scratch.

  Returns:
    1 if an error is encountered while uploading files to the GCS bucket.
    0 if no error is encountered.
  """

  # Create a directory in the persistent disk.
  subprocess.call('mkdir {}'.format(persistent_disk_url), shell=True)

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
    result += _create_directory_structure(gcs_bucket_url + folder.name + '/',
                                          persistent_disk_url + folder.name + '/',
                                          folder, create_files_in_gcs)

  return int(result > 0)


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
  # directory in the first index. We don't want the current directory, so
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
    new_url = url + folder.name + '/'
    if new_url not in folders:
      return False
    result = result and _compare_directory_structure(new_url, folder)

  return result


def _unmount_gcs_bucket(gcs_bucket) -> None:
  """Unmounts the GCS bucket.

  Args:
    gcs_bucket: Name of the directory to which GCS bucket is mounted to.

  Raises:
    An error if the GCS bucket could not be umnounted and aborts the program.
  """

  log.info('Unmounting the GCS Bucket.\n')
  exit_code = subprocess.call('umount -l {}'.format(gcs_bucket), shell=True)
  if exit_code != 0:
    log.error('Error encountered in umounting the bucket. Aborting!\n')
    subprocess.call('bash', shell=True)
  else:
    subprocess.call('rm -rf {}'.format(gcs_bucket), shell=True)
    log.info(
        'Successfully unmounted the bucket and deleted %s directory.\n',
        gcs_bucket)


def _mount_gcs_bucket(bucket_name, gcsfuse_flags) -> str:
  """Mounts the GCS bucket into the gcs_bucket directory.

  Args:
    bucket_name: Name of the bucket to be mounted.
    gcsfuse_flags: Set of flags for which bucket_name will be mounted

  Returns:
    A string which contains the name of the directory to which the bucket
    is mounted.

  Raises:
    Aborts the program if error is encountered while mounting the bucket.
  """

  log.info('Started mounting the GCS Bucket using GCSFuse.\n')
  gcs_bucket = bucket_name
  subprocess.call('mkdir {}'.format(gcs_bucket), shell=True)

  exit_code = subprocess.call(
      'gcsfuse {} {} {}'.format(
          gcsfuse_flags, bucket_name, gcs_bucket), shell=True)
  if exit_code != 0:
    log.error('Cannot mount the GCS bucket due to exit code %s.\n', exit_code)
    subprocess.call('bash', shell=True)
  else:
    return gcs_bucket


def _parse_arguments(argv):
  """Parses the arguments provided to the script via command line.

  Args:
    argv: List of arguments received by the script.

  Returns:
    A class containing the parsed arguments.
  """

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
      '--upload_gs',
      help='Upload the results to the Google Sheet.',
      action='store_true',
      default=False,
      required=False,
  )
  parser.add_argument(
      '--upload_bq',
      help='Upload the results to the BigQuery.',
      action='store_true',
      default=False,
      required=False,
  )
  parser.add_argument(
      '--message',
      help='Puts a message/title describing the test.',
      action='store',
      nargs=1,
      default=['Performance Listing Benchmark'],
      required=False,
  )
  parser.add_argument(
      '--num_samples',
      help='Number of samples to collect of each test.',
      action='store',
      nargs=1,
      default=[300],
      required=False,
  )
  parser.add_argument(
      '--config_id',
      help='Configuration ID of the experiment.',
      action='store',
      nargs=1,
      required=False,
  )
  parser.add_argument(
      '--start_time_build',
      help='Start time of the build.',
      action='store',
      nargs=1,
      required=False,
  )
  parser.add_argument(
      '--command',
      help='Command to run the tests on.',
      action='store',
      nargs=1,
      default=['ls -R'],
      required=False,
  )
  parser.add_argument(
      '--gcsfuse_flags',
      help='Gcsfuse flags for mounting the list tests bucket. Example set of flags - "--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library --debug_fuse --debug_gcs --log-file gcsfuse-list-logs.txt --log-format \"text\" --stackdriver-export-interval=30s"',
      action='store',
      nargs=1,
      required=True,
  )
  # Ignoring the first parameter, as it is the path of this python
  # script itself.
  return parser.parse_args(argv[1:])

def _check_dependencies(packages) -> None:
  """Check whether the dependencies are installed or not.

  Args:
    packages: List containing the names of the dependencies to be checked.

  Raises:
    Aborts the execution if a particular dependency is not found.
  """

  for curr_package in packages:
    log.info('Checking whether %s is installed.\n', curr_package)
    exit_code = subprocess.call(
        '{} --version'.format(curr_package), shell=True)
    if exit_code != 0:
      log.error(
          '%s not installed. Please install. Aborting!\n', curr_package)
      subprocess.call('bash', shell=True)

  return

def _extract_vm_metrics(time_intervals_list, folders, mount_type) -> list:
  """Extracts VM metrics for testing folders '1KB_100000files_0subdir' and testing_folder.name
   for mount type 'gcs_bucket'

  Args:
    time_intervals_list (list): List containing the start and end times
                        (for all samples) for each testing folder.
    folders (list): List containing protobufs of testing folders.
    mount_type (str): Access type of folder. It can take 2 values:
                      'gcs_bucket' or 'persistent_disk'
  Returns:
    list: A list of extracted metrics
  """
  vm_metrics_obj = vm_metrics.VmMetrics()
  vm_metrics_data = {}
  # Getting VM metrics for listing tests on each folder
  for testing_folder in folders:
    # Getting start and end times for all 300 samples
    start_time_first_sample = time_intervals_list[testing_folder.name][0][0]
    end_time_last_sample = time_intervals_list[testing_folder.name][-1][-1]
    metrics_data = []

    if testing_folder.name in VM_METRICS_EXTRACTION_DIRS and mount_type == 'gcs_bucket':
      metrics_data = vm_metrics_obj.fetch_metrics(start_time_first_sample, end_time_last_sample,
                                                  fetch_metrics.INSTANCE, fetch_metrics.PERIOD_SEC, 'list')

    else:
      metrics_data = [[start_time_first_sample, end_time_last_sample, None, None, None, None]]
    vm_metrics_data[testing_folder.name] = metrics_data[0]

  return vm_metrics_data


def _export_to_gsheet(worksheet, ls_data):
  """Writes list results to Google Spreadsheets

  Args:
    worksheet (str): Google sheet name to which results will be uploaded
    ls_data (list): List results to be uploaded
  """
  # Changing directory to comply with "cred.json" path in "gsheet.py".
  os.chdir('..')

  gsheet.write_to_google_sheet(worksheet, ls_data)

  os.chdir('./ls_metrics')  # Changing the directory back to current directory.
  return


def _export_to_bigquery(test_type, config_id, start_time_build, ls_data):
  """Writes list results to BigQuery

  Args:
    test_type (str): Table name to which results will be uploaded
    config_id (str): Configuration ID of the experiment
    start_time_build (str): Start time of the build
    ls_data (list): List results to be uploaded
  """
  bigquery_obj = experiments_gcsfuse_bq.ExperimentsGCSFuseBQ(constants.PROJECT_ID, constants.DATASET_ID)
  ls_data_upload = [[test_type] + row for row in ls_data]
  bigquery_obj.upload_metrics_to_table(constants.LS_TABLE_ID, config_id, start_time_build, ls_data_upload)
  return


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 6:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 listing_benchmark.py [--keep_files] [--upload_gs] [--upload_bq] [--num_samples NUM_SAMPLES] [--message MESSAGE] [--config_id CONFIG_ID] [--start_time_build START_TIME_BUILD] --gcsfuse_flags GCSFUSE_FLAGS --command COMMAND config_file')

  args = _parse_arguments(argv)

  _check_dependencies(['gsutil', 'gcsfuse'])

  with open(os.path.abspath(args.config_file)) as file:
    config_json = json.load(file)
  directory_structure = ParseDict(config_json, directory_proto.Directory())

  log.info('Started checking the directory structure in the bucket.\n')
  directory_structure_present = _compare_directory_structure(
      'gs://{}/'.format(directory_structure.name), directory_structure)

  # Removing the already present folder in persistent disk to create the
  # files from scratch.
  persistent_disk = 'persistent_disk'
  if os.path.exists('./{}'.format(persistent_disk)):
    subprocess.call('rm -rf {}'.format(persistent_disk), shell=True)

  # If similar directory structure not found in the GCS bucket then delete all
  # the files in the bucket and make it from scratch.
  if not directory_structure_present:
    log.info(
        """Similar directory structure not found in the GCS bucket.
        Creating a new one.\n""")
    log.info('Deleting previously present directories in the GCS bucket.\n')
    subprocess.call(
        'gsutil -m rm -r gs://{}/*'.format(directory_structure.name),
        shell=True, stdout=subprocess.DEVNULL, stderr=subprocess.STDOUT)

  # Creating a temp directory which will be needed by the generate_files
  # method to create files in batches.
  temp_dir = generate_files.TEMPORARY_DIRECTORY
  if os.path.exists(os.path.dirname(temp_dir)):
    subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)
  subprocess.call('mkdir -p {}'.format(temp_dir), shell=True)

  exit_code = _create_directory_structure(
      'gs://{}/'.format(directory_structure.name),
      './{}/'.format(persistent_disk), directory_structure,
      not directory_structure_present)

  # Deleting the temp folder after the creation of files is done.
  subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)

  if exit_code != 0:
    log.error('Cannot create files in the GCS bucket. Error encountered.\n')
    subprocess.call('bash', shell=True)
  log.info('Directory Structure Created.\n')

  gcs_bucket = _mount_gcs_bucket(directory_structure.name, args.gcsfuse_flags[0])

  gcs_bucket_time_intervals, persistent_disk_time_intervals = _perform_testing(
      directory_structure.folders, gcs_bucket, persistent_disk,
      int(args.num_samples[0]), args.command[0])

  gcs_parsed_metrics = _parse_results(
      directory_structure.folders, gcs_bucket_time_intervals, args.message[0],
      int(args.num_samples[0]))
  pd_parsed_metrics = _parse_results(
      directory_structure.folders, persistent_disk_time_intervals, args.message[0],
      int(args.num_samples[0]))

  print('Waiting for 360 seconds for metrics to be updated on VM...')
  # It takes up to 240 seconds for sampled data to be visible on the VM metrics graph
  # So, waiting for 360 seconds to ensure the returned metrics are not empty.
  # Intermittently custom metrics are not available after 240 seconds, hence
  # waiting for 360 secs instead of 240 secs
  time.sleep(360)

  gcs_results = _extract_vm_metrics(gcs_bucket_time_intervals, directory_structure.folders, 'gcs_bucket')
  pd_results = _extract_vm_metrics(persistent_disk_time_intervals, directory_structure.folders, 'persistent_disk')

  for folder in directory_structure.folders:
    print(f'VM metrics for listing tests (gcs bucket) for folder: {folder.name}...')
    print(gcs_results[folder.name])

  gcs_results_vm = {}
  pd_results_vm = {}

  for folder in directory_structure.folders:
    gcs_results_vm[folder.name] = gcs_results[folder.name][2:]
    pd_results_vm[folder.name] = pd_results[folder.name][2:]

  upload_values_gcs = _get_values_to_export(directory_structure.folders, gcs_parsed_metrics, args.command[0])
  upload_values_pd = _get_values_to_export(directory_structure.folders, pd_parsed_metrics, args.command[0])

  results_gcs = []
  results_pd = []

  for folder, values_gcs, values_pd in zip(directory_structure.folders, upload_values_gcs, upload_values_pd):
    # Combining list metrics and VM metrics
    results_gcs.append([gcs_bucket_time_intervals[folder.name][0][0], gcs_bucket_time_intervals[folder.name][-1][-1]] + upload_values_gcs + gcs_results_vm[folder.name])
    results_gcs.append([persistent_disk_time_intervals[folder.name][0][0], persistent_disk_time_intervals[folder.name][-1][-1]] + upload_values_pd + pd_results_vm[folder.name])

  if args.upload_gs:
    log.info('Uploading files to the Google Sheet.\n')
    _export_to_gsheet(WORKSHEET_NAME_GCS, results_gcs)
    _export_to_gsheet(WORKSHEET_NAME_PD, results_pd)

  if args.upload_bq:
    if not args.config_id or not args.start_time_build:
      raise Exception("Pass required arguments experiments configuration ID and start time of build for uploading to BigQuery")
    log.info('Uploading results to the BigQuery.\n')
    _export_to_bigquery(MOUNT_TYPE_GCS, args.config_id[0], args.start_time_build[0], results_gcs)
    _export_to_bigquery(MOUNT_TYPE_PD, args.config_id[0], args.start_time_build[0], results_pd)

  if not args.keep_files:
    log.info('Deleting files from persistent disk.\n')
    subprocess.call('rm -rf {}'.format(persistent_disk), shell=True)

  _unmount_gcs_bucket(gcs_bucket)
