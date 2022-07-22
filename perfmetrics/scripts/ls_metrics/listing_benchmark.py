"""Python script for benchmarking listing operation.

This python script benchmarks and compares the latency of listing operation in
persistent disk vs GCS bucket. It creates the necessary directory structure,
containing files and folders, needed to test the listing operation. Furthermore it can
optionally upload the results of the test to a Google Sheet. It takes input a
JSON config file whcih contains the info regrading directory structure and also through
which multiple tests of different configurations can be performed in a single run.

Typical usage example:
  $ python3 listing_benchmark.py [-h] [--keep_files] [--upload] [--num_samples NUM_SAMPLES] [--message MESSAGE] --command COMMAND config_file

  Flag -h: Typical help interface of the script.
  Flag --keep_files: Do not delete the generated directory structure from the persistent disk after running the tests.
  Flag --upload: Uploads the results of the test to the Google Sheet.
  Flag --num_smaples NUM_SAMPLES: Runs each test for NUM_SAMPLES times.
  Flag --message MESSAGE: Takes input a message string, which describes/titles the test.
  Flag --command COMMAND (required): Takes a input a string, which is the command to run the tests on.
  config_file (required): Path to the JSON config file which contains the details of the tests.

Note: This python script is dependent on generate_files.py.
"""

import argparse
import ast
import configparser
import json
import logging
import os
import statistics as stat
import subprocess
import sys
import time

import directory_pb2 as directory_proto

sys.path.insert(0, '..')
import generate_files
import numpy as np
import texttable

from google.protobuf.json_format import Parse, ParseDict


logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()


def OutputResults(folders, gcs_bucket_results, persistent_disk_results, message, num_samples) -> None:
  """Outputs the results on the console.

  This function takes in dictionary containing the list of results (for all samples) for each
  testing folder, for both the gcs bucket and persistent disk. Then it generates various metrics
  out of these lists and outputs them into the console. It further uses the texttable library to
  output the results in forms of tables. A single row of the table represents a single test(testing folder).
  For every test a side by side comparision is made for the GCS bucket and persistent disk.
  Also a quantile comparision is shown in the table.

  The metrics present in the output are (in msec):
  Mean, Median, Standard Dev, 0th %ile, 20th %ile, 40th %ile, 60th %ile, 80th %ile, 90th %ile,
  95th %ile, 98th %ile, 99th %ile, 100th %ile.

  Args:
    folders: List containing protobufs of testing folders.
    gcs_bucket_results: Dictionary containing the list of results (for all samples)
                        for each testing folder in the GCS bucket.
    persistent_disk_results: Dictionary containing the list of results (for all samples)
                             for each testing folder in the persistent disk.
    message: String which describes/titles the test.
    num_samples: Number of samples to collect for each test.
  """
  log.info('Showing Results of %s samples:\n', num_samples)

  print('Test Description:  {}\n'.format(message))

  main_table_contents = []
  main_table_contents.append(['Testing Folder', 'GCS Bucket Results', 'Persistent Disk Results', 'Quantile Comparision'])

  for testing_folder in folders:
    gcs_bucket_results[testing_folder.name] = sorted(gcs_bucket_results[testing_folder.name])
    persistent_disk_results[testing_folder.name] = sorted(persistent_disk_results[testing_folder.name])

    gcs_bucket_subtable = texttable.Texttable()
    gcs_bucket_subtable.set_deco(texttable.Texttable.HEADER)
    gcs_bucket_subtable.set_cols_dtype(['t', 'f'])
    gcs_bucket_subtable.set_cols_align(['l', 'r'])
    gcs_bucket_subtable.add_rows([
        ['Statistic', 'Value (msec)'],
        ['Mean', stat.mean(gcs_bucket_results[testing_folder.name])],
        ['Median', stat.median(gcs_bucket_results[testing_folder.name])],
        ['Standard Dev', stat.stdev(gcs_bucket_results[testing_folder.name])],
    ])

    persistent_disk_subtable = texttable.Texttable()
    persistent_disk_subtable.set_deco(texttable.Texttable.HEADER)
    persistent_disk_subtable.set_cols_dtype(['t', 'f'])
    persistent_disk_subtable.set_cols_align(['l', 'r'])
    persistent_disk_subtable.add_rows([
        ['Statistic', 'Value (msec)'],
        ['Mean', stat.mean(persistent_disk_results[testing_folder.name])],
        ['Median', stat.median(persistent_disk_results[testing_folder.name])],
        ['Standard Dev', stat.stdev(persistent_disk_results[testing_folder.name])],

    ])

    quantiles_gcs = []
    for percentile in range(0, 100, 20):
      quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder.name], percentile))

    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder.name], 90))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder.name], 95))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder.name], 98))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder.name], 99))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder.name], 100))

    quantiles_pd = []
    for percentile in range(0, 100, 20):
      quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder.name], percentile))

    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder.name], 90))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder.name], 95))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder.name], 98))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder.name], 99))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder.name], 100))

    quantile_table = texttable.Texttable()
    quantile_table.set_deco(texttable.Texttable.HEADER)
    quantile_table.set_cols_dtype(['t', 'f', 'f'])
    quantile_table.set_cols_align(['l', 'r', 'r'])
    quantile_table.add_rows([
        ['Quantile', 'GCS Bucket', 'Persistent Disk'],
        ['0th %ile', quantiles_gcs[0], quantiles_pd[0]],
        ['20th %ile', quantiles_gcs[1], quantiles_pd[1]],
        ['40th %ile', quantiles_gcs[2], quantiles_pd[2]],
        ['60th %ile', quantiles_gcs[3], quantiles_pd[3]],
        ['80th %ile', quantiles_gcs[4], quantiles_pd[4]],
        ['90th %ile', quantiles_gcs[5], quantiles_pd[5]],
        ['95th %ile', quantiles_gcs[6], quantiles_pd[6]],
        ['98th %ile', quantiles_gcs[7], quantiles_pd[7]],
        ['99th %ile', quantiles_gcs[8], quantiles_pd[8]],
        ['100th %ile', quantiles_gcs[9], quantiles_pd[9]]
    ])

    main_table_contents.append(['{}'.format(testing_folder.name),
                                gcs_bucket_subtable.draw(),
                                persistent_disk_subtable.draw(),
                                quantile_table.draw()])

  main_table = texttable.Texttable()
  main_table.set_cols_align(['l', 'c', 'c', 'c'])
  main_table.set_cols_valign(['m', 'm', 'm', 'm'])
  main_table.set_cols_width([30, 35, 35, 60])
  main_table.add_rows(main_table_contents)

  print(main_table.draw())
  print('')


