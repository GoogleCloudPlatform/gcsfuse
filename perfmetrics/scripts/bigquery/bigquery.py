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
configurations and metrics data in BigQuery.

Note:
  Make sure BigQuery API is enabled for the project
"""
import sys
from google.cloud import bigquery


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

  CONFIGURATION_TABLE_ID = 'experiment_configuration'
  FIO_TABLE_ID = 'read_write_fio_metrics'
  VM_TABLE_ID = 'read_write_vm_metrics'
  LS_TABLE_ID = 'list_metrics'

  def __init__(self, project_id, dataset_id, bq_client=None):
    if bq_client is None:
      self.client = bigquery.Client(project=project_id)
    self.project_id = project_id
    self.dataset_id = dataset_id

  @property
  def dataset_ref(self):
    """
      Gets the reference of the dataset

      Returns:
        google.cloud.bigquery.dataset.Dataset: The retrieved dataset object
    """
    return self.client.get_dataset(self.dataset_id)

  def _execute_query_and_check_for_error(self, query) -> QueryJob:
    """Executes the query in BigQuery and raises an exception if query
       execution could not be completed.

    Args:
      query (str): Query that will be executed in BigQuery.

    Raises:
      Exception: If query execution failed.
    """
    job = self.client.query(query)
    # Wait for query to be completed
    job.result()
    if job.errors:
      for error in job.errors:
        raise Exception(f"Error message: {error['message']}")
    return job

  def setup_dataset_and_tables(self):
    """
      Creates the dataset to store the tables and the experiment configuration table
      to store the configuration details and creates the list_metrics, read_write_fio_metrics
      and read_write_vm_metrics tables to store the metrics data if it doesn't already exist
      in the dataset.
    """
    # Create dataset if not exists
    dataset = bigquery.Dataset(f"{self.project_id}.{self.dataset_id}")
    self.client.create_dataset(dataset, exists_ok=True)

    # Query for creating experiment_configuration table if it does not exist
    query_create_table_experiment_configuration = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id INT64,
        configuration_name STRING,
        gcsfuse_flags STRING,
        branch STRING,
        end_date TIMESTAMP,
        PRIMARY KEY (configuration_id) NOT ENFORCED
      ) OPTIONS (description = 'Table for storing Job Configurations and respective VM instance name on which the job was run');
    """.format(self.project_id, self.dataset_id, ExperimentsGCSFuseBQ.CONFIGURATION_TABLE_ID)

    # Query for creating fio_metrics table
    query_create_table_fio_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id INT64, 
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
    """.format(self.project_id, self.dataset_id, ExperimentsGCSFuseBQ.FIO_TABLE_ID, self.dataset_id, ExperimentsGCSFuseBQ.CONFIGURATION_TABLE_ID)

    # Query for creating vm_metrics table
    query_create_table_vm_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id INT64, 
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
    """.format(self.project_id, self.dataset_id, ExperimentsGCSFuseBQ.VM_TABLE_ID, self.dataset_id, ExperimentsGCSFuseBQ.CONFIGURATION_TABLE_ID)

    # Query for creating ls_metrics table
    query_create_table_ls_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id INT64,
        start_time_build INT64,
        mount_type STRING, 
        command STRING,
        start_time FLOAT64, 
        end_time FLOAT64,
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
    """.format(self.project_id, self.dataset_id, ExperimentsGCSFuseBQ.LS_TABLE_ID, self.dataset_id, ExperimentsGCSFuseBQ.CONFIGURATION_TABLE_ID)

    self._execute_query_and_check_error(query_create_table_experiment_configuration)
    self._execute_query_and_check_error(query_create_table_fio_metrics)
    self._execute_query_and_check_error(query_create_table_vm_metrics)
    self._execute_query_and_check_error(query_create_table_ls_metrics)



