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
"""Python module for setting up the dataset and tables in BigQuery.

This python module creates the dataset and the tables that will store experiment
configurations and metrics data in BigQuery. It can also be used to upload data to the tables.

Note:
  Make sure BigQuery API is enabled for the project
"""

import uuid
import time
from google.cloud import bigquery
from google.cloud.bigquery.job import QueryJob
import constants

class ExperimentsGCSFuseBQ:
  """
    Class to create and interact with Bigquery dataset and tables for storing
    experiments configurations and their results.

    Attributes:
      project_id (str): The GCP project in which dataset and tables will be created
      dataset_id (str): The name of the dataset in the project that will store the tables
      bq_client (Optional[google.cloud.bigquery.client.Client]): The client for interacting with Bigquery.
                                                                 Default value is bigquery.Client(project=project_id).
  """

  def __init__(self, project_id, dataset_id, bq_client=None):
    if bq_client is None:
      self.client = bigquery.Client(project=project_id)
    else:
      self.client = bq_client
    self.project_id = project_id
    self.dataset_id = dataset_id

  @property
  def dataset_ref(self):
    """Gets the reference of the dataset

    Returns:
      google.cloud.bigquery.dataset.Dataset: The retrieved dataset object
    """
    return self.client.get_dataset(self.dataset_id)

  def _get_table_from_table_id(self, table_id):
    """Gets the table from BigQuery from table ID

    Args:
      table_id (str): String representing the ID or name of the table
    Returns:
      google.cloud.bigquery.table.Table: The table in BigQuery
    """
    table_ref = self.dataset_ref.table(table_id)
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

  def _check_if_config_valid(self, exp_config_id) -> bool:
    """Checks if exp_config_id exists in the experiment_configuration table.

    Args:
      exp_config_id (str): An id that uniquely identifies an experiment

    Returns:
      bool: Returns true if exp_config_id exists, false otherwise.
    """
    query_check_if_config_valid = """
      SELECT *
      FROM `{}.{}.{}`
      WHERE configuration_id = '{}'
    """.format(self.project_id, self.dataset_id, constants.CONFIGURATION_TABLE_ID, exp_config_id)

    job = self._execute_query(query_check_if_config_valid)
    row_count = job.result().total_rows
    if row_count:
      return True
    return False

  def _insert_rows(self, table, rows_to_insert, table_id = None, config_id = None, start_time_build = None):
    """Insert rows in table. If insertion of some nth row fails, delete (n-1) rows
    that were inserted before and raise an exception

    Args:
      table (str): Table in which rows are being inserted
      rows_to_insert (str): Rows to insert in the table
      table_id (str): ID of table to which results are being uploaded
      config_id (str): config_id of the experiment for which results are being uploaded
      start_time_build (timestamp): Start epoch time of the build

    Raises:
      Exception: If some row insertion failed.
    """
    result = self.client.insert_rows(table, rows_to_insert)
    if result:
      if config_id:
        query_check_if_row_exists = """
          DELETE FROM `{}.{}.{}`
          WHERE configuration_id = '{}'
          AND start_time_build = '{}'
        """.format(self.project_id, self.dataset_id, table_id, config_id, start_time_build)
        job = self._execute_query(query_check_if_row_exists)
      raise Exception(f'Error inserting data to BigQuery tables: {result}')

  def setup_dataset_and_tables(self):
    f"""
      Creates the dataset to store the tables and the experiment configuration table
      to store the configuration details and creates the {constants.LS_TABLE_ID},
      {constants.FIO_TABLE_ID}, {constants.VM_TABLE_ID} tables to store the metrics 
    """
    # Create dataset if not exists
    dataset = bigquery.Dataset(f"{self.project_id}.{self.dataset_id}")
    self.client.create_dataset(dataset, exists_ok=True)
    # Wait for the dataset to be created and ready to be referenced
    time.sleep(120)

    # Query for creating experiment_configuration table if it does not exist
    query_create_table_experiment_configuration = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id STRING,
        configuration_name STRING,
        gcsfuse_flags STRING,
        branch STRING,
        end_date TIMESTAMP,
        PRIMARY KEY (configuration_id) NOT ENFORCED
      ) OPTIONS (description = 'Table for storing Job Configurations and respective VM instance name on which the job was run');
    """.format(self.project_id, self.dataset_id, constants.CONFIGURATION_TABLE_ID)

    # Query for creating fio_metrics table
    query_create_table_fio_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id STRING, 
        start_time_build INT64,
        test_type STRING, 
        num_threads INT64, 
        file_size_kb INT64, 
        block_size_kb INT64,
        start_time INT64, 
        end_time INT64, 
        iops FLOAT64, 
        bandwidth_bytes_per_sec INT64, 
        io_bytes INT64, 
        min_latency FLOAT64, 
        max_latency FLOAT64, 
        mean_latency FLOAT64, 
        percentile_latency_20 FLOAT64, 
        percentile_latency_50 FLOAT64, 
        percentile_latency_90 FLOAT64, 
        percentile_latency_95 FLOAT64, 
        FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
      ) OPTIONS (description = 'Table for storing FIO metrics extracted from experiments.');
    """.format(self.project_id, self.dataset_id, constants.FIO_TABLE_ID, self.dataset_id, constants.CONFIGURATION_TABLE_ID)

    # Query for creating vm_metrics table
    query_create_table_vm_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id STRING, 
        start_time_build INT64,
        end_time INT64, 
        cpu_utilization_peak_percentage FLOAT64, 
        cpu_utilization_mean_percentage FLOAT64, 
        received_bytes_peak_per_sec FLOAT64, 
        received_bytes_mean_per_sec FLOAT64, 
        read_bytes_count INT64,
        ops_error_count INT64, 
        ops_mean_latency_sec FLOAT64, 
        sent_bytes_peak_per_sec FLOAT64, 
        sent_bytes_mean_per_sec FLOAT64, 
        sent_bytes_count INT64,
        iops FLOAT64, 
        ops_count_list_object INT64, 
        ops_count_create_object INT64, 
        ops_count_stat_object INT64, 
        ops_count_new_reader INT64, 
        FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
      ) OPTIONS (description = 'Table for storing VM metrics extracted from experiments.');
    """.format(self.project_id, self.dataset_id, constants.VM_TABLE_ID, self.dataset_id, constants.CONFIGURATION_TABLE_ID)

    # Query for creating ls_metrics table
    query_create_table_ls_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id STRING,
        start_time_build INT64,
        mount_type STRING,
        start_time FLOAT64, 
        end_time FLOAT64, 
        command STRING,
        num_files INT64, 
        num_samples INT64, 
        min_latency_msec FLOAT64,
        max_latency_msec FLOAT64,
        mean_latency_msec FLOAT64, 
        median_latency_msec FLOAT64, 
        standard_dev_latency_msec FLOAT64, 
        percentile_latency_20 FLOAT64, 
        percentile_latency_50 FLOAT64, 
        percentile_latency_90 FLOAT64, 
        percentile_latency_95 FLOAT64, 
        cpu_utilization_peak_percentage FLOAT64, 
        cpu_utilization_mean_percentage FLOAT64,            
        received_bytes_peak_per_sec FLOAT64, 
        received_bytes_mean_per_sec FLOAT64,
        FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
      ) OPTIONS (description = 'Table for storing GCSFUSE metrics extracted from list experiments.');

    """.format(self.project_id, self.dataset_id, constants.LS_TABLE_ID, self.dataset_id, constants.CONFIGURATION_TABLE_ID)

    self._execute_query(query_create_table_experiment_configuration)
    self._execute_query(query_create_table_fio_metrics)
    self._execute_query(query_create_table_vm_metrics)
    self._execute_query(query_create_table_ls_metrics)

  def get_experiment_configuration_id(self, gcsfuse_flags, branch, end_date, config_name) -> str:

    """Gets the configuration ID of the experiment from experiment details
       If experiment configuration exists: Check if end date needs update and
                                           then return the configuration ID
       Else: Insert new experiment configuration and return the configuration ID

    Args:
      gcsfuse_flags (str): Set of flags the gcsfuse flags used for experiment.
      branch (str): GCSFuse repo branch used for building GCSFuse.
      end_date (timestamp): Date till when experiments of this configuration are run.
                            Format: 'YYYY-MM-DD HH:MM:SS'
      config_name (str): Name of the experiment configuration.

    Returns:
      str: Configuration ID of the experiment
    """
    # Check if the experiment configuration is already present in table
    query_check_config_exists = """
      SELECT configuration_id
      FROM `{}.{}.{}`
      WHERE gcsfuse_flags = '{}'
      AND branch = '{}'
      AND configuration_name = '{}'
    """.format(self.project_id, self.dataset_id, constants.CONFIGURATION_TABLE_ID, gcsfuse_flags, branch, config_name)

    job = self._execute_query(query_check_config_exists)
    result_count = job.result().total_rows

    # If more than 1 result -> duplicate experiment configuration present -> throw error
    if result_count > 1:
      raise Exception("Duplicate experiment configurations exist. Data corrupted")

    # If result empty, then experiment configuration not present -> insert new experiment configuration -> return configuration ID
    elif result_count == 0:
      table = self._get_table_from_table_id(constants.CONFIGURATION_TABLE_ID)
      uuid_str = str(uuid.uuid4())
      rows_to_insert = [(uuid_str, config_name, gcsfuse_flags, branch, end_date)]
      self._insert_rows(table, rows_to_insert)
      return uuid_str

    # If exactly one result -> update end date -> return configuration ID
    else:
      config_id = list(job)[0].get('configuration_id')
      query_update_end_date = """
        UPDATE `{}.{}.{}`
        SET end_date = '{}'
        WHERE configuration_id = '{}'
        """.format(self.project_id, self.dataset_id, constants.CONFIGURATION_TABLE_ID, end_date, config_id)
      self._execute_query(query_update_end_date)
      return config_id

  def upload_metrics_to_table(self, table_id, config_id, start_time_build, metrics_data):

    """Uploads metrics_data to the table corresponding to 'table_name'.

    Args:
      table_id (str): ID of table to which results are being uploaded
      config_id (str): config_id of the experiment for which results are being uploaded
      start_time_build (int): Start epoch time of the build
      metrics_data (list): A 2D list containing the experiment results
                           For example: metrics data for fio jobs will look like:
                           [['read', 40, 256, 1687928088, 1687928159, 27032.61141, 443600529,
                           26647527424, 0.000126831, 0.323205657, 0.09454765585, 0.08650752,
                           0.0917504, 0.106430464, 0.113770496],
                           ['write', 40, 256, 1687928278, 1687928364, 87.361631, 1979988,
                           149176320, 0.032581924, 45.73434076, 20.26386098, 13.75731712,
                           17.11276032, 17.11276032, 17.11276032]]
    """

    # Check if the configuration ID of the experiment is valid
    config_valid = self._check_if_config_valid(config_id)

    if not config_valid:
      raise Exception("Invalid configuration ID")

    table = self._get_table_from_table_id(table_id)

    rows_to_insert = []
    for row in metrics_data:
      rows_to_insert = rows_to_insert + [(config_id, start_time_build) + tuple(row)]

    self._insert_rows(table, rows_to_insert, table_id, config_id, start_time_build)