def RecordTimeOfOperation(command, path, num_samples) -> list:
  """Runs the command on the given path for given num_samples times.

  Args:
    command: Command to run.
    path: Path at which to run the command.
    num_samples: Number of times to run the command.

  Returns:
    A list containing the latencies of operations in milisecond.
  """
  result_list = []
  for _ in range(num_samples):
    st = time.time()
    subprocess.call('{} {}'.format(command, path), shell=True,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.STDOUT)
    et = time.time()
    result_list.append((et-st)*1000)
  return result_list


def Testing(folders, gcs_bucket, persistent_disk, num_samples, command):
  """This function tests the listing operation on the testing folders.

  Going through all the testing folders one by one for both GCS bucket and
  peristent disk, we calculate the latency (in msec) of listing operation
  and store the results in a list of that particular testing folder. Reading
  are taken multiple times as specified by num_samples argument.

  Args:
    folders: List of protobufs containing the testing folders.
    gcs_bucket: Name of the directory to which GCS bucket is mounted to.
    persistent_disk: Name of the directory in persistent disk containing all the testing folders.

  Returns:
    gcs_bucket_results: A dictionary containing the list of results (all samples)
                        for each testing folder.
    persistent_disk_results: A dictionary containing the list of results (all samples)
                             for each testing folder.
  """
  gcs_bucket_results = {}
  persistent_disk_results = {}

  for testing_folder in folders:
    log.info('Testing started for testing folder: %s\n', testing_folder.name)

    local_dir_path = './{}/{}/'.format(persistent_disk, testing_folder.name)
    gcs_bucket_path = './{}/{}/'.format(gcs_bucket, testing_folder.name)

    persistent_disk_results[testing_folder.name] = RecordTimeOfOperation(command, local_dir_path, num_samples)
    gcs_bucket_results[testing_folder.name] = RecordTimeOfOperation(command, gcs_bucket_path, num_samples)

  log.info('Testing completed. Generating output.\n')
  return gcs_bucket_results, persistent_disk_results


