# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Python script for fetching and printing experiment configuration ID.

This python script calls the bigquery module with the experiment details and inserts configuration
if it doesn't exist and only updates end_date in case of existing configuration and gets the configuration ID

Typical usage example from bigquery folder:
  $ python3 get_experiments_config.py [-h] [--gcsfuse_flags GCSFUSE_FLAGS] [--branch BRANCH] [--end_date END_DATE] [--config_name CONFIG_NAME]

  Flag -h: Typical help interface of the script.
  Flag gcsfuse_flags (required: str): Set of gcsfuse flags used for experiment.
  Flag branch (required: str): GCSFuse repo branch used for building GCSFuse.
  Flag end_date (required: timestamp): Date till when experiments of this configuration are run.
  Flag config_name (required: str): Name of the experiment configuration.

Note: BigQuery API should be enabled for the project
"""
import bigquery
import constants
import argparse
import sys

def parse_arguments(argv):
  """Parses the arguments provided to the script via command line.

  Args:
    argv: List of arguments received by the script.

  Returns:
    A class containing the parsed arguments.
  """
  argv = sys.argv
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--gcsfuse_flags',
      help='Set of GCSFuse flags.',
      action='store',
      nargs=1,
      required=True
  )
  parser.add_argument(
      '--branch',
      help='GCSFuse repo branch used for building GCSFuse.',
      action='store',
      nargs=1,
      required=True
  )
  parser.add_argument(
      '--end_date',
      help='Date upto when tests are run.',
      action='store',
      nargs=1,
      required=True
  )
  parser.add_argument(
      '--config_name',
      help='Name of the experiment configuration.',
      action='store',
      nargs=1,
      required=True
  )
  return parser.parse_args(argv[1:])

if __name__ == '__main__':
  argv = sys.argv
  args = parse_arguments(argv)
  bigquery_obj = bigquery.ExperimentsGCSFuseBQ(constants.PROJECT_ID, constants.DATASET_ID)
  exp_config_id = bigquery_obj.get_experiment_configuration_id(args.gcsfuse_flags[0], args.branch[0], args.end_date[0], args.config_name[0])
  print(exp_config_id)
