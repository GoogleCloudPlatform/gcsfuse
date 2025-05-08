# Copyright 2025 Google LLC
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

"""Python module for setting up the dataset and tables in BigQuery.

This python module creates the dataset and the table that will store fio
workload
configurations and metrics data in BigQuery. It can also be used to upload data
to the tables.

Note:
  Make sure BigQuery API is enabled for the project
"""

import argparse
import os
import socket
import sys
import time
import uuid

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../'))

from google.cloud import bigquery
from google.cloud.bigquery import table
from google.cloud.bigquery.job import QueryJob

"""Constants for bigquery table."""

DEFAULT_PROJECT_ID = 'gcs-fuse-test-ml'
DEFAULT_DATASET_ID = 'gke_test_tool_outputs'
DEFAULT_TABLE_ID = 'fio_outputs'


""" Timestamp is a new data-type to represent Timestamp
values in string form."""


class Timestamp:

  def __init__(self, val: str):
    self.val = val


"""FIO_TABLE_ROW_SCHEMA specifies the names of the fields and the order in which they are columns in the BQ table."""
FIO_TABLE_ROW_SCHEMA = [
    'fio_workload_id',
    'experiment_id',
    'epoch',
    'operation',
    'file_size',
    'file_size_in_bytes',
    'block_size',
    'block_size_in_bytes',
    'num_threads',
    'files_per_thread',
    'bucket_name',
    'machine_type',
    'gcsfuse_mount_options',
    'start_time',
    'end_time',
    'start_epoch',
    'end_epoch',
    'duration_in_seconds',
    'lowest_cpu_usage',
    'highest_cpu_usage',
    'lowest_memory_usage',
    'highest_memory_usage',
    'pod_name',
    'scenario',
    'e2e_latency_ns_max',
    'e2e_latency_ns_p50',
    'e2e_latency_ns_p90',
    'e2e_latency_ns_p99',
    'e2e_latency_ns_p99_9',
    'iops',
    'throughput_in_mbps',
]


class FioTableRow:
  """Class containing all the fields of the fio bigquery table as elements.

  This class represents the types and zero-values of all the fields/columns, and
  will also be handy to send data for inserting rows into this table.
  """

  def __init__(self):
    self.fio_workload_id = str('')
    self.experiment_id = str('')
    self.epoch = int(0)
    self.operation = str('')
    self.file_size = str('')
    self.file_size_in_bytes = int(0)
    self.block_size = str('')
    self.block_size_in_bytes = int(0)
    self.num_threads = int(0)
    self.files_per_thread = int(0)
    self.bucket_name = str('')
    self.machine_type = str('')
    self.gcsfuse_mount_options = str()
    self.start_time = Timestamp('')
    self.end_time = Timestamp('')
    self.start_epoch = int(0)
    self.end_epoch = int(0)
    self.duration_in_seconds = int(0)
    self.lowest_cpu_usage = float(0.0)
    self.highest_cpu_usage = float(0.0)
    self.lowest_memory_usage = float(0.0)
    self.highest_memory_usage = float(0.0)
    self.pod_name = str('')
    self.scenario = str('')
    self.e2e_latency_ns_max = float(0.0)
    self.e2e_latency_ns_p50 = float(0.0)
    self.e2e_latency_ns_p90 = float(0.0)
    self.e2e_latency_ns_p99 = float(0.0)
    self.e2e_latency_ns_p99_9 = float(0.0)
    self.iops = float(0.0)
    self.throughput_in_mbps = float(0.0)


def map_type_to_bq_type_str(t) -> str:
  if t == str:
    return 'STRING'
  elif t == int:
    return 'INT64'
  elif t == float:
    return 'FLOAT64'
  elif t == Timestamp:
    return 'TIMESTAMP'
  else:
    raise Exception(f'Unknown type: {t}')


