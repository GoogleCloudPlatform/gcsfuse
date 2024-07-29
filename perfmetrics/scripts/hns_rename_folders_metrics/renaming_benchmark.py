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
"""Python script for benchmarking renaming operation.

This python script benchmarks and compares the latency of rename folder operations
using existing renamedir API vs hns supported rename folder API.

Note: This python script is dependent on generate_folders_and_files.py.
"""

import argparse
import sys
import json
import logging
import subprocess
import generate_folders_and_files

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()


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
  For testing renameFolder API,we mount using config-file flag which contains
  enable-hns set to true
  For testing renameDir API, we mount using implicit-dirs flag.

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


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 renaming_benchmark.py <config_file> [--keep_files]')

  parser = argparse.ArgumentParser()
  parser.add_argument(
      'dir_config',
      help='Provide path of the directory structure config file', )
  parser.add_argument(
      '--num_samples',
      help='Number of samples to collect of each test.',
      action='store',
      nargs=1,
      default=[10],
      required=False,
  )

  args = parser.parse_args(argv[1:])

  _check_dependencies(['gcloud', 'gcsfuse'])

  directory_structure=json.load(open(args.dir_config))

  # Consistency checks for the input JSON file.
  if generate_folders_and_files.check_for_config_file_inconsistency(directory_structure):
    log.error("Inconsistency exists in the input JSON file.")
    subprocess.call('bash',shell=True)

  # Ensure directory structure exists in the bucket before running tests.
  # In case of mistmatch, run generate_folders_and_files.py script with the
  # input config file to get the desired structure.
  if not generate_folders_and_files.check_if_dir_structure_exists(directory_structure):
    log.error("Mismatch exists between passed dir structure json file and existing"
              "structure in the GCS bucket.")
    subprocess.call('bash',shell=True)