def CreateDirectoryStructure(gcs_bucket_url, persistent_disk_url, directory_structure, create_files_in_gcs) -> int:
  """Creates new directory structure using generate_files.py as a library.

  This function creates new directory structure in persistent disk. If create_files_in_gcs
  is True, then it also creates the same structure in GCS bucket.
  For more info regarding how the generation of files is happening, please read the generate_files.py.

  Args:
   gcs_bucket_url: Path of the directory in GCS bucket in which to create the files.
   persistent_disk_url: Path in the persistent disk in which to create the files in.
   directory_structure: Protobuf of the current directory.
   create_files_in_gcs: Bool value which is True if we have to create files in GCS bucket (similar directory
                        strucutre not present). Otherwise it is False, means that we will not
                        create files in GCS bucket from scratch.

  Returns:
    1 if an error is encountered while uploading files to the GCS bucket. 0 if no error is encountered.
  """
  # Create a directory in the persistent disk.
  subprocess.call('mkdir {}'.format(persistent_disk_url), shell=True)

  if directory_structure.num_files != 0:
    file_size = int(directory_structure.file_size[:-2])
    file_size_unit = directory_structure.file_size[-2:]
    exit_code = generate_files.generate_files_and_upload_to_gcs_bucket(gcs_bucket_url,
                                                                       directory_structure.num_files,
                                                                       file_size_unit, file_size,
                                                                       directory_structure.file_name_prefix,
                                                                       persistent_disk_url, create_files_in_gcs,
                                                                       'return', log)
    if exit_code != 0:
      return 1

  result = 0
  for folder in directory_structure.folders:
    result += CreateDirectoryStructure(gcs_bucket_url + folder.name + '/',
                                       persistent_disk_url + folder.name + '/',
                                       folder, create_files_in_gcs)

  return int(result > 0)


def ListDirectory(path) -> (list, int):
  """Returns the list containing path of all the contents present in the current directory and also the number of contents.

  Args:
    path: Path of the directory.

  Returns:
    A list containing path of all contents present in the input path. Also returns the number of contents.
  """
  contents = subprocess.check_output('gsutil -m ls {}'.format(path), shell=True)
  num_contents = len(contents.decode('utf-8').split('\n')[:-1])
  contents_url = contents.decode('utf-8').split('\n')[:-1]
  return contents_url, num_contents


def CompareDirectoryStructure(url, directory_structure) -> bool:
  """Compares the directory structure present in the GCS bucket with the structure present in the JSON config file.

  This function checks whether a same directory structure is present in the GCS bucket or not. If a same structure
  is already present then we dont have to make the whole structure from scratch everytime. This saves a lot of time
  as writing to the GCS bucket is a time expensive operation.

  Args:
    url: Path of the directory to compare in the GCS bucket.
    directory_structure: Protobuf of the current directory.

  Returns:
    A boolean value which is true if the similar directory structure is already present in the GCS bucket.
    Otherwise returns a false and we have to create the directory structure from scratch in the GCS bucket.
  """
  contents_url, num_contents = ListDirectory(url)

  # gsutil in some cases return the contents_url list with the current directory in
  # the first index. We dont want the current directory so we remove it
  # manually.
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
    result = result and CompareDirectoryStructure(new_url, folder)

  return result


def UnmountGCSBucket(gcs_bucket) -> None:
  """Unmounts the GCS bucket.

  Args:
    gcs_bucket: Name of the directory to which GCS bucket is mounted to.

  Raises:
    An error if the GCS bucket cound not be umnounted.
  """
  log.info('Unmounting the GCS Bucket.\n')

  exit_code = subprocess.call('umount -l {}'.format(gcs_bucket), shell=True)

  if exit_code != 0:
    log.error('Error encountered in umounting the bucket. Aborting!\n')
    subprocess.call('bash', shell=True)

  else:
    subprocess.call('rm -rf {}'.format(gcs_bucket), shell=True)
    log.info('Successfully unmounted the bucket and deleted %s directory.\n', gcs_bucket)


def MountGCSBucket(bucket_name) -> str:
  """Mounts the GCS bucket into the gcs_bucket directory.

  Args:
    bucket_name: Name of the bucket to be mounted.

  Returns:
    A string which contains the name of the directory to which the bucket is mounted.

  Raises:
    An error into the logs if cannot mount the bucket due to some error. Also shut offs the program.
  """
  log.info('Started mounting the GCS Bucket using GCSFuse.\n')
  gcs_bucket = bucket_name

  subprocess.call('mkdir {}'.format(gcs_bucket), shell=True)

  exit_code = subprocess.call('gcsfuse --implicit-dirs --disable-http2 --max-conns-per-host 100 {} {}'.format(
      bucket_name, gcs_bucket), shell=True)
  print('')

  if exit_code != 0:
    log.error('Cannot mount the GCS bucket due to exit code %s.\n', exit_code)
    subprocess.call('bash', shell=True)
  else:
    return gcs_bucket


