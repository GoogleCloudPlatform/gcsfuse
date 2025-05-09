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
workload and metrics data in BigQuery. It can also be used to upload data
to the tables.
"""

import argparse
import os
import socket
import sys
import time
import uuid

# Add relative path ../../../ for class ExperimentsGCSFuseBQ .
sys.path.append(os.path.join(os.path.dirname(__file__), '../../../'))

from google.cloud import bigquery
from google.cloud.bigquery import table
from google.cloud.bigquery.job import QueryJob
from bigquery.experiments_gcsfuse_bq import ExperimentsGCSFuseBQ


""" Timestamp is a new data-type to represent Timestamp
values in string form."""


class Timestamp:

  def __init__(self, val: str):
    self.val = val

  def __str__(self):
    return self.val


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


class FioBigqueryExporter(ExperimentsGCSFuseBQ):
  """Class to create and interact with create/update Bigquery dataset and table for storing fio workload configurations and their output metrics.

  Attributes:
    project_id (str): The GCP project in which dataset and tables will be
      created
    dataset_id (str): The name of the dataset in the project that will store the
      tables
    table_id (str): The name of the bigquery table configurations and output
      metrics will be stored.
    bq_client (google.cloud.bigquery.client.Client): The client for interacting
      with Bigquery. Default value is bigquery.Client(project=project_id).
  """

  def __init__(self, project_id: str, dataset_id: str, table_id: str):
    super().__init__(project_id, dataset_id)
    self.table_id = table_id

    self._setup_dataset_and_tables()

  def _setup_dataset_and_tables(self):
    f"""
      Creates the dataset to store the tables and the experiment configuration table
      to store the configuration details and creates the
      {self.table_id} table to store the metrics.
    """
    # Create dataset if not exists
    dataset = bigquery.Dataset(f'{self.project_id}.{self.dataset_id}')
    try:
      self.client.create_dataset(dataset, exists_ok=True)
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
        self.table_id,
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
      print(f'Failed to create fio table {self.table_id}: {e}')
      raise

  def _insert_rows_with_retry(self, table, rows_to_insert: []):
    """Inserts given rows to the given table in a single transaction.

    If the transaction fails, it parses the returned result variable, for the
    rows that failed and the reasons they failed. It then retries inserting
    the rest of the rows.

    This function is a wrapper over BQ client insert_rows function.

    Arguments:

    table: A BQ table handle.
    rows_to_insert: A list of tuples to insert rows into the above BQ table.

    Exceptions:
    When the retry transaction fails too, then it re-throws that exception.
    """
    result = self.client.insert_rows(table, rows_to_insert)
    if result:
      if isinstance(result, list):
        failed_row_indices = dict()
        for result_entry in result:
          if (
              isinstance(result_entry, dict)
              and 'index' in result_entry
              and 'errors' in result_entry
              and isinstance(result_entry['errors'], list)
          ):
            for result_entry_error in result_entry['errors']:
              if (
                  isinstance(result_entry_error, dict)
                  and 'reason' in result_entry_error
                  and result_entry_error['reason'] != 'stopped'
              ):
                failed_row_indices[result_entry['index']] = result_entry_error[
                    'message'
                ]
                break
        print(
            'The following rows failed to insert to BQ table: \n{}'.format([
                f'Row # {_failed_row_index}:'
                f' {rows_to_insert[_failed_row_index]}, error message:'
                f' {failed_row_indices[_failed_row_index]}\n'
                for _failed_row_index in failed_row_indices
            ])
        )
        print('Going to retry inserting the rest ...')
        row_indices_to_retry = set(range(len(rows_to_insert))) - (
            failed_row_indices.keys()
        )
        rows_to_retry = [rows_to_insert[i] for i in row_indices_to_retry]
        result = self.client.insert_rows(table, rows_to_retry)
        if result:
          raise Exception(f'{result}')
      else:
        raise Exception(f'{result}')

  def insert_rows(self, fioTableRows: []):
    """Pass a list of FioTableRow objects to insert into the fio-table.

    This inserts all the given rows of data in a single transaction.

    Arguments:

    fioTableRows: a list of FioTableRow objects.

    Raises:
      Exception: If some row insertion failed.
    """

    # Edge-case.
    if fioTableRows is None or len(fioTableRows) == 0:
      return

    # Create a list of tuples from the given list of FioTableRow objects.
    # Each tuple should have the values for each row in the
    # same order as in FIO_TABLE_ROW_SCHEMA.
    rows_to_insert = []
    for fioTableRow in fioTableRows:
      # Create a temporary list first for appending because tuples are immutable.
      row_to_be_inserted = []
      for field in FIO_TABLE_ROW_SCHEMA:
        row_to_be_inserted.append(str(getattr(fioTableRow, field)))
      rows_to_insert.append(tuple(row_to_be_inserted))

    # Now that the list of tuples is available, insert it
    # into the table.
    table = self._get_table_from_table_id(self.table_id)
    try:
      self._insert_rows_with_retry(table, rows_to_insert)
    except Exception as e:
      raise Exception(
          'Error inserting data to BigQuery table'
          f' {self.project_id}:{self.dataset_id}.{self.table_id}: {e}'
      )