class FioBigqueryExporter:
  """Class to create and interact with create/update Bigquery dataset and table for storing fio workload configurations and their output metrics.

  Attributes:
    project_id (str): The GCP project in which dataset and tables will be
      created
    dataset_id (str): The name of the dataset in the project that will store the
      tables
    table_name (str): The name of the bigquery table configurations and output
      metrics will be stored.
    bq_client (google.cloud.bigquery.client.Client): The client for interacting
      with Bigquery. Default value is bigquery.Client(project=project_id).
  """

  def __init__(self, project_id: str, dataset_id: str, table_name: str):
    self.client = bigquery.Client(project=project_id)
    self.project_id = project_id
    self.dataset_id = dataset_id
    self.table_name = table_name

    self._setup_dataset_and_tables()

  @property
  def dataset_ref(self):
    """Gets the reference of the dataset

    Returns:
      google.cloud.bigquery.dataset.Dataset: The retrieved dataset object
    """
    return self.client.get_dataset(self.dataset_id)

  def _get_table_from_table_name(self, table_name):
    """Gets the table from BigQuery from table ID

    Args:
      table_name (str): String representing the ID or name of the table

    Returns:
      google.cloud.bigquery.table.Table: The table in BigQuery
    """
    table_ref = self.dataset_ref.table(table_name)
    table = self.client.get_table(table_ref)
    return table

  def _execute_query(self, query) -> QueryJob:
    """Executes the query in BigQuery and raises an exception if query

       execution could not be completed.

    Args:
      query (str): Query that will be executed in BigQuery.

    Raises:
      Exception: If query execution failed.
    """
    job = self.client.query(query)
    if job.errors:
      for error in job.errors:
        raise Exception(f"Error message: {error['message']}")
    return job

  def _setup_dataset_and_tables(self):
    f"""
      Creates the dataset to store the tables and the experiment configuration table
      to store the configuration details and creates the
      {self.table_name} table to store the metrics.
    """
    # Create dataset if not exists
    dataset = bigquery.Dataset(f'{self.project_id}.{self.dataset_id}')
    try:
      self.client.create_dataset(dataset, exists_ok=True)
      print(
          f'Created dataset {dataset}, now sleeping for sometime to let it'
          ' reflect ...'
      )
      # Wait for the dataset to be created and ready to be referenced
      time.sleep(120)
    except Exception as e:
      print(f'Failed to create dataset {dataset}: {e}')
      raise

    # Query for creating fio_metrics table
    query_create_table_fio_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(""".format(
        self.project_id,
        self.dataset_id,
        self.table_name,
    )
    fio_table_header = FioTableRow()
    for field in FIO_TABLE_ROW_SCHEMA:
      bqFieldType = map_type_to_bq_type_str(
          type(getattr(fio_table_header, field))
      )
      query_create_table_fio_metrics += f'{field} {bqFieldType}, '
    query_create_table_fio_metrics += """) OPTIONS (description = 'Table for storing FIO metrics extracted from gke AI/ML tool.');
    """
    try:
      self._execute_query(query_create_table_fio_metrics)
    except Exception as e:
      print(f'Failed to create fio table {self.table_name}: {e}')
      raise


# The following functions are purely for testing.
# This file is a library to be used only for
# exporting the fio workload output metrics to bigquery.
def parse_arguments() -> object:
  parser = argparse.ArgumentParser(
      prog='',
      description=(),
  )
  parser.add_argument(
      '--project-id',
      metavar='GCP Project ID/name',
      help=(),
      default=DEFAULT_PROJECT_ID,
      required=False,
  )
  parser.add_argument(
      '--dataset-id',
      help='',
      default=DEFAULT_DATASET_ID,
      required=False,
  )
  parser.add_argument(
      '--table-name',
      help='Optional table name. Default=khregrh',
      default=DEFAULT_TABLE_ID,
      required=False,
  )
  return parser.parse_args()


if __name__ == '__main__':
  args = parse_arguments()

  fioBqExporter = FioBigqueryExporter(
      args.project_id, args.dataset_id, args.table_name
  )
