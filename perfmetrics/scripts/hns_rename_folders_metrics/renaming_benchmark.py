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
from utils.mount_unmount_util import mount_gcs_bucket, unmount_gcs_bucket
from utils.checks_util import check_dependencies

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()


def _extract_folder_name_prefix(folder_name) -> (str):
  """
  Extract the prefix from folder_name_for_ease of rename using regex matching.
  Example: filename_0 must get renamed to filename_1.This function returns
  the prefix excluding the iteration i.e. in this case , filename_ .

  Args:
    folder_name: String representing folder name.

  Returns:
    Prefix of folder_name which excludes the iteration.
  """
  try:
    folder_prefix = re.search("(?s:.*)\_", folder_name).group()
    return folder_prefix
  except:
    log.error("Folder name format is incorrect. Must be in the format prefix_0 \
          to begin.Exiting...")
    subprocess.call('bash', shell=True)


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
  folder_prefix = _extract_folder_name_prefix(folder["name"])
  folder_path_prefix = '{}/{}'.format(mount_point, folder_prefix)
  time_op = []
  for iter in range(num_samples):
    # For the first half of iterations, iteration value being appended to the
    # folder_prefix increases i.e. the rename operations look like:
    # folder_name_0 to folder_name_1 .
    if iter < num_samples / 2:
      rename_from = '{}{}'.format(folder_path_prefix, iter)
      rename_to = '{}{}'.format(folder_path_prefix, iter + 1)
    # For the second half of iterations, iteration value being appended to the
    # folder_prefix decreases,so that after the tests are complete , the folder
    # names are restored back to original i.e. the rename operations look like:
    # folder_name_1 to folder_name_0 .
    else:
      rename_from = '{}{}'.format(folder_path_prefix, num_samples - iter)
      rename_to = '{}{}'.format(folder_path_prefix, num_samples - iter - 1)
    start_time_sec = time.time()
    subprocess.call('mv ./{} ./{}'.format(rename_from, rename_to), shell=True)
    end_time_sec = time.time()
    time_op.append(end_time_sec - start_time_sec)

  return time_op


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
  if len(argv) < 4:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 renaming_benchmark.py  [--upload_gs] [--num_samples NUM_SAMPLES] config_file ')

  args = _parse_arguments(argv)
  check_dependencies(['gsutil', 'gcsfuse'], log)

  with open(os.path.abspath(args.dir_config_file)) as file:
    dir_str = json.load(file)

  # The script requires the num of samples to be even in order to restore test
  # data to original state after the tests are complete.
  if args.num_samples % 2 != 0:
    log.error("Only even number of samples allowed to restore the test data to\
                original state at the end of test.")
    subprocess.call('bash', shell=True)
