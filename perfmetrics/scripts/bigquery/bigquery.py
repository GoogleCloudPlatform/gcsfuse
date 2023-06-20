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
"""Python module for setting up tables in BigQuery.

This python module creates tables that will store experiment configuration and metrics data in BigQuery.

Note:
  Make sure BigQuery API is enabled for the project
"""
import sys
from google.cloud import bigquery

class BigQuery():

  configuration_table_id = 'experiment_configuration'
  fio_table_id = 'read_write_fio_metrics'
  vm_table_id = 'read_write_vm_metrics'
  ls_table_id = 'list_metrics'
  def __int__(self, project_id, dataset_id, bq_client=None):
    if bq_client is None:
      self.client = bigquery.Client(project=project_id)
    self.project_id = project_id
    self.dataset_id = dataset_id
    self.dataset_ref = self.client.get_dataset(dataset_id)

  def _execute_query_and_check_error(self, query):
    """Executes the query in BigQuery.

    Args:
      query: Query that will be executed in BigQuery.

    Raises:
      Aborts the program if error is encountered while executing th query.
    """
    job = self.client.query(query)
    job.results()
    if job.errors:
      for error in job.errors:
        print(f"Error message: {error['message']}")
      sys.exit(1)
    return

  def setup_bigquery(self):

    """Creates the experiment configuration table to store the configuration details and
       creates the list_metrics, read_write_fio_metrics and read_write_vm_metrics tables
       to store the metrics data if they don't already exist in the dataset
    """
    # Query for creating performance_metrics dataset
    query_create_dataset_performance_metrics= """
      CREATE SCHEMA IF NOT EXISTS performance_metrics 
      OPTIONS (description = 'Dataset for storing the experiment configurations tables and metric data tables'
    )"""

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
    """.format(self.project_id, self.dataset_id, BigQuery.configuration_table_id)

    # Query for creating fio_metrics table
    query_create_table_fio_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id INT64, 
        start_time_build TIMESTAMP,
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
    """.format(self.project_id, self.dataset_id, BigQuery.fio_table_id, self.dataset_id, BigQuery.configuration_table_id)

    # Query for creating vm_metrics table
    query_create_table_vm_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id INT64, 
        start_time_build TIMESTAMP,
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
        memory_utilization_ram FLOAT64,
        memory_utilization_disk_tempdir FLOAT64,
        iops FLOAT64, 
        ops_count_list_object INT64, 
        ops_count_create_object INT64, 
        ops_count_stat_object INT64, 
        ops_count_new_reader INT64, 
        FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
      ) OPTIONS (description = 'Table for storing VM metrics extracted from experiments.');
    """.format(self.project_id, self.dataset_id, BigQuery.vm_table_id, self.dataset_id, BigQuery.configuration_table_id)

    # Query for creating ls_metrics table
    query_create_table_ls_metrics = """
      CREATE TABLE IF NOT EXISTS {}.{}.{}(
        configuration_id INT64,
        start_time_build TIMESTAMP,
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
        memory_utilization_ram FLOAT64, 
        FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
      ) OPTIONS (description = 'Table for storing GCSFUSE metrics extracted from list experiments.');
    """.format(self.project_id, self.dataset_id, BigQuery.ls_table_id, self.dataset_id, BigQuery.configuration_table_id)

    transaction = self.client.transaction()

    try:
      # Begin the transaction
      transaction.begin()
      self.client.query(query_create_dataset_performance_metrics, transaction=transaction).result()
      self.client.query(query_create_table_experiment_configuration, transaction=transaction).result()
      self.client.query(query_create_table_fio_metrics, transaction=transaction).result()
      self.client.query(query_create_table_vm_metrics, transaction=transaction).result()
      self.client.query(query_create_table_ls_metrics, transaction=transaction).result()
      # Commit the transaction
      transaction.commit()

    except Exception as e:
      # Rollback the transaction
      transaction.rollback()
      print(f"Transaction failed: {e}")
      sys.exit(1)

    # self._execute_query_and_check_error(query_create_dataset_performance_metrics)
    # self._execute_query_and_check_error(query_create_table_experiment_configuration)
    # self._execute_query_and_check_error(query_create_table_fio_metrics)
    # self._execute_query_and_check_error(query_create_table_vm_metrics)
    # self._execute_query_and_check_error(query_create_table_ls_metrics)