def ParseArguments(argv):
  """Parses the arguments provided to the script via command line.

  Args:
    argv: List of arguments recevied by the script.

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
      '--upload',
      help='Upload the results to the Google Sheet.',
      action='store_true',
      default=False,
      required=False,
  )

  parser.add_argument(
      '--message',
      help='Puts a message/title describing the test.',
      action='store',
      nargs=1,
      default='Performance Listing Benchmark',
      required=False,
  )

  parser.add_argument(
      '--num_samples',
      help='Number of samples to collect of each test.',
      action='store',
      nargs=1,
      default=10,
      required=False,
  )

  parser.add_argument(
      '--command',
      help='Command to run the tests on.',
      action='store',
      nargs=1,
      default='ls -R',
      required=True,
  )

  args = parser.parse_args(argv[1:])

  return args


def CheckDependencies(packages) -> None:
  """Check whether the dependencies are installed or not.

  Args:
    packages: List containing the names of the dependencies to be checked.

  Raises:
    An error into the logs if a particular dependency is not installed and shuts of the program.
  """
  for curr_package in packages:
    log.info('Checking whether %s is installed.\n', curr_package)

    exit_code = subprocess.call(
        '{} --version'.format(curr_package), shell=True)
    print('')

    if exit_code != 0:
      log.error('%s not installed. Please install. Aborting!\n', curr_package)
      subprocess.call('bash', shell=True)

  return


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 3:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 listing_benchmark.py [--keep_files] [--upload] [--num_samples NUM_SAMPLES] [--message MESSAGE] --command COMMAND config_file')

  args = ParseArguments(argv)

  CheckDependencies(['gsutil', 'gcsfuse'])

  f = open(os.path.abspath(args.config_file))
  config_json = json.load(f)
  f.close()

  directory_structure = ParseDict(config_json, directory_proto.Directory())

  gcs_bucket = MountGCSBucket(directory_structure.name)

  log.info('Started checking the directory structure in the bucket.\n')
  directory_structure_present = CompareDirectoryStructure('gs://{}/'.format(directory_structure.name),
                                                                            directory_structure)
  if not directory_structure_present:
    log.info('Similar directory structure not found in the GCS bucket. Creating a new one.\n')
  else:
    log.info('Similar directory already present in the bucket. Not making a new one.\n')

  # Removing the already present folder in persistent disk so as to create the
  # files from scratch.
  persistent_disk = 'persistent_disk'
  if os.path.exists('./{}'.format(persistent_disk)):
    subprocess.call('rm -rf {}'.format(persistent_disk), shell=True)

  # If similar directory structure not found in the GCS bucket then delete all
  # the files in the bucket and make it from scratch.
  if not directory_structure_present:
    log.info('Deleting previously present directories in the GCS bucket.\n')
    subprocess.call('gsutil -m rm -r gs://{}/*'.format(directory_structure.name),
                    shell=True, stdout = subprocess.DEVNULL, stderr = subprocess.STDOUT)

  # Creating a temp directory which will be needed by the generate_files method to
  # create files in batches.
  temp_dir = generate_files.TEMPORARY_DIRECTORY
  if os.path.exists(os.path.dirname(temp_dir)):
    subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)
  subprocess.call('mkdir -p {}'.format(temp_dir), shell=True)

  exit_code = CreateDirectoryStructure('gs://{}/'.format(directory_structure.name), './{}/'.format(persistent_disk),
                                       directory_structure, not directory_structure_present)

  # Deleting the temp folder after the creation of files is done.
  subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)) ,shell=True)

  if exit_code != 0:
    UnmountGCSBucket(gcs_bucket)
    log.error('Cannot create files in the GCS bucket. Error encountered.\n')
    subprocess.call('bash', shell=True)

  log.info('Directory Structure Created.\n')

  gcs_bucket_results, persistent_disk_results = Testing(directory_structure.folders, gcs_bucket, persistent_disk,
                                                        args.num_samples, args.command[0])

  OutputResults(directory_structure.folders, gcs_bucket_results, persistent_disk_results, args.message, args.num_samples)

  if not args.keep_files:
    log.info('Deleting files from persistent disk.\n')
    subprocess.call('rm -rf {}'.format(persistent_disk), shell=True)

  UnmountGCSBucket(gcs_bucket)
